package main

import (
	"corgiAnalytics/api/internal/controller"

	"os"

	"go.uber.org/zap"
)

func main() {
	host := os.Getenv(controller.EnvHost)
	port := os.Getenv(controller.EnvPort)

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	controller.NewServer(host, port, logger).Start()
}
