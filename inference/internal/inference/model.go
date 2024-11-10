package inference

import (
	"corgiAnalytics/inference/internal/db"
	"corgiAnalytics/inference/internal/entity"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"sync"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gocv.io/x/gocv"
)

const (
	ratio   = 0.003921568627
	swapRGB = false
)

type Inference struct {
	net         *gocv.Net
	params      gocv.ImageToBlobParams
	outputNames []string

	wg     *sync.WaitGroup
	logger *zap.Logger

	brokers      []string
	consumer     sarama.Consumer
	partConsumer sarama.PartitionConsumer
	producer     sarama.SyncProducer
	framesTopic  string
	predictTopic string

	redis *redis.Client

	workersCount int
}

func ConnectProducer(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 2

	return sarama.NewSyncProducer(brokers, config)
}

func New(client *redis.Client, logger *zap.Logger, modelPath string, workers int, brokers []string) *Inference {
	logger = logger.Named("inference")

	net := gocv.ReadNetFromONNX(modelPath)
	net.SetPreferableBackend(gocv.NetBackendOpenCV)
	net.SetPreferableTarget(gocv.NetTargetCPU)

	outputNames := getOutputNames(&net)
	if len(outputNames) == 0 {
		return nil
	}

	var (
		mean     = gocv.NewScalar(0, 0, 0, 0)
		padValue = gocv.NewScalar(144.0, 0, 0, 0)
	)

	params := gocv.NewImageToBlobParams(
		ratio,
		image.Pt(640, 640),
		mean,
		swapRGB,
		gocv.MatTypeCV32F,
		gocv.DataLayoutNCHW,
		gocv.PaddingModeLetterbox,
		padValue,
	)

	consumer, err := sarama.NewConsumer(brokers, nil)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}

	partConsumer, err := consumer.ConsumePartition("inference", 0, sarama.OffsetOldest)
	if err != nil {
		log.Fatalf("Failed to consume partition: %v", err)
	}
	logger.Info("started service")

	producer, err := ConnectProducer(brokers)
	if err != nil {
		panic(fmt.Errorf("create producer: %w", err))
	}

	return &Inference{
		net:          &net,
		params:       params,
		outputNames:  outputNames,
		wg:           new(sync.WaitGroup),
		logger:       logger,
		workersCount: workers,
		consumer:     consumer,
		partConsumer: partConsumer,
		producer:     producer,
		redis:        client,
		framesTopic:  "inference",
		predictTopic: "predict",
	}
}

func (r *Inference) Start() {
	ch := make(chan *entity.Frame)

	for i := 0; i < r.workersCount; i++ {
		r.wg.Add(1)
		go r.worker(ch)
	}

	for msg := range r.partConsumer.Messages() {
		frameMeta := new(entity.FrameMeta)

		if err := json.Unmarshal(msg.Value, &frameMeta); err != nil {
			break
		}

		frame, err := db.ReadFrame(r.redis, frameMeta.ID)
		if err != nil {
			r.logger.Error("read frame from redis", zap.Error(err))
			continue
		}

		ch <- frame
	}

	close(ch)

	r.wg.Wait()
}

func (r *Inference) worker(ch chan *entity.Frame) {
	defer r.wg.Done()

	for frame := range ch {
		newImg, err := gocv.NewMatFromBytes(
			int(frame.Rows),
			int(frame.Cols),
			gocv.MatType(frame.FrameType),
			frame.Payload,
		)
		if err != nil {
			newImg.Close()
			return
		}

		if newImg.Empty() {
			newImg.Close()
			continue
		}

		predicts := r.detect(&newImg)

		// We send only frames with prediction
		if len(predicts) == 0 {
			continue
		}

		frameResult := entity.Result{
			ID:       frame.ID,
			Predicts: predicts,
		}

		res, err := json.Marshal(frameResult)
		if err != nil {
			r.logger.Error("marshal result", zap.Error(err))
		}

		msg := &sarama.ProducerMessage{
			Topic: r.predictTopic,
			Value: sarama.StringEncoder(res),
		}

		_, _, err = r.producer.SendMessage(msg)
		if err != nil {
			r.logger.Error("send msg", zap.Error(err))
		}

		newImg.Close()
	}
}

func (r *Inference) detect(src *gocv.Mat) []entity.Predict {
	blob := gocv.BlobFromImageWithParams(*src, r.params)
	defer blob.Close()

	// feed the blob into the detector
	r.net.SetInput(blob, "")

	// run a forward pass thru the network
	probs := r.net.ForwardLayers(r.outputNames)
	defer func() {
		for _, prob := range probs {
			prob.Close()
		}
	}()

	boxes, confidences, classIds := performDetection(probs)
	if len(boxes) == 0 {
		r.logger.Error("No classes detected")
		return nil
	}

	iboxes := r.params.BlobRectsToImageRects(boxes, image.Pt(src.Cols(), src.Rows()))
	indices := gocv.NMSBoxes(iboxes, confidences, scoreThreshold, nmsThreshold)

	return makeResult(iboxes, classes, classIds, indices)
}

func makeResult(boxes []image.Rectangle, classes []string, classIds []int, indices []int) []entity.Predict {
	var predicts []entity.Predict

	for _, idx := range indices {
		if idx == 0 {
			continue
		}

		predicts = append(predicts, entity.Predict{
			Class:     classes[classIds[idx]],
			Rectangle: boxes[idx],
		})
	}

	return predicts
}

func performDetection(outs []gocv.Mat) ([]image.Rectangle, []float32, []int) {
	var classIds []int
	var confidences []float32
	var boxes []image.Rectangle

	// needed for yolov8
	gocv.TransposeND(outs[0], []int{0, 2, 1}, &outs[0])

	for _, out := range outs {
		out = out.Reshape(1, out.Size()[1])

		for i := 0; i < out.Rows(); i++ {
			cols := out.Cols()
			scoresCol := out.RowRange(i, i+1)

			scores := scoresCol.ColRange(4, cols)
			_, confidence, _, classIDPoint := gocv.MinMaxLoc(scores)

			if confidence > 0.5 {
				centerX := out.GetFloatAt(i, cols)
				centerY := out.GetFloatAt(i, cols+1)
				width := out.GetFloatAt(i, cols+2)
				height := out.GetFloatAt(i, cols+3)

				left := centerX - width/2
				top := centerY - height/2
				right := centerX + width/2
				bottom := centerY + height/2
				classIds = append(classIds, classIDPoint.X)
				confidences = append(confidences, float32(confidence))

				boxes = append(boxes, image.Rect(int(left), int(top), int(right), int(bottom)))
			}
		}
	}

	return boxes, confidences, classIds
}

func getOutputNames(net *gocv.Net) []string {
	var outputLayers []string
	for _, i := range net.GetUnconnectedOutLayers() {
		layer := net.GetLayer(i)
		layerName := layer.GetName()
		if layerName != "_input" {
			outputLayers = append(outputLayers, layerName)
		}
	}

	return outputLayers
}
