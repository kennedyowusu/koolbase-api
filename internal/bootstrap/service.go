package bootstrap

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
)

// BootstrapService handles the READ path only.
// It resolves an environment from a public key and returns the prebuilt snapshot.
// All payload assembly happens in SnapshotBuilder (the write path).
type BootstrapService struct {
	envRepo      *EnvironmentRepository
	snapshotRepo *SnapshotRepository
	builder      *SnapshotBuilder
}

func NewService(
	envRepo *EnvironmentRepository,
	snapshotRepo *SnapshotRepository,
	builder *SnapshotBuilder,
) *BootstrapService {
	return &BootstrapService{
		envRepo:      envRepo,
		snapshotRepo: snapshotRepo,
		builder:      builder,
	}
}

// GetSnapshot returns the prebuilt bootstrap snapshot for a public key.
// Read path: public_key → environment_id → Redis snapshot → raw JSON bytes.
// If no snapshot exists in Redis, it triggers a rebuild from Postgres.
func (s *BootstrapService) GetSnapshot(ctx context.Context, publicKey string) ([]byte, string, error) {
	// 1. Resolve environment
	env, err := s.envRepo.FindByPublicKey(ctx, publicKey)
	if err != nil {
		return nil, "", err
	}

	// 2. Fetch prebuilt snapshot from Redis
	data, err := s.snapshotRepo.Get(ctx, env.ID)
	if err == nil {
		return data, env.ID, nil
	}

	// 3. Cache miss — rebuild snapshot from Postgres
	if errors.Is(err, ErrSnapshotNotFound) {
		log.Info().Str("env_id", env.ID).Msg("snapshot not found, rebuilding")

		if buildErr := s.builder.Rebuild(ctx, env.ID); buildErr != nil {
			return nil, "", buildErr
		}

		snap, err := s.snapshotRepo.Get(ctx, env.ID)
		return snap, env.ID, err
	}

	return nil, "", err
}
