package bootstrap

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var ErrEnvironmentNotFound = errors.New("environment not found")

// Request holds parsed query params from the SDK
type Request struct {
	PublicKey  string
	DeviceID   string
	Platform   string
	AppVersion string
}

// Payload is the atomic bootstrap response returned to the SDK.
// NOTE: No rollout bucket — SDK computes stableHash(deviceID+":"+flagKey) % 100 locally.
type Payload struct {
	PayloadVersion string          `json:"payload_version"`
	Flags          map[string]Flag `json:"flags"`
	Config         map[string]any  `json:"config"`
	Version        VersionPolicy   `json:"version"`
}

// Flag represents a single feature flag rule.
// SDK evaluates: stableHash(deviceID + ":" + flagKey) % 100 < RolloutPercentage
type Flag struct {
	Enabled           bool `json:"enabled"`
	RolloutPercentage int  `json:"rollout_percentage"`
	KillSwitch        bool `json:"kill_switch"`
}

// VersionPolicy controls forced/soft update behavior
type VersionPolicy struct {
	Latest        string `json:"latest_version"`
	MinSupported  string `json:"min_version"`
	ForceUpdate   bool   `json:"force_update"`
	UpdateMessage string `json:"update_message"`
}

// registerDevice records device activity asynchronously.
// Called as a goroutine — never blocks the bootstrap response.
func registerDevice(ctx context.Context, db *pgxpool.Pool, req *Request) {
	// Resolve environment_id from public_key for device registration
	var envID string
	err := db.QueryRow(ctx,
		`SELECT id FROM environments WHERE public_key = $1 LIMIT 1`,
		req.PublicKey,
	).Scan(&envID)
	if err != nil {
		return
	}

	_, err = db.Exec(ctx,
		`INSERT INTO devices (environment_id, device_id, platform, app_version, last_seen_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (environment_id, device_id)
		 DO UPDATE SET app_version = $4, last_seen_at = NOW()`,
		envID, req.DeviceID, req.Platform, req.AppVersion,
	)
	if err != nil {
		log.Error().Err(err).Str("device_id", req.DeviceID).Msg("device registration failed")
	}
}
