package db

import (
	"context"
	"corgiAnalytics/runner/internal/entity"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func PostFrame(client *redis.Client, id string, frame *entity.Frame) error {
	res, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	cmd := client.Set(context.Background(), id, res, time.Duration(3)*time.Minute)
	if err := cmd.Err(); err != nil {
		return err
	}

	return nil
}

func ReadFrame(client *redis.Client, id string) (*entity.Frame, error) {
	cmd := client.Get(context.Background(), id)
	if err := cmd.Err(); err != nil {
		return nil, err
	}

	var frame entity.Frame

	b, err := cmd.Bytes()
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(b, &frame); err != nil {
		return nil, err
	}

	return &frame, nil
}
