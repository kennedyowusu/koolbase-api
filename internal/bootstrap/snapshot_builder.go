package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type SnapshotBuilder struct {
	db           *pgxpool.Pool
	snapshotRepo *SnapshotRepository
}

func NewSnapshotBuilder(db *pgxpool.Pool, snapshotRepo *SnapshotRepository) *SnapshotBuilder {
	return &SnapshotBuilder{db: db, snapshotRepo: snapshotRepo}
}

func (b *SnapshotBuilder) Rebuild(ctx context.Context, environmentID string) error {
	log.Info().Str("env_id", environmentID).Msg("rebuilding bootstrap snapshot")

	flags, err := b.fetchFlags(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("fetch flags: %w", err)
	}

	configs, err := b.fetchConfigs(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("fetch configs: %w", err)
	}

	versionPolicy, err := b.fetchVersionPolicy(ctx, environmentID)
	if err != nil {
		return fmt.Errorf("fetch version policy: %w", err)
	}

	payload := &Payload{
		Flags:   flags,
		Config:  configs,
		Version: versionPolicy,
	}

	intermediate, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal intermediate: %w", err)
	}
	payload.PayloadVersion = generateHash(intermediate)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal final payload: %w", err)
	}

	if err := b.snapshotRepo.Set(ctx, environmentID, data); err != nil {
		return fmt.Errorf("store snapshot: %w", err)
	}

	log.Info().
		Str("env_id", environmentID).
		Str("payload_version", payload.PayloadVersion).
		Msg("snapshot rebuilt successfully")

	return nil
}

func (b *SnapshotBuilder) fetchFlags(ctx context.Context, envID string) (map[string]Flag, error) {
	rows, err := b.db.Query(ctx,
		`SELECT key, enabled, rollout_percentage, kill_switch
		 FROM feature_flags
		 WHERE environment_id = $1`,
		envID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flags := make(map[string]Flag)
	for rows.Next() {
		var key string
		var f Flag
		if err := rows.Scan(&key, &f.Enabled, &f.RolloutPercentage, &f.KillSwitch); err != nil {
			return nil, err
		}
		flags[key] = f
	}
	return flags, rows.Err()
}

func (b *SnapshotBuilder) fetchConfigs(ctx context.Context, envID string) (map[string]any, error) {
	rows, err := b.db.Query(ctx,
		`SELECT key, value
		 FROM remote_configs
		 WHERE environment_id = $1`,
		envID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make(map[string]any)
	for rows.Next() {
		var key string
			var rawValue []byte
		if err := rows.Scan(&key, &rawValue); err != nil {
			return nil, err
		}
		configs[key] = json.RawMessage(rawValue)
	}
	return configs, rows.Err()
}

func (b *SnapshotBuilder) fetchVersionPolicy(ctx context.Context, envID string) (VersionPolicy, error) {
	var vp VersionPolicy
	err := b.db.QueryRow(ctx,
		`SELECT latest_version, min_version, force_update, update_message
		 FROM version_policies
		 WHERE environment_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`,
		envID,
	).Scan(&vp.Latest, &vp.MinSupported, &vp.ForceUpdate, &vp.UpdateMessage)

	if err != nil {
		log.Warn().Str("env_id", envID).Msg("no version policy found for environment")
		return VersionPolicy{}, nil
	}
	return vp, nil
}

func generateHash(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:6])
}

func castConfigValue(raw, typ string) any {
	switch typ {
	case "boolean":
		return raw == "true"
	case "number":
		var n float64
		fmt.Sscanf(raw, "%f", &n)
		return n
	case "json":
		var v any
		json.Unmarshal([]byte(raw), &v)
		return v
	default:
		return raw
	}
}
