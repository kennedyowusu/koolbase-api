package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/kennedyowusu/hatchway-api/internal/functions"
)

type Worker struct {
	svc   *Service
	fnSvc *functions.Service
}

func NewWorker(svc *Service, fnSvc *functions.Service) *Worker {
	return &Worker{svc: svc, fnSvc: fnSvc}
}

func (w *Worker) Start() {
	fmt.Println("[cron] worker started")
	go w.run()
}

func (w *Worker) run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Run immediately on start to catch any missed schedules
	w.tick()

	for range ticker.C {
		w.tick()
	}
}

func (w *Worker) tick() {
	ctx := context.Background()

	due, err := w.svc.GetDue(ctx)
	if err != nil {
		fmt.Printf("[cron] failed to get due schedules: %v\n", err)
		return
	}

	if len(due) == 0 {
		return
	}

	fmt.Printf("[cron] firing %d due schedule(s)\n", len(due))

	for _, schedule := range due {
		go w.fire(ctx, schedule)
	}
}

func (w *Worker) fire(ctx context.Context, schedule CronSchedule) {
	fmt.Printf("[cron] firing %s → %s\n", schedule.CronExpression, schedule.FunctionName)

	// Mark as ran first to prevent double-firing
	if err := w.svc.MarkRan(ctx, &schedule); err != nil {
		fmt.Printf("[cron] failed to mark schedule %s as ran: %v\n", schedule.ID, err)
		return
	}

	// Invoke the function
	w.fnSvc.InvokeForTrigger(
		ctx,
		schedule.ProjectID,
		schedule.FunctionName,
		"",
		"cron",
		"",
		map[string]interface{}{
			"cron_id":    schedule.ID,
			"expression": schedule.CronExpression,
			"fired_at":   time.Now().UTC().Format(time.RFC3339),
		},
	)
}
