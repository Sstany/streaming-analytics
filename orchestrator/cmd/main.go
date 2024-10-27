package main

import (
	"context"
	"corgiAnalytics/orchestrator/internal/controller/server"
	"corgiAnalytics/orchestrator/internal/db"
	"os"

	"go.uber.org/zap"
)

func main() {
	host := os.Getenv(server.EnvHost)
	port := os.Getenv(server.EnvPort)

	postgresClient := db.NewPostgresClient(context.Background(), os.Getenv(server.EnvPostgres))
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
}
