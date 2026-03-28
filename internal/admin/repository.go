package admin

import (
	"context"
	"time"

	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ─── Stats ─────────────────────────────────────────────────────────────────

type Stats struct {
	TotalUsers        int `json:"total_users"`
	UsersThisWeek     int `json:"users_this_week"`
	TotalOrgs         int `json:"total_orgs"`
	TotalProjects     int `json:"total_projects"`
	FunctionCallsToday int `json:"function_calls_today"`
	DeadLettersCount  int `json:"dead_letters_count"`
	ActiveProjects    int `json:"active_projects"`
}

func (r *Repository) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{}
	weekAgo := time.Now().UTC().AddDate(0, 0, -7)
	dayAgo := time.Now().UTC().AddDate(0, 0, -1)
	sevenDaysAgo := time.Now().UTC().AddDate(0, 0, -7)

	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&stats.TotalUsers)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND created_at >= $1`, weekAgo).Scan(&stats.UsersThisWeek)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&stats.TotalOrgs)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM projects`).Scan(&stats.TotalProjects)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM function_logs WHERE created_at >= $1`, dayAgo).Scan(&stats.FunctionCallsToday)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM function_dead_letters`).Scan(&stats.DeadLettersCount)
	r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT p.id)
		FROM projects p
		WHERE EXISTS (
			SELECT 1 FROM db_records dr
			JOIN db_collections dc ON dc.id = dr.collection_id
			WHERE dc.project_id = p.id AND dr.created_at >= $1
		)`, sevenDaysAgo).Scan(&stats.ActiveProjects)

	return stats, nil
}

// ─── Orgs ──────────────────────────────────────────────────────────────────

type OrgRow struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Plan         string    `json:"plan"`
	MemberCount  int       `json:"member_count"`
	ProjectCount int       `json:"project_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type PaginatedOrgs struct {
	Data  []OrgRow `json:"data"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
	Pages int      `json:"pages"`
}

func (r *Repository) GetOrgs(ctx context.Context, page, limit int) (*PaginatedOrgs, error) {
	offset := (page - 1) * limit
	var total int
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT
			o.id, o.name, o.plan, o.created_at,
			COUNT(DISTINCT u.id) AS member_count,
			COUNT(DISTINCT p.id) AS project_count
		FROM organizations o
		LEFT JOIN users u ON u.org_id = o.id AND u.deleted_at IS NULL
		LEFT JOIN projects p ON p.org_id = o.id
		GROUP BY o.id, o.name, o.plan, o.created_at
		ORDER BY o.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []OrgRow{}
	for rows.Next() {
		var row OrgRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Plan, &row.CreatedAt, &row.MemberCount, &row.ProjectCount); err != nil {
			return nil, err
		}
		data = append(data, row)
	}

	pages := (total + limit - 1) / limit
	return &PaginatedOrgs{Data: data, Total: total, Page: page, Limit: limit, Pages: pages}, nil
}

// ─── Users ─────────────────────────────────────────────────────────────────

type UserRow struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	OrgName   string    `json:"org_name"`
	Plan      string    `json:"plan"`
	Role      string    `json:"role"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
}

type PaginatedUsers struct {
	Data  []UserRow `json:"data"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
	Pages int       `json:"pages"`
}

func (r *Repository) GetUsers(ctx context.Context, page, limit int, emailSearch string) (*PaginatedUsers, error) {
	offset := (page - 1) * limit
	var total int

	if emailSearch != "" {
		r.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND email ILIKE $1`,
			"%"+emailSearch+"%").Scan(&total)
	} else {
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&total)
	}

	query := `
		SELECT u.id, u.email, o.name, o.plan, u.role, u.verified, u.created_at
		FROM users u
		JOIN organizations o ON o.id = u.org_id
		WHERE u.deleted_at IS NULL`

	args := []interface{}{}
	argIdx := 1

	if emailSearch != "" {
		query += ` AND u.email ILIKE $` + itoa(argIdx)
		args = append(args, "%"+emailSearch+"%")
		argIdx++
	}

	query += ` ORDER BY u.created_at DESC LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []UserRow{}
	for rows.Next() {
		var row UserRow
		if err := rows.Scan(&row.ID, &row.Email, &row.OrgName, &row.Plan, &row.Role, &row.Verified, &row.CreatedAt); err != nil {
			return nil, err
		}
		data = append(data, row)
	}

	pages := (total + limit - 1) / limit
	return &PaginatedUsers{Data: data, Total: total, Page: page, Limit: limit, Pages: pages}, nil
}

// ─── Projects ──────────────────────────────────────────────────────────────

type ProjectRow struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	OrgName         string    `json:"org_name"`
	CollectionCount int       `json:"collection_count"`
	FunctionCount   int       `json:"function_count"`
	CreatedAt       time.Time `json:"created_at"`
}

type PaginatedProjects struct {
	Data  []ProjectRow `json:"data"`
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
	Pages int          `json:"pages"`
}

func (r *Repository) GetProjects(ctx context.Context, page, limit int) (*PaginatedProjects, error) {
	offset := (page - 1) * limit
	var total int
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM projects`).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT
			p.id, p.name, o.name, p.created_at,
			COUNT(DISTINCT dc.id) AS collection_count,
			COUNT(DISTINCT pf.id) AS function_count
		FROM projects p
		JOIN organizations o ON o.id = p.org_id
		LEFT JOIN db_collections dc ON dc.project_id = p.id
		LEFT JOIN project_functions pf ON pf.project_id = p.id AND pf.is_active = TRUE
		GROUP BY p.id, p.name, o.name, p.created_at
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []ProjectRow{}
	for rows.Next() {
		var row ProjectRow
		if err := rows.Scan(&row.ID, &row.Name, &row.OrgName, &row.CreatedAt, &row.CollectionCount, &row.FunctionCount); err != nil {
			return nil, err
		}
		data = append(data, row)
	}

	pages := (total + limit - 1) / limit
	return &PaginatedProjects{Data: data, Total: total, Page: page, Limit: limit, Pages: pages}, nil
}

// ─── Function Logs ─────────────────────────────────────────────────────────

type FunctionLogRow struct {
	ID           string    `json:"id"`
	FunctionName string    `json:"function_name"`
	ProjectName  string    `json:"project_name"`
	OrgName      string    `json:"org_name"`
	Status       string    `json:"status"`
	DurationMs   int       `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

type PaginatedFunctionLogs struct {
	Data  []FunctionLogRow `json:"data"`
	Total int              `json:"total"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
	Pages int              `json:"pages"`
}

func (r *Repository) GetFunctionLogs(ctx context.Context, page, limit int, status string) (*PaginatedFunctionLogs, error) {
	offset := (page - 1) * limit
	var total int

	if status != "" {
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM function_logs WHERE status = $1`, status).Scan(&total)
	} else {
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM function_logs`).Scan(&total)
	}

	query := `
		SELECT
			fl.id, pf.name, p.name, o.name, fl.status, fl.duration_ms, fl.created_at
		FROM function_logs fl
		JOIN project_functions pf ON pf.id = fl.function_id
		JOIN projects p ON p.id = fl.project_id
		JOIN organizations o ON o.id = p.org_id`

	args := []interface{}{}
	argIdx := 1

	if status != "" {
		query += ` WHERE fl.status = $` + itoa(argIdx)
		args = append(args, status)
		argIdx++
	}

	query += ` ORDER BY fl.created_at DESC LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []FunctionLogRow{}
	for rows.Next() {
		var row FunctionLogRow
		if err := rows.Scan(&row.ID, &row.FunctionName, &row.ProjectName, &row.OrgName, &row.Status, &row.DurationMs, &row.CreatedAt); err != nil {
			return nil, err
		}
		data = append(data, row)
	}

	pages := (total + limit - 1) / limit
	return &PaginatedFunctionLogs{Data: data, Total: total, Page: page, Limit: limit, Pages: pages}, nil
}

// ─── Dead Letters ──────────────────────────────────────────────────────────

type DeadLetterRow struct {
	ID           string    `json:"id"`
	FunctionName string    `json:"function_name"`
	ProjectName  string    `json:"project_name"`
	OrgName      string    `json:"org_name"`
	EventType    string    `json:"event_type"`
	LastError    string    `json:"last_error"`
	RetryCount   int       `json:"retry_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type PaginatedDeadLetters struct {
	Data  []DeadLetterRow `json:"data"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
	Pages int             `json:"pages"`
}

func (r *Repository) GetDeadLetters(ctx context.Context, page, limit int) (*PaginatedDeadLetters, error) {
	offset := (page - 1) * limit
	var total int
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM function_dead_letters`).Scan(&total)

	rows, err := r.db.Query(ctx, `
		SELECT
			dl.id, dl.function_name, p.name, o.name,
			COALESCE(dl.event_type, ''), dl.last_error, dl.retry_count, dl.created_at
		FROM function_dead_letters dl
		JOIN projects p ON p.id = dl.project_id
		JOIN organizations o ON o.id = p.org_id
		ORDER BY dl.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []DeadLetterRow{}
	for rows.Next() {
		var row DeadLetterRow
		if err := rows.Scan(&row.ID, &row.FunctionName, &row.ProjectName, &row.OrgName, &row.EventType, &row.LastError, &row.RetryCount, &row.CreatedAt); err != nil {
			return nil, err
		}
		data = append(data, row)
	}

	pages := (total + limit - 1) / limit
	return &PaginatedDeadLetters{Data: data, Total: total, Page: page, Limit: limit, Pages: pages}, nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
