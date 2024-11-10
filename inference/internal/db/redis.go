package db

import (
	"context"
	"corgiAnalytics/inference/internal/entity"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

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
