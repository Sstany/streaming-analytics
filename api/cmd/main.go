package main

import (
	"corgiAnalytics/api/internal/controller/server"
	"os"

	"go.uber.org/zap"
)

func main() {
	host := os.Getenv(server.EnvHost)
	port := os.Getenv(server.EnvPort)

	// server := server.

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
}
