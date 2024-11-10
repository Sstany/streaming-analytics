package db

import (
	"context"
	"corgiAnalytics/orchestrator/internal/entity"
)

func (r *PostgresClient) CreateJob(ctx context.Context, job *entity.Job) error {
	_, err := r.conn.ExecContext(
		ctx,
		createJobQuery,
		&job.ID,
		&job.Status,
		&job.URL,
	)
	if err != nil {
		return err
	}

	return nil
}
