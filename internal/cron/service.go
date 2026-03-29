package cron

import (
	"context"
	"errors"
	"fmt"
	"time"

	robfigcron "github.com/robfig/cron/v3"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

var parser = robfigcron.NewParser(
	robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow,
)

func validateCronExpression(expr string) error {
	_, err := parser.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}

func nextRunTime(expr string) (interface{}, error) {
	schedule, err := parser.Parse(expr)
	if err != nil {
		return nil, err
	}
	return schedule.Next(time.Now()), nil
}

func (s *Service) Create(ctx context.Context, projectID string, req CreateCronRequest) (*CronSchedule, error) {
	if req.FunctionName == "" {
		return nil, errors.New("function_name is required")
	}
	if req.CronExpression == "" {
		return nil, errors.New("cron_expression is required")
	}
	if err := validateCronExpression(req.CronExpression); err != nil {
		return nil, err
	}

	schedule, err := parser.Parse(req.CronExpression)
	if err != nil {
		return nil, err
	}
	nextRun := schedule.Next(time.Now())

	return s.repo.Create(ctx, projectID, req.FunctionName, req.CronExpression, nextRun)
}

func (s *Service) List(ctx context.Context, projectID string) ([]CronSchedule, error) {
	return s.repo.List(ctx, projectID)
}

func (s *Service) Delete(ctx context.Context, projectID, cronID string) error {
	return s.repo.Delete(ctx, projectID, cronID)
}

func (s *Service) Update(ctx context.Context, projectID, cronID string, req UpdateCronRequest) (*CronSchedule, error) {
	if req.Enabled == nil {
		return nil, errors.New("enabled is required")
	}
	return s.repo.Update(ctx, projectID, cronID, *req.Enabled)
}

func (s *Service) GetDue(ctx context.Context) ([]CronSchedule, error) {
	return s.repo.GetDue(ctx)
}

func (s *Service) MarkRan(ctx context.Context, c *CronSchedule) error {
	schedule, err := parser.Parse(c.CronExpression)
	if err != nil {
		return err
	}
	nextRun := schedule.Next(time.Now())
	return s.repo.MarkRan(ctx, c.ID, nextRun)
}
