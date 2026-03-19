package database

import "time"

type Collection struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Name       string    `json:"name"`
	ReadRule   string    `json:"read_rule"`
	WriteRule  string    `json:"write_rule"`
	DeleteRule string    `json:"delete_rule"`
	CreatedAt  time.Time `json:"created_at"`
}

type Record struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	CollectionID string                 `json:"collection_id"`
	CreatedBy    *string                `json:"created_by,omitempty"`
	Data         map[string]interface{} `json:"data"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

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
}

type CreateCollectionRequest struct {
	Name       string `json:"name"`
	ReadRule   string `json:"read_rule"`
	WriteRule  string `json:"write_rule"`
	DeleteRule string `json:"delete_rule"`
}
