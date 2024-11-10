package runner

import (
	"corgiAnalytics/orchestrator/pkg"
	"corgiAnalytics/runner/internal/controller/entity"
	"encoding/json"
	"log"
	"sync"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type Service struct {
	consumer     sarama.PartitionConsumer
	partConsumer sarama.PartitionConsumer
	wg           *sync.WaitGroup
	brokers      []string
	logger       *zap.Logger
}

func New(logger *zap.Logger, brokers []string) *Service {
	return &Service{
		wg:      &sync.WaitGroup{},
		brokers: brokers,
		logger:  logger.Named("runner"),
	}
}

func (r *Service) Start() {
	consumer, err := sarama.NewConsumer(r.brokers, nil)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}

	r.partConsumer, err = consumer.ConsumePartition(pkg.TopicJobs, 0, sarama.OffsetNewest)

	if err != nil {
		log.Fatalf("Failed to consume partition: %v", err)
	}
	r.logger.Info("started service")

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
	}
}
