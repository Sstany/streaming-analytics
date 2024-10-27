package db

import (
	"context"
	"database/sql"

	"go.uber.org/zap"
)

type PostgresClient struct {
	conn   *sql.Conn
	logger *zap.Logger
}

func NewPostgresClient(ctx context.Context, connStr string) *PostgresClient {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		panic(err)
	}

	return &PostgresClient{
		conn: conn,
	}
}
