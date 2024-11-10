package main

import (
	"context"

	"corgiAnalytics/orchestrator/internal/controller"
	"corgiAnalytics/orchestrator/internal/db"

	"corgiAnalytics/orchestrator/internal/orchestrator"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	host := os.Getenv(controller.EnvHost)
	port := os.Getenv(controller.EnvPort)

	logger, err := zap.NewProduction(zap.AddStacktrace(zapcore.ErrorLevel), zap.AddCaller())
	if err != nil {
		panic(err)
	}

	postgresClient := db.NewPostgresClient(context.Background(), os.Getenv(controller.EnvPostgres), logger)

	orchestratorService := orchestrator.New(*postgresClient, logger)
	controller.NewServer(host, port, *postgresClient, orchestratorService, logger).Start()
}
