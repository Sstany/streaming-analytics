package controller

import (
	"corgiAnalytics/orchestrator/internal/db"
	"corgiAnalytics/orchestrator/internal/entity"
	"corgiAnalytics/orchestrator/internal/orchestrator"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Server struct {
	host         string
	port         string
	dbClient     db.PostgresClient
	orchestrator *orchestrator.Service
	logger       *zap.Logger
}

func NewServer(host string, port string, dbClient db.PostgresClient, orchestrator *orchestrator.Service, logger *zap.Logger) *Server {
	s := &Server{
		host:         host,
		port:         port,
		dbClient:     dbClient,
		orchestrator: orchestrator,
		logger:       logger,
	}

	return s
}

func (r *Server) Start() {
	api := r.newAPI()

	r.orchestrator.Start()

	api.Run(r.host + ":" + r.port)

	r.orchestrator.Stop()

	r.orchestrator.Wait()
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
	var reg entity.Job

	if err := ctx.Bind(&reg); err != nil {
		r.logger.Error("bind failed", zap.Any("email", reg.URL), zap.Error(err))
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	reg.Status = entity.Created
	reg.ID = uuid.NewString()

	r.orchestrator.Send(reg)
	ctx.Status(http.StatusOK)
}

func (r *Server) updateJob(ctx *gin.Context) {

}
