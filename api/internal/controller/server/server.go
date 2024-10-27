package server

import "go.uber.org/zap"

type Server struct {
	host   string
	port   string
	logger *zap.Logger
}

func NewServer(host string, port string) *Server {
	return &Server{
		host: host,
		port: port,
	}
}
