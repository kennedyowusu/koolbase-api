package functions

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Exponential backoff: 1, 2, 4, 8, 16 minutes
var retryBackoff = []time.Duration{
	1 * time.Minute,
	2 * time.Minute,
	4 * time.Minute,
	8 * time.Minute,
	16 * time.Minute,
}

type Worker struct {
	repo *Repository
	svc  *Service
}

func NewWorker(repo *Repository, svc *Service) *Worker {
	return &Worker{repo: repo, svc: svc}
}

func (w *Worker) Start(ctx context.Context) {
	fmt.Println("[worker] retry worker started")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	w.processRetries(ctx)

	for {
		select {
		case <-ticker.C:
			w.processRetries(ctx)
		case <-ctx.Done():
			fmt.Println("[worker] retry worker stopped")
			return
		}
	}
}

func (w *Worker) processRetries(ctx context.Context) {
	jobs, err := w.repo.GetDueRetries(ctx)
	if err != nil {
		fmt.Printf("[worker] failed to get due retries: %v\n", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	fmt.Printf("[worker] processing %d retry job(s)\n", len(jobs))

	for _, job := range jobs {
		w.processJob(ctx, job)
	}
}

func (w *Worker) processJob(ctx context.Context, job RetryJob) {
	fmt.Printf("[worker] retrying %s attempt=%d/%d event=%s collection=%s\n",
		job.FunctionName, job.Attempt+1, job.MaxAttempts, job.EventType, job.Collection)

	fn, err := w.repo.GetActiveFunction(ctx, job.ProjectID, job.FunctionName)
	if err != nil {
		fmt.Printf("[worker] function %s not found: %v\n", job.FunctionName, err)
		w.moveToDLQ(ctx, job, "function not found: "+err.Error())
		return
	}

	input := ExecutionInput{
		Request: map[string]interface{}{
			"event":      job.EventType,
			"collection": job.Collection,
			"payload":    job.Payload,
		},
		Env: map[string]string{},
		DB: DBContext{
			ProjectID: job.ProjectID,
			APIKey:    job.APIKey,
			BaseURL:   os.Getenv("APP_URL"),
		},
	}

	result := Execute(fn, input)

	// Determine status
	status := "success"
	if result.Status == 504 {
		status = "timeout"
	} else if result.Status >= 400 {
		status = "error"
	}

	// Log the attempt
	var outputPtr, errorPtr *string
	if result.Output != "" {
		outputPtr = &result.Output
	}
	if result.Error != "" {
		errorPtr = &result.Error
	}
	evtType := job.EventType
	col := job.Collection
	w.repo.InsertLog(ctx, Log{
		FunctionID:      fn.ID,
		ProjectID:       job.ProjectID,
		FunctionVersion: fn.Version,
		TriggerType:     "db",
		EventType:       &evtType,
		Collection:      &col,
		Status:          status,
		DurationMs:      result.DurationMs,
		Output:          outputPtr,
		Error:           errorPtr,
	})

	if result.Status == 200 {
		fmt.Printf("[worker] retry succeeded: %s\n", job.FunctionName)
		w.repo.MarkRetrySuccess(ctx, job.ID)
		return
	}

	// Failed — increment attempt
	newAttempt := job.Attempt + 1
	lastError := result.Error
	if lastError == "" {
		lastError = fmt.Sprintf("execution failed with status %d", result.Status)
	}

	if newAttempt >= job.MaxAttempts {
		fmt.Printf("[worker] max attempts reached for %s — moving to DLQ\n", job.FunctionName)
		job.Attempt = newAttempt
		job.LastError = lastError
		w.moveToDLQ(ctx, job, lastError)
		return
	}

	// Schedule next retry with exponential backoff
	backoff := retryBackoff[newAttempt-1]
	if newAttempt-1 >= len(retryBackoff) {
		backoff = 16 * time.Minute
	}
	nextRetryAt := time.Now().UTC().Add(backoff)

	fmt.Printf("[worker] scheduling retry %d for %s in %v\n", newAttempt, job.FunctionName, backoff)
	w.repo.MarkRetryFailed(ctx, job.ID, newAttempt, lastError, nextRetryAt)
}

func (w *Worker) moveToDLQ(ctx context.Context, job RetryJob, reason string) {
	job.LastError = reason
	if err := w.repo.MoveToDeadLetter(ctx, job); err != nil {
		fmt.Printf("[worker] failed to move job to DLQ: %v\n", err)
	}
}
