package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kennedyowusu/hatchway-api/internal/functions"
	"github.com/kennedyowusu/hatchway-api/internal/realtime"
)

type Service struct {
	repo  *Repository
	hub   *realtime.Hub
	fnSvc *functions.Service
}

func NewService(repo *Repository, hub *realtime.Hub, fnSvc *functions.Service) *Service {
	return &Service{repo: repo, hub: hub, fnSvc: fnSvc}
}

const (
	RulePublic        = "public"
	RuleAuthenticated = "authenticated"
	RuleOwner         = "owner"
	RuleScoped        = "scoped"
	RuleConditional   = "conditional"
)

func (s *Service) CreateCollection(ctx context.Context, projectID string, req CreateCollectionRequest) (*Collection, error) {
	if req.Name == "" {
		return nil, errors.New("collection name is required")
	}
	if err := validateCollectionName(req.Name); err != nil {
		return nil, err
	}

	readRule := req.ReadRule
	if readRule == "" {
		readRule = RuleAuthenticated
	}
	writeRule := req.WriteRule
	if writeRule == "" {
		writeRule = RuleAuthenticated
	}
	deleteRule := req.DeleteRule
	if deleteRule == "" {
		deleteRule = RuleOwner
	}

	if !isValidReadRule(readRule) {
		return nil, errors.New("invalid read_rule: must be public, authenticated, owner, scoped, or conditional")
	}
	if !isValidWriteRule(writeRule) {
		return nil, errors.New("invalid write_rule: must be authenticated, owner, scoped, or conditional")
	}
	if !isValidDeleteRule(deleteRule) {
		return nil, errors.New("invalid delete_rule: must be authenticated, owner, scoped, or conditional")
	}

	ruleMode := req.RuleMode
	if ruleMode == "" {
		ruleMode = "all"
	}
	if ruleMode != "all" && ruleMode != "any" {
		return nil, errors.New("invalid rule_mode: must be all or any")
	}

	conditionsJSON := []byte("[]")
	if len(req.RuleConditions) > 0 {
		if err := validateConditions(req.RuleConditions); err != nil {
			return nil, err
		}
		var err error
		conditionsJSON, err = encodeConditions(req.RuleConditions)
		if err != nil {
			return nil, err
		}
	}

	return s.repo.CreateCollection(ctx, projectID, req.Name, readRule, writeRule, deleteRule, req.OwnerField, ruleMode, conditionsJSON)
}

func (s *Service) ListCollections(ctx context.Context, projectID string) ([]Collection, error) {
	return s.repo.ListCollections(ctx, projectID)
}

func (s *Service) DeleteCollection(ctx context.Context, projectID, name string) error {
	return s.repo.DeleteCollection(ctx, projectID, name)
}

func (s *Service) Insert(ctx context.Context, projectID, userID string, req InsertRequest) (*Record, error) {
	if req.Collection == "" {
		return nil, errors.New("collection is required")
	}
	if len(req.Data) == 0 {
		return nil, errors.New("data cannot be empty")
	}

	col, err := s.repo.GetCollection(ctx, projectID, req.Collection)
	if err != nil {
		return nil, ErrCollectionNotFound
	}

	if err := checkWritePermission(col.WriteRule, userID); err != nil {
		return nil, err
	}

	// Enforce scoped/conditional write rules
	if col.WriteRule == RuleScoped || col.WriteRule == RuleConditional {
		userCtx, err := s.repo.GetUserContext(ctx, userID)
		if err != nil {
			return nil, errors.New("failed to resolve user context")
		}
		autoInjectUserFields(req.Data, userCtx, col.RuleConditions)
		if col.WriteRule == RuleScoped && col.OwnerField != nil {
			if fmt.Sprintf("%v", req.Data[*col.OwnerField]) != fmt.Sprintf("%v", userCtx[*col.OwnerField]) {
				return nil, errors.New("permission denied: data does not match user scope")
			}
		}
		if col.WriteRule == RuleConditional {
			if !evaluateRule(req.Data, userCtx, col.RuleMode, col.RuleConditions) {
				return nil, errors.New("permission denied: write conditions not met")
			}
		}
	}

	var createdBy *string
	if userID != "" {
		createdBy = &userID
	}

	rec, err := s.repo.InsertRecord(ctx, projectID, col.ID, createdBy, req.Data)
	if err != nil {
		return nil, err
	}
	if s.hub != nil {
		s.hub.PublishRecordCreated(projectID, col.Name, rec)
	}
	if s.fnSvc != nil {
		s.fireTriggers(ctx, projectID, "db.record.created", col.Name, map[string]interface{}{"record": rec})
	}
	return rec, nil
}

func (s *Service) Get(ctx context.Context, projectID, userID, recordID string) (*Record, error) {
	rec, err := s.repo.GetRecord(ctx, projectID, recordID)
	if err != nil {
		return nil, err
	}

	col, err := s.repo.GetCollectionByID(ctx, rec.CollectionID)
	if err == nil {
		if err := checkReadPermission(col.ReadRule, userID, rec.CreatedBy); err != nil {
			return nil, err
		}
	}

	return rec, nil
}

func (s *Service) Query(ctx context.Context, projectID, userID string, req QueryRequest) ([]Record, int, error) {
	if req.Collection == "" {
		return nil, 0, errors.New("collection is required")
	}

	col, err := s.repo.GetCollection(ctx, projectID, req.Collection)
	if err != nil {
		return nil, 0, ErrCollectionNotFound
	}

	if col.ReadRule == RuleOwner && userID == "" {
		return nil, 0, errors.New("authentication required")
	}
	if col.ReadRule == RuleAuthenticated && userID == "" {
		return nil, 0, errors.New("authentication required")
	}
	if (col.ReadRule == RuleScoped || col.ReadRule == RuleConditional) && userID == "" {
		return nil, 0, errors.New("authentication required")
	}

	filters := req.Filters
	if filters == nil {
		filters = map[string]interface{}{}
	}

	// If owner rule — scope to user's own records
	if col.ReadRule == RuleOwner && userID != "" {
		filters["created_by"] = userID
	}

	// Enforce scoped read rule — inject owner_field as filter
	if col.ReadRule == RuleScoped && col.OwnerField != nil {
		userCtx, err := s.repo.GetUserContext(ctx, userID)
		if err == nil && userCtx[*col.OwnerField] != nil {
			filters[*col.OwnerField] = fmt.Sprintf("%v", userCtx[*col.OwnerField])
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	records, total, err := s.repo.QueryRecords(ctx, projectID, col.ID, filters, limit, req.Offset, req.OrderBy, req.OrderDesc)
	if err != nil {
		return nil, 0, err
	}

	// Post-fetch filtering for conditional read rules
	if col.ReadRule == RuleConditional && len(col.RuleConditions) > 0 {
		userCtx, _ := s.repo.GetUserContext(ctx, userID)
		filtered := records[:0]
		for _, rec := range records {
			if evaluateRule(rec.Data, userCtx, col.RuleMode, col.RuleConditions) {
				filtered = append(filtered, rec)
			}
		}
		records = filtered
		total = len(records)
	}

	// Populate related records if requested
	if len(req.Populate) > 0 {
		if err := s.repo.PopulateRecords(ctx, projectID, records, req.Populate); err != nil {
			return nil, 0, err
		}
	}

	return records, total, nil
}

func (s *Service) Update(ctx context.Context, projectID, userID, recordID string, req UpdateRequest) (*Record, error) {
	if len(req.Data) == 0 {
		return nil, errors.New("data cannot be empty")
	}

	rec, err := s.repo.GetRecord(ctx, projectID, recordID)
	if err != nil {
		return nil, err
	}

	col, err := s.repo.GetCollectionByID(ctx, rec.CollectionID)
	if err == nil {
		if err := checkWritePermission(col.WriteRule, userID); err != nil {
			return nil, err
		}
		if col.WriteRule == RuleOwner {
			if rec.CreatedBy == nil || *rec.CreatedBy != userID {
				return nil, errors.New("permission denied: not the owner")
			}
		}
	}

	updated, err := s.repo.UpdateRecord(ctx, projectID, recordID, req.Data)
	if err != nil {
		return nil, err
	}
	colName := ""
	if col, err2 := s.repo.GetCollectionByID(ctx, updated.CollectionID); err2 == nil {
		colName = col.Name
	}
	if s.hub != nil {
		s.hub.PublishRecordUpdated(projectID, colName, updated)
	}
	if s.fnSvc != nil && colName != "" {
		s.fireTriggers(ctx, projectID, "db.record.updated", colName, map[string]interface{}{"record": updated})
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, projectID, userID, recordID string) error {
	rec, err := s.repo.GetRecord(ctx, projectID, recordID)
	if err != nil {
		return err
	}

	col, err := s.repo.GetCollectionByID(ctx, rec.CollectionID)
	if err == nil {
		if col.DeleteRule == RuleOwner {
			if rec.CreatedBy == nil || *rec.CreatedBy != userID {
				return errors.New("permission denied: not the owner")
			}
		}
	}

	colName := ""
	if col != nil {
		colName = col.Name
	}
	if err := s.repo.DeleteRecord(ctx, projectID, recordID); err != nil {
		return err
	}
	if s.hub != nil {
		s.hub.PublishRecordDeleted(projectID, colName, recordID)
	}
	if s.fnSvc != nil && colName != "" {
		s.fireTriggers(ctx, projectID, "db.record.deleted", colName, map[string]interface{}{"record_id": recordID})
	}
	return nil
}

// ─── Permission Helpers ────────────────────────────────────────────────────

func checkReadPermission(rule, userID string, createdBy *string) error {
	if rule == RulePublic {
		return nil
	}
	if rule == RuleAuthenticated && userID == "" {
		return errors.New("authentication required")
	}
	if rule == RuleOwner {
		if userID == "" {
			return errors.New("authentication required")
		}
		if createdBy == nil || *createdBy != userID {
			return errors.New("permission denied: not the owner")
		}
	}
	return nil
}

func checkWritePermission(rule, userID string) error {
	if rule == RuleAuthenticated && userID == "" {
		return errors.New("authentication required")
	}
	return nil
}

func isValidReadRule(r string) bool {
	return r == RulePublic || r == RuleAuthenticated || r == RuleOwner || r == RuleScoped || r == RuleConditional
}

func isValidWriteRule(r string) bool {
	return r == RuleAuthenticated || r == RuleOwner || r == RuleScoped || r == RuleConditional
}

func isValidDeleteRule(r string) bool {
	return r == RuleAuthenticated || r == RuleOwner || r == RuleScoped || r == RuleConditional
}

func validateCollectionName(name string) error {
	if len(name) > 63 {
		return errors.New("collection name too long")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return errors.New("collection name must be lowercase letters, numbers, underscores or hyphens")
		}
	}
	return nil
}

// ─── Rule Evaluation ──────────────────────────────────────────────────────

func evaluateRule(record map[string]interface{}, user map[string]interface{}, mode string, conditions []Condition) bool {
	if len(conditions) == 0 {
		return true
	}
	if mode == "any" {
		for _, c := range conditions {
			if evaluateCondition(record, user, c) {
				return true
			}
		}
		return false
	}
	// default = "all"
	for _, c := range conditions {
		if !evaluateCondition(record, user, c) {
			return false
		}
	}
	return true
}

func evaluateCondition(record map[string]interface{}, user map[string]interface{}, c Condition) bool {
	recordVal := record[c.Field]

	var compareVal interface{}
	if c.Source == "user" {
		compareVal = user[c.Field]
	} else {
		compareVal = c.Value
	}

	switch c.Type {
	case "equals":
		return fmt.Sprintf("%v", recordVal) == fmt.Sprintf("%v", compareVal)
	case "not_equals":
		return fmt.Sprintf("%v", recordVal) != fmt.Sprintf("%v", compareVal)
	case "in":
		return containsValue(compareVal, recordVal)
	case "not_in":
		return !containsValue(compareVal, recordVal)
	default:
		return false
	}
}

func containsValue(list interface{}, val interface{}) bool {
	valStr := fmt.Sprintf("%v", val)
	switch v := list.(type) {
	case []interface{}:
		for _, item := range v {
			if fmt.Sprintf("%v", item) == valStr {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if item == valStr {
				return true
			}
		}
	}
	return false
}

func autoInjectUserFields(data map[string]interface{}, user map[string]interface{}, conditions []Condition) {
	for _, c := range conditions {
		if c.Source == "user" {
			if _, exists := data[c.Field]; !exists {
				if userVal, ok := user[c.Field]; ok {
					data[c.Field] = userVal
				}
			}
		}
	}
}

func validateConditions(conditions []Condition) error {
	validTypes := map[string]bool{"equals": true, "not_equals": true, "in": true, "not_in": true}
	for _, c := range conditions {
		if !validTypes[c.Type] {
			return fmt.Errorf("invalid condition type: %s", c.Type)
		}
		if c.Field == "" {
			return errors.New("condition field is required")
		}
		if c.Source != "user" && c.Source != "" && c.Value == nil {
			return errors.New("condition must have source=user or a value")
		}
	}
	return nil
}

func encodeConditions(conditions []Condition) ([]byte, error) {
	return json.Marshal(conditions)
}

func (s *Service) fireTriggers(ctx context.Context, projectID, eventType, collection string, payload map[string]interface{}) {
	triggers, err := s.fnSvc.GetTriggersForEvent(ctx, projectID, eventType, collection)
	if err != nil || len(triggers) == 0 {
		return
	}
	for _, t := range triggers {
		s.fnSvc.InvokeForTrigger(ctx, projectID, t.FunctionName, "", eventType, collection, payload)
	}
}
