package runner

import (
	"corgiAnalytics/orchestrator/pkg"
	"corgiAnalytics/runner/internal/db"
	"corgiAnalytics/runner/internal/entity"
	"corgiAnalytics/runner/internal/framer"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gocv.io/x/gocv"
)

const (
	KafkaServer1 = "localhost:9091"
)

type Service struct {
	brokers      []string
	runPath      string
	consumer     sarama.Consumer
	partConsumer sarama.PartitionConsumer
	producer     sarama.SyncProducer
	wg           *sync.WaitGroup
	jobChan      chan entity.Job
	resultsChan  chan entity.Result
	redis        *redis.Client
	framer       framer.Framer
	logger       *zap.Logger
}

func New(logger *zap.Logger, runPath string, brokers []string, redis *redis.Client) *Service {
	return &Service{
		runPath:     runPath,
		wg:          &sync.WaitGroup{},
		brokers:     brokers,
		jobChan:     make(chan entity.Job),
		resultsChan: make(chan entity.Result),
		redis:       redis,
		framer:      framer.NewVideoFramer(),
		logger:      logger.Named("runner"),
	}
}

func ConnectProducer(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 2

	return sarama.NewSyncProducer(brokers, config)
}

func (r *Service) Start() {
	consumer, err := sarama.NewConsumer(r.brokers, nil)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	r.consumer = consumer
	r.partConsumer, err = consumer.ConsumePartition(pkg.TopicJobs, 0, sarama.OffsetNewest)

	if err != nil {
		log.Fatalf("Failed to consume partition: %v", err)
	}
	r.logger.Info("started service")

	brokers := []string{KafkaServer1}
	producer, err := ConnectProducer(brokers)
	if err != nil {
		panic(fmt.Errorf("create producer: %w", err))
	}

	r.producer = producer

	count, err := strconv.Atoi(os.Getenv(entity.EnvCount))
	if err != nil {
		r.logger.Fatal("worker count: %w", zap.Error(err))
	}

	for range count {
		r.wg.Add(1)
		go r.startFramer()
		go r.startSaver()
	}

	r.wg.Add(1)
	go r.consume()
}

func (r *Service) Stop() {
	r.consumer.Close()
	r.partConsumer.Close()
	r.producer.Close()
	close(r.jobChan)
}

func (r *Service) Wait() {
	r.wg.Wait()
}

func (r *Service) consume() {
	defer r.wg.Done()
	for msg := range r.partConsumer.Messages() {
		var job entity.Job
		if err := json.Unmarshal(msg.Value, &job); err != nil {
			r.logger.Error("failed to unmarshal job", zap.Error(err))
		}
		r.logger.Info("get job: %w", zap.Any("job:", job))

		if _, _, err := r.producer.SendMessage(&sarama.ProducerMessage{
			Topic: pkg.TopicJobs,
			Key:   sarama.StringEncoder(job.ID),
			Value: sarama.StringEncoder(entity.Startup),
		}); err != nil {
			r.logger.Error("send startup msg failed", zap.String("jobID", job.ID), zap.Error(err))
		}

		r.jobChan <- job
	}
}

func (r *Service) startFramer() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for j := range r.jobChan {
			if err := os.Mkdir(r.runPath+j.ID, 0777); err != nil {
				if !errors.Is(err, os.ErrExist) {
					r.logger.Error("create dir for job", zap.Error(err))
				}
			}

			if err := r.framer.Start(j); err != nil {
				r.logger.Error("start framer", zap.Error(err))
				continue
			}

			if _, _, err := r.producer.SendMessage(&sarama.ProducerMessage{
				Topic: pkg.TopicJobs,
				Key:   sarama.StringEncoder(j.ID),
				Value: sarama.StringEncoder(entity.Active),
			}); err != nil {
				r.logger.Error("send finished msg failed", zap.String("jobID", j.ID), zap.Error(err))
			}

			for {
				fr, err := r.framer.Next()
				if err != nil {
					r.logger.Error("framer: $w", zap.Error(err))
					r.framer.Stop()
					break
				}

				if fr == nil {
					r.logger.Info("frame is nil abort job processing")
					break
				}

				if err := db.PostFrame(r.redis, fr.ID, fr); err != nil {
					r.logger.Error("post frame meta", zap.Error(err))
				}

				r.logger.Debug("frame", zap.String("frame id", fr.ID))

				frame, err := json.Marshal(&fr.FrameMeta)
				if err != nil {
					r.logger.Error("marshal: %w", zap.Error(err))
				}

				msg := &sarama.ProducerMessage{
					Topic: pkg.TopicInference,
					Value: sarama.StringEncoder(frame),
				}

				_, _, err = r.producer.SendMessage(msg)
				if err != nil {
					r.logger.Error("produce frame: %w", zap.Error(err))
					r.framer.Stop()
					break
				}
			}

			r.logger.Info("finished processing job", zap.String("job", j.ID))

			if _, _, err := r.producer.SendMessage(&sarama.ProducerMessage{
				Topic: "status",
				Key:   sarama.StringEncoder(j.ID),
				Value: sarama.StringEncoder(entity.Inactive),
			}); err != nil {
				r.logger.Error("send finished msg failed", zap.String("jobID", j.ID), zap.Error(err))
			}

			r.framer.Stop()
		}
	}()
}

func (r *Service) startSaver() {
	defer r.wg.Done()

	consumer, err := r.consumer.ConsumePartition("predict", 0, sarama.OffsetOldest)
	if err != nil {
		r.logger.Error("create consumer for predict", zap.Error(err))
		return
	}

	for result := range consumer.Messages() {
		var predictResult entity.Result

		if err := json.Unmarshal(result.Value, &predictResult); err != nil {
			r.logger.Error("unmarshal failed", zap.Error(err))
			continue
		}

		frame, err := db.ReadFrame(r.redis, predictResult.ID)
		if err != nil {
			r.logger.Error("read frame for result faield", zap.Error(err))
			continue
		}

		if frame == nil {
			continue
		}

		img, err := gocv.NewMatFromBytes(
			int(frame.Rows),
			int(frame.Cols),
			gocv.MatType(frame.FrameType),
			frame.Payload,
		)
		if err != nil {
			img.Close()
			return
		}

		if img.Empty() {
			img.Close()
			continue
		}

		for _, pred := range predictResult.Predicts {
			gocv.Rectangle(
				&img,
				pred.Rectangle,
				color.RGBA{0, 255, 0, 0},
				2,
			)

			gocv.PutText(
				&img,
				pred.Class,
				image.Point{int(pred.Rectangle.Min.X), int(pred.Rectangle.Min.Y) - 10},
				gocv.FontHersheyPlain,
				0.6,
				color.RGBA{0, 255, 0, 0},
				1,
			)
		}

		fp := r.runPath + frame.JobID + "/" + strconv.Itoa(int(frame.Sequence)) + frame.ID + "-" + ".jpg"
		ok := gocv.IMWrite(fp, img)
		r.logger.Debug("write image to file", zap.String("filepath", fp), zap.Bool("ok", ok))
		img.Close()
	}
}
