package main

import (
	"context"

	"corgiAnalytics/runner/internal/entity"
	"corgiAnalytics/runner/internal/runner"
	"os"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	// host := os.Getenv(controller.EnvHost)
	// port := os.Getenv(controller.EnvPort)

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv(entity.EnvRedis),
		Password: os.Getenv(entity.EnvRedisPassword),
		DB:       1,
	})

	status := client.Ping(context.Background())
	if err := status.Err(); err != nil {
		panic(err)
	}

	brokers := []string{os.Getenv(entity.EnvKafka)}
	runPath := os.Getenv(entity.EnvRunPath)
	runnerService := runner.New(logger, runPath, brokers, client)
	runnerService.Start()
	runnerService.Wait()

	// controller.NewServer(host, port, runnerService, logger).Start()
}
