package billing

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Repository interface {
	GetOrgPlan(ctx context.Context, orgID string) (string, error)
	CountEnvironments(ctx context.Context, orgID string) (int, error)
	CountFlags(ctx context.Context, orgID string) (int, error)
	CountConfigs(ctx context.Context, orgID string) (int, error)
	CountMembers(ctx context.Context, orgID string) (int, error)
	CountFunctions(ctx context.Context, projectID string) (int, error)
	CountSecrets(ctx context.Context, projectID string) (int, error)
}

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetUsage(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")
	projectID := r.URL.Query().Get("project_id")
	ctx := r.Context()

	plan, err := h.repo.GetOrgPlan(ctx, orgID)
	if err != nil {
		plan = "free"
	}

	type usageResponse struct {
		Plan   string  `json:"plan"`
		Limits []Usage `json:"limits"`
	}

	envCount, _ := h.repo.CountEnvironments(ctx, orgID)
	flagCount, _ := h.repo.CountFlags(ctx, orgID)
	configCount, _ := h.repo.CountConfigs(ctx, orgID)
	memberCount, _ := h.repo.CountMembers(ctx, orgID)

	usages := []Usage{
		{Resource: "environments", Current: envCount, Limit: GetLimit(plan, "environments"), Plan: plan},
		{Resource: "flags", Current: flagCount, Limit: GetLimit(plan, "flags"), Plan: plan},
		{Resource: "configs", Current: configCount, Limit: GetLimit(plan, "configs"), Plan: plan},
		{Resource: "members", Current: memberCount, Limit: GetLimit(plan, "members"), Plan: plan},
	}

	if projectID != "" {
		fnCount, _ := h.repo.CountFunctions(ctx, projectID)
		secretCount, _ := h.repo.CountSecrets(ctx, projectID)
		usages = append(usages,
			Usage{Resource: "functions", Current: fnCount, Limit: GetLimit(plan, "functions"), Plan: plan},
			Usage{Resource: "secrets", Current: secretCount, Limit: GetLimit(plan, "secrets"), Plan: plan},
		)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usageResponse{Plan: plan, Limits: usages})
}
