package orchestrator

import (
	"context"
	"corgiAnalytics/orchestrator/internal/db"
	"corgiAnalytics/orchestrator/internal/entity"
	"corgiAnalytics/orchestrator/pkg"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	KafkaServer1 = "localhost:9091"
)

type Service struct {
	dbClient db.PostgresClient
	jobChan  chan entity.Job
	producer sarama.SyncProducer
	wg       *sync.WaitGroup
	logger   *zap.Logger
}

type Message struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func ConnectProducer(brokers []string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 2

	return sarama.NewSyncProducer(brokers, config)
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

func New(dbClient db.PostgresClient, logger *zap.Logger) *Service {
	return &Service{
		dbClient: dbClient,
		jobChan:  make(chan entity.Job),
		wg:       new(sync.WaitGroup),
		logger:   logger.Named("orchestrator"),
	}
}

func (r *Service) Send(job entity.Job) {
	r.jobChan <- job
}

func (r *Service) Wait() {
	r.wg.Wait()
}

func (r *Service) Stop() {
	defer r.producer.Close()

	close(r.jobChan)

}
func (r *Service) Start() error {
	job := &entity.Job{
		ID:     uuid.NewString(),
		Status: entity.Init,
	}

	if err := r.dbClient.CreateJob(context.Background(), job); err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	brokers := []string{KafkaServer1}
	producer, err := ConnectProducer(brokers)
	if err != nil {
		return err
	}

	r.producer = producer

	r.wg.Add(1)

	go func() {
		defer r.wg.Done()
		for j := range r.jobChan {
			if err := r.produce(pkg.TopicJobs, &j); err != nil {
				r.logger.Error("produce job failed", zap.String("jobID", j.ID), zap.Error(err))
				return
			}
			r.logger.Info("send job: %w", zap.Any("job:", j))

		}

	}()

	r.logger.Info("start orchestrator")

	return nil
}
