package main

import (
	"context"
	"corgiAnalytics/runner/internal/controller"
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
		Addr:     os.Getenv(controller.EnvRedis),
		Password: os.Getenv(controller.EnvRedisPassword),
		DB:       1,
	})

	status := client.Ping(context.Background())
	if err := status.Err(); err != nil {
		panic(err)
	}

	brokers := []string{os.Getenv(controller.EnvKafka)}
	runnerService := runner.New(logger, brokers)
	runnerService.Start()
	runnerService.Wait()

	// controller.NewServer(host, port, runnerService, logger).Start()
}
