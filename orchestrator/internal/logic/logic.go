package logic

import "go.uber.org/zap"

type Service struct {
	dbClient db.Client
	logger   *zap.Logger

	pb.UnimplementedOrchestratorServer
}
