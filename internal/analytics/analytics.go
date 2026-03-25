package analytics

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

type DailyStat struct {
	Date         string `json:"date"`
	RequestCount int    `json:"request_count"`
}

type PlatformStat struct {
	Platform     string `json:"platform"`
	RequestCount int    `json:"request_count"`
}

type EnvironmentStat struct {
	EnvironmentID   string `json:"environment_id"`
	EnvironmentName string `json:"environment_name"`
	RequestCount    int    `json:"request_count"`
}

type Summary struct {
	TotalRequests   int `json:"total_requests"`
	TotalDevices    int `json:"total_devices"`
	RequestsToday   int `json:"requests_today"`
	ActiveEnvs      int `json:"active_envs"`
}

func (h *Handler) GetOrgStats(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")
	days := 30

	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	var summary Summary

	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(bs.request_count), 0)
		 FROM bootstrap_stats bs
		 JOIN environments e ON e.id = bs.environment_id
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1 AND bs.date >= $2`,
		orgID, since,
	).Scan(&summary.TotalRequests)

	h.db.QueryRow(r.Context(),
		`SELECT COUNT(DISTINCT d.device_id)
		 FROM devices d
		 JOIN environments e ON e.id = d.environment_id
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1`,
		orgID,
	).Scan(&summary.TotalDevices)

	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(bs.request_count), 0)
		 FROM bootstrap_stats bs
		 JOIN environments e ON e.id = bs.environment_id
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1 AND bs.date = CURRENT_DATE`,
		orgID,
	).Scan(&summary.RequestsToday)

	h.db.QueryRow(r.Context(),
		`SELECT COUNT(DISTINCT bs.environment_id)
		 FROM bootstrap_stats bs
		 JOIN environments e ON e.id = bs.environment_id
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1 AND bs.date >= $2`,
		orgID, since,
	).Scan(&summary.ActiveEnvs)

	dailyRows, _ := h.db.Query(r.Context(),
		`SELECT bs.date::text, COALESCE(SUM(bs.request_count), 0)
		 FROM generate_series($2::date, CURRENT_DATE, '1 day'::interval) AS gs(date)
		 LEFT JOIN bootstrap_stats bs ON bs.date = gs.date
		   AND bs.environment_id IN (
		     SELECT e.id FROM environments e
		     JOIN projects p ON p.id = e.project_id
		     WHERE p.org_id = $1
		   )
		 GROUP BY gs.date
		 ORDER BY gs.date ASC`,
		orgID, since,
	)
	defer dailyRows.Close()

	daily := []DailyStat{}
	for dailyRows.Next() {
		var s DailyStat
		dailyRows.Scan(&s.Date, &s.RequestCount)
		daily = append(daily, s)
	}

	platformRows, _ := h.db.Query(r.Context(),
		`SELECT bs.platform, COALESCE(SUM(bs.request_count), 0)
		 FROM bootstrap_stats bs
		 JOIN environments e ON e.id = bs.environment_id
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1 AND bs.date >= $2
		 GROUP BY bs.platform
		 ORDER BY SUM(bs.request_count) DESC`,
		orgID, since,
	)
	defer platformRows.Close()

	platforms := []PlatformStat{}
	for platformRows.Next() {
		var s PlatformStat
		platformRows.Scan(&s.Platform, &s.RequestCount)
		platforms = append(platforms, s)
	}

	envRows, _ := h.db.Query(r.Context(),
		`SELECT e.id, e.name, COALESCE(SUM(bs.request_count), 0)
		 FROM environments e
		 JOIN projects p ON p.id = e.project_id
		 LEFT JOIN bootstrap_stats bs ON bs.environment_id = e.id AND bs.date >= $2
		 WHERE p.org_id = $1
		 GROUP BY e.id, e.name
		 ORDER BY SUM(bs.request_count) DESC
		 LIMIT 5`,
		orgID, since,
	)
	defer envRows.Close()

	envs := []EnvironmentStat{}
	for envRows.Next() {
		var s EnvironmentStat
		envRows.Scan(&s.EnvironmentID, &s.EnvironmentName, &s.RequestCount)
		envs = append(envs, s)
	}

	respond.OK(w, map[string]interface{}{
		"summary":      summary,
		"daily":        daily,
		"platforms":    platforms,
		"environments": envs,
		"period_days":  days,
	})
}
