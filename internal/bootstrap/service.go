package bootstrap

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
)

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

func (s *BootstrapService) GetSnapshot(ctx context.Context, publicKey string) ([]byte, string, error) {
	env, err := s.envRepo.FindByPublicKey(ctx, publicKey)
	if err != nil {
		return nil, "", err
	}

	data, err := s.snapshotRepo.Get(ctx, env.ID)
	if err == nil {
		return data, env.ID, nil
	}

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
