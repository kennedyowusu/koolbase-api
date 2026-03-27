package database

import "time"

// ─── Security Rules ────────────────────────────────────────────────────────

type Condition struct {
	Type   string      `json:"type"`             // equals | not_equals | in | not_in
	Field  string      `json:"field"`            // field in record.data to check
	Source string      `json:"source,omitempty"` // "user" — compare to user field
	Value  interface{} `json:"value,omitempty"`  // literal value to compare against
}

type RuleConfig struct {
	Mode       string      `json:"mode"`       // "all" (AND) | "any" (OR)
	Conditions []Condition `json:"conditions"` // flat list — no nesting
}

// ─── Collection ────────────────────────────────────────────────────────────

type Collection struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"project_id"`
	Name           string     `json:"name"`
	ReadRule       string     `json:"read_rule"`
	WriteRule      string     `json:"write_rule"`
	DeleteRule     string     `json:"delete_rule"`
	OwnerField     *string    `json:"owner_field,omitempty"`
	RuleMode       string     `json:"rule_mode"`
	RuleConditions []Condition `json:"rule_conditions"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ─── Record ────────────────────────────────────────────────────────────────

type Record struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	CollectionID string                 `json:"collection_id"`
	CreatedBy    *string                `json:"created_by,omitempty"`
	Data         map[string]interface{} `json:"data"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// ─── Requests ──────────────────────────────────────────────────────────────

type InsertRequest struct {
	Collection string                 `json:"collection"`
	Data       map[string]interface{} `json:"data"`
}

type UpdateRequest struct {
	Data map[string]interface{} `json:"data"`
}

type QueryRequest struct {
	Collection string                 `json:"collection"`
	Filters    map[string]interface{} `json:"filters"`
	Limit      int                    `json:"limit"`
	Offset     int                    `json:"offset"`
	OrderBy    string                 `json:"order_by"`
	OrderDesc  bool                   `json:"order_desc"`
	Populate   []string               `json:"populate"`
}

type CreateCollectionRequest struct {
	Name           string      `json:"name"`
	ReadRule       string      `json:"read_rule"`
	WriteRule      string      `json:"write_rule"`
	DeleteRule     string      `json:"delete_rule"`
	OwnerField     string      `json:"owner_field"`
	RuleMode       string      `json:"rule_mode"`
	RuleConditions []Condition `json:"rule_conditions"`
}
