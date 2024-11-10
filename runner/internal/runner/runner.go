package runner

import (
	"corgiAnalytics/orchestrator/pkg"
	"corgiAnalytics/runner/internal/db"
	"corgiAnalytics/runner/internal/entity"
	"corgiAnalytics/runner/internal/framer"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	KafkaServer1 = "localhost:9091"
)

type Service struct {
	consumer     sarama.Consumer
	partConsumer sarama.PartitionConsumer
	wg           *sync.WaitGroup
	brokers      []string
	jobChan      chan entity.Job
	producer     sarama.SyncProducer
	redis        *redis.Client
	framer       framer.Framer
	logger       *zap.Logger
}

func New(logger *zap.Logger, brokers []string, redis *redis.Client) *Service {
	return &Service{
		wg:      &sync.WaitGroup{},
		brokers: brokers,
		jobChan: make(chan entity.Job),
		redis:   redis,
		framer:  framer.NewVideoFramer(),
		logger:  logger.Named("runner"),
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
		log.Fatalf("create producer: $w", zap.Error(err))
	}

	r.producer = producer

	count, err := strconv.Atoi(os.Getenv(entity.EnvCount))
	if err != nil {
		r.logger.Fatal("worker count: %w", zap.Error(err))
	}

	for range count {
		r.wg.Add(1)
		go r.startFramer()
	}

	r.wg.Add(1)
	go r.consume()
}

func (r *Service) Stop() {
	defer r.consumer.Close()

	r.partConsumer.Close()
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

		r.jobChan <- job
	}
}

func (r *Service) startFramer() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for j := range r.jobChan {
			r.framer.Start(j)
			for {
				fr, err := r.framer.Next()
				if err != nil {
					r.logger.Error("framer: $w", zap.Error(err))
					r.framer.Stop()
					break
				}

				db.PostFrame(r.redis, fr.Id, fr)
				r.logger.Info("frame", zap.String("frame id", fr.Id))

				frame, err := json.Marshal(fr)
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

			r.logger.Info("send job: %w", zap.Any("job:", j))

		}

	}()

}
func (r *Service) produce(topic string, message *entity.Job) error {
	res, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(res),
	}

	_, _, err = r.producer.SendMessage(msg)
	if err != nil {
		return err
	}

	return nil
}
