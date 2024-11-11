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
		&job.Timestamp,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *PostgresClient) GetJob(ctx context.Context, jobID string) (*entity.Job, error) {
	var job entity.Job

	err := r.conn.QueryRowContext(
		ctx,
		readJobQuery,
		jobID,
	).Scan(
		&job.ID,
		&job.Status,
		&job.URL,
		&job.Timestamp,
	)
	if err != nil {
		return nil, err
	}

	return &job, nil
}
