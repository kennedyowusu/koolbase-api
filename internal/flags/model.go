package flags

import "time"

type Flag struct {
	ID                string    `json:"id"`
	EnvironmentID     string    `json:"environment_id"`
	Key               string    `json:"key"`
	Enabled           bool      `json:"enabled"`
	RolloutPercentage int       `json:"rollout_percentage"`
	KillSwitch        bool      `json:"kill_switch"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
