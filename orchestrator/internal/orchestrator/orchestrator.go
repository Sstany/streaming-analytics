package orchestrator

import (
	"context"
	"corgiAnalytics/orchestrator/internal/db"
	"corgiAnalytics/orchestrator/internal/entity"
	"corgiAnalytics/orchestrator/pkg"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	dbClient db.PostgresClient
	jobChan  chan entity.Job
	producer sarama.SyncProducer
	brokers  []string
	wg       *sync.WaitGroup
	logger   *zap.Logger
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

func New(dbClient db.PostgresClient, logger *zap.Logger, brokers []string) *Service {
	return &Service{
		dbClient: dbClient,
		jobChan:  make(chan entity.Job),
		wg:       new(sync.WaitGroup),
		brokers:  brokers,
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

	producer, err := ConnectProducer(r.brokers)
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

	r.wg.Add(1)

	go func() {
		defer r.wg.Done()

		consumer, err := sarama.NewConsumer(r.brokers, nil)
		if err != nil {
			log.Fatalf("Failed to create consumer: %v", err)
		}

		defer consumer.Close()

		partConsumer, err := consumer.ConsumePartition("status", 0, sarama.OffsetOldest)
		if err != nil {
			r.logger.Error("create consumer for predict", zap.Error(err))
			return
		}

		defer partConsumer.Close()

		for status := range partConsumer.Messages() {
			var job entity.Job

			job.ID = string(status.Key)
			job.Status = entity.Status(status.Value)
			job.Timestamp = time.Now().UnixMilli()

			if err := r.dbClient.CreateJob(context.Background(), &job); err != nil {
				r.logger.Error("create job failed", zap.Error(err))
			}
		}
	}()

	r.logger.Info("start orchestrator")

	return nil
}
