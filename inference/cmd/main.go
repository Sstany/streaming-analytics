package main

import (
	"context"
	"corgiAnalytics/inference/config"
	"corgiAnalytics/inference/internal/entity"
	"corgiAnalytics/inference/internal/inference"
	"os"
	"runtime/debug"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const workers = 1

func main() {
	debug.SetGCPercent(10)

	modelPath := os.Getenv(config.EnvModelPath)

	logger, err := zap.NewProduction(zap.AddStacktrace(zapcore.ErrorLevel), zap.AddCaller())
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

	brokers := []string{os.Getenv("KAFKA_ADDRESS")}

	inference.New(client, logger, modelPath, workers, brokers).Start()
}
