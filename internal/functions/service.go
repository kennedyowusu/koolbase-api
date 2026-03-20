package functions

import (
	"context"
	"errors"
	"fmt"
	"os"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Deploy(ctx context.Context, projectID, name, code string, timeoutMs int) (*Function, error) {
	if name == "" {
		return nil, errors.New("function name is required")
	}
	if err := validateFunctionName(name); err != nil {
		return nil, err
	}
	if code == "" {
		return nil, errors.New("code is required")
	}
	if len(code) > 1024*1024 {
		return nil, errors.New("code exceeds 1MB limit")
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if timeoutMs > 30000 {
		timeoutMs = 30000
	}

	fn, err := s.repo.DeployFunction(ctx, projectID, name, code, timeoutMs)
	if err != nil {
		return nil, fmt.Errorf("deploy failed: %w", err)
	}

	if err := SyncFunctionToDisk(fn); err != nil {
		return nil, fmt.Errorf("failed to write function to disk: %w", err)
	}

	fmt.Printf("[functions] deployed %s v%d project=%s\n", fn.Name, fn.Version, projectID)
	return fn, nil
}

func (s *Service) Invoke(ctx context.Context, projectID, name, apiKey string, req InvokeRequest) (*InvokeResponse, error) {
	// Fix 2 — context cancellation check
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fn, err := s.repo.GetActiveFunction(ctx, projectID, name)
	if err != nil {
		// Fix 1 — don't swallow real errors
		if errors.Is(err, ErrFunctionNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	reqBody := map[string]interface{}{
		"body":    req.Body,
		"headers": req.Headers,
	}
	if req.Test && req.Payload != nil {
		reqBody["event"] = req.Event
		reqBody["collection"] = req.Collection
		reqBody["payload"] = req.Payload
	}
	input := ExecutionInput{
		Request: reqBody,
		Env: map[string]string{},
		DB: DBContext{
			ProjectID: projectID,
			APIKey:    apiKey,
			BaseURL:   os.Getenv("APP_URL"),
		},
	}

	result := Execute(fn, input)

	// Fix 4 — observability
	fmt.Printf("[functions] executed %s v%d status=%d duration=%dms\n",
		fn.Name, fn.Version, result.Status, result.DurationMs)

	status := "success"
	if result.Status == 504 {
		status = "timeout"
	} else if result.Status >= 400 {
		status = "error"
	}

	var outputPtr, errorPtr *string
	if result.Output != "" {
		outputPtr = &result.Output
	}
	if result.Error != "" {
		errorPtr = &result.Error
	}

	savedLog, logErr := s.repo.InsertLog(ctx, Log{
		FunctionID:      fn.ID,
		ProjectID:       projectID,
		FunctionVersion: fn.Version,
		TriggerType:     "http",
		Status:          status,
		DurationMs:      result.DurationMs,
		Output:          outputPtr,
		Error:           errorPtr,
	})

	logID := ""
	if logErr == nil && savedLog != nil {
		logID = savedLog.ID
	}

	return &InvokeResponse{
		Status: result.Status,
		Body:   result.Body,
		LogID:  logID,
	}, nil
}

// Fix 3 — trigger execution is non-blocking
func (s *Service) InvokeForTrigger(ctx context.Context, projectID, functionName, apiKey, eventType, collection string, payload map[string]interface{}) {
	// context check before spinning goroutine
	select {
	case <-ctx.Done():
		return
	default:
	}
	go s.executeTriggerAsync(projectID, functionName, apiKey, eventType, collection, payload)
}

func (s *Service) executeTriggerAsync(projectID, functionName, apiKey, eventType, collection string, payload map[string]interface{}) {
	ctx := context.Background()

	fn, err := s.repo.GetActiveFunction(ctx, projectID, functionName)
	if err != nil {
		fmt.Printf("[functions] trigger: function %s not found for project %s\n", functionName, projectID)
		return
	}

	input := ExecutionInput{
		Request: map[string]interface{}{
			"event":      eventType,
			"collection": collection,
			"payload":    payload,
		},
		Env: map[string]string{},
		DB: DBContext{
			ProjectID: projectID,
			APIKey:    apiKey,
			BaseURL:   os.Getenv("APP_URL"),
		},
	}

	result := Execute(fn, input)

	// Fix 4 — observability
	fmt.Printf("[functions] trigger %s v%d event=%s status=%d duration=%dms\n",
		fn.Name, fn.Version, eventType, result.Status, result.DurationMs)

	status := "success"
	if result.Status == 504 {
		status = "timeout"
	} else if result.Status >= 400 {
		status = "error"
	}

	var outputPtr, errorPtr *string
	if result.Output != "" {
		outputPtr = &result.Output
	}
	if result.Error != "" {
		errorPtr = &result.Error
	}

	evtType := eventType
	col := collection

	s.repo.InsertLog(ctx, Log{
		FunctionID:      fn.ID,
		ProjectID:       projectID,
		FunctionVersion: fn.Version,
		TriggerType:     "db",
		EventType:       &evtType,
		Collection:      &col,
		Status:          status,
		DurationMs:      result.DurationMs,
		Output:          outputPtr,
		Error:           errorPtr,
	})
}

func (s *Service) ListFunctions(ctx context.Context, projectID string) ([]Function, error) {
	return s.repo.ListFunctions(ctx, projectID)
}

func (s *Service) DeleteFunction(ctx context.Context, projectID, name string) error {
	return s.repo.DeleteFunction(ctx, projectID, name)
}

func (s *Service) ListLogs(ctx context.Context, projectID, functionID string, limit int) ([]Log, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListLogs(ctx, projectID, functionID, limit)
}

func (s *Service) CreateTrigger(ctx context.Context, projectID string, req CreateTriggerRequest) (*Trigger, error) {
	if req.FunctionName == "" || req.EventType == "" || req.Collection == "" {
		return nil, errors.New("function_name, event_type and collection are required")
	}
	// Fix 5 — validate collection name length
	if len(req.Collection) > 63 {
		return nil, errors.New("collection name too long")
	}
	validEvents := map[string]bool{
		"db.record.created": true,
		"db.record.updated": true,
		"db.record.deleted": true,
	}
	if !validEvents[req.EventType] {
		return nil, errors.New("invalid event_type")
	}
	if _, err := s.repo.GetActiveFunction(ctx, projectID, req.FunctionName); err != nil {
		return nil, errors.New("function not found or not active")
	}
	return s.repo.CreateTrigger(ctx, projectID, req.FunctionName, req.EventType, req.Collection)
}

func (s *Service) ListTriggers(ctx context.Context, projectID string) ([]Trigger, error) {
	return s.repo.ListTriggers(ctx, projectID)
}

func (s *Service) DeleteTrigger(ctx context.Context, projectID, triggerID string) error {
	return s.repo.DeleteTrigger(ctx, projectID, triggerID)
}

func validateFunctionName(name string) error {
	if len(name) > 63 {
		return errors.New("function name too long")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return errors.New("function name must be lowercase letters, numbers, hyphens or underscores")
		}
	}
	return nil
}

func (s *Service) GetTriggersForEvent(ctx context.Context, projectID, eventType, collection string) ([]Trigger, error) {
	return s.repo.GetTriggersForEvent(ctx, projectID, eventType, collection)
}
