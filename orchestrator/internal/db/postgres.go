package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type PostgresClient struct {
	conn   *sql.Conn
	logger *zap.Logger
}

func NewPostgresClient(ctx context.Context, connStr string, logger *zap.Logger) *PostgresClient {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if _, err := db.ExecContext(ctx, initTableQuery); err != nil {
		panic(err)
	}
	return &PostgresClient{
		conn:   conn,
		logger: logger.Named("postgres"),
	}
}
