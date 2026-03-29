package cron

import "time"

type CronSchedule struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"project_id"`
	FunctionName   string     `json:"function_name"`
	CronExpression string     `json:"cron_expression"`
	Enabled        bool       `json:"enabled"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type CreateCronRequest struct {
	FunctionName   string `json:"function_name"`
	CronExpression string `json:"cron_expression"`
}

type UpdateCronRequest struct {
	Enabled *bool `json:"enabled"`
}
