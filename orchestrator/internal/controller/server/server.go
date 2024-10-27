package server

import (
	"corgiAnalytics/orchestrator/internal/db"

	"go.uber.org/zap"
)

type Server struct {
	host     string
	port     int
	dbClient db.PostgresClient
	logger   *zap.Logger
}
