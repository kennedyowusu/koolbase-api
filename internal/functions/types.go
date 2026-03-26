package functions

import "time"

type Function struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"project_id"`
	Name           string     `json:"name"`
	Runtime        string     `json:"runtime"`
	EntryFile      string     `json:"entry_file"`
	Code           string     `json:"code"`
	Version        int        `json:"version"`
	IsActive       bool       `json:"is_active"`
	TimeoutMs      int        `json:"timeout_ms"`
	Enabled        bool       `json:"enabled"`
	LastDeployedAt *time.Time `json:"last_deployed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Trigger struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	FunctionName string    `json:"function_name"`
	EventType    string    `json:"event_type"`
	Collection   string    `json:"collection"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

type Log struct {
	ID              string    `json:"id"`
	FunctionID      string    `json:"function_id"`
	ProjectID       string    `json:"project_id"`
	FunctionVersion int       `json:"function_version"`
	TriggerType     string    `json:"trigger_type"`
	EventType       *string   `json:"event_type,omitempty"`
	Collection      *string   `json:"collection,omitempty"`
	Status          string    `json:"status"`
	DurationMs      int       `json:"duration_ms"`
	Output          *string   `json:"output,omitempty"`
	Error           *string   `json:"error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type DeployRequest struct {
	Runtime   string `json:"runtime"`   // "deno" (default) or "dart"
	Name      string `json:"name"`
	Code      string `json:"code"`
	TimeoutMs int    `json:"timeout_ms"`
}

type InvokeRequest struct {
	Body       map[string]interface{} `json:"body"`
	Headers    map[string]string      `json:"headers"`
	Test       bool                   `json:"test"`
	Event      string                 `json:"event"`
	Collection string                 `json:"collection"`
	Payload    map[string]interface{} `json:"payload"`
}

type InvokeResponse struct {
	Status int                    `json:"status"`
	Body   map[string]interface{} `json:"body"`
	LogID  string                 `json:"log_id"`
}

type CreateTriggerRequest struct {
	FunctionName string `json:"function_name"`
	EventType    string `json:"event_type"`
	Collection   string `json:"collection"`
}

type Secret struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpsertSecretRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
