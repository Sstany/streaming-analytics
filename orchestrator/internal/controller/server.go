package controller

import (
	"corgiAnalytics/orchestrator/internal/db"
	"corgiAnalytics/orchestrator/internal/entity"
	"corgiAnalytics/orchestrator/internal/orchestrator"
	"database/sql"
	"errors"
	"net/http"
	"time"

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
	jobID := ctx.Param("id")

	job, err := r.dbClient.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		r.logger.Error("get job failed", zap.String("jobID", jobID), zap.Error(err))
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func (r *Server) createJob(ctx *gin.Context) {
	var reg entity.Job

	if err := ctx.Bind(&reg); err != nil {
		r.logger.Error("bind failed", zap.Any("email", reg.URL), zap.Error(err))
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	reg.Status = entity.Init
	reg.ID = uuid.NewString()
	reg.Timestamp = time.Now().UnixMilli()

	if err := r.dbClient.CreateJob(ctx, &reg); err != nil {
		r.logger.Error("create job", zap.Error(err))
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	r.orchestrator.Send(reg)

	ctx.JSON(http.StatusOK, reg)
}

func (r *Server) updateJob(ctx *gin.Context) {

}
