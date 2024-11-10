package controller

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	host   string
	port   string
	logger *zap.Logger
}

func NewServer(host string, port string, logger *zap.Logger) *Server {
	return &Server{
		host:   host,
		port:   port,
		logger: logger,
	}
}

func (r *Server) Start() {
	api := r.newAPI()

	api.Run(r.host, r.port)
}

func (r *Server) newAPI() *gin.Engine {
	eng := gin.New()

	apiV1 := eng.Group("/v1")
	apiV1.GET("/jobs/:id", r.getJob)
	apiV1.POST("/jobs", r.createJob)
	apiV1.POST("/jobs/:id", r.updateJob)

	return eng
}

func (r *Server) getJob(ctx *gin.Context) {

}

func (r *Server) createJob(ctx *gin.Context) {

}

func (r *Server) updateJob(ctx *gin.Context) {

}
