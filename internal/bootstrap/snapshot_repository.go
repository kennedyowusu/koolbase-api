package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const snapshotTTL = 24 * time.Hour

var ErrSnapshotNotFound = errors.New("snapshot not found")

type SnapshotRepository struct {
	rdb *redis.Client
}

func NewSnapshotRepository(rdb *redis.Client) *SnapshotRepository {
	return &SnapshotRepository{rdb: rdb}
}

func (r *SnapshotRepository) Get(ctx context.Context, environmentID string) ([]byte, error) {
	key := snapshotKey(environmentID)
	data, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("snapshot get failed: %w", err)
	}
	return data, nil
}

func (r *SnapshotRepository) Set(ctx context.Context, environmentID string, data []byte) error {
	key := snapshotKey(environmentID)
	return r.rdb.Set(ctx, key, data, snapshotTTL).Err()
}

func (r *SnapshotRepository) Invalidate(ctx context.Context, environmentID string) error {
	key := snapshotKey(environmentID)
	return r.rdb.Del(ctx, key).Err()
}

func snapshotKey(environmentID string) string {
	return fmt.Sprintf("bootstrap:env:%s", environmentID)
}
