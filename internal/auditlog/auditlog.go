package auditlog

import (
	"time"
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type Action string

const (
	ActionFlagCreated        Action = "flag.created"
	ActionFlagUpdated        Action = "flag.updated"
	ActionFlagDeleted        Action = "flag.deleted"
	ActionConfigCreated      Action = "config.created"
	ActionConfigUpdated      Action = "config.updated"
	ActionConfigDeleted      Action = "config.deleted"
	ActionVersionUpdated     Action = "version.updated"
	ActionProjectCreated     Action = "project.created"
	ActionProjectDeleted     Action = "project.deleted"
	ActionEnvironmentCreated Action = "environment.created"
	ActionEnvironmentDeleted Action = "environment.deleted"
	ActionMemberInvited      Action = "member.invited"
	ActionMemberRemoved      Action = "member.removed"
	ActionInviteRevoked      Action = "invite.revoked"
)

type Entry struct {
	OrgID        string
	ActorID      string
	ResourceType string
	ResourceID   string
	Action       Action
	Diff         any
	IP           string
}

type Writer struct {
	db *pgxpool.Pool
}

func NewWriter(db *pgxpool.Pool) *Writer {
	return &Writer{db: db}
}

func (w *Writer) Write(ctx context.Context, e Entry) {
	go func() {
		var diffJSON []byte
		if e.Diff != nil {
			diffJSON, _ = json.Marshal(e.Diff)
		}

		_, err := w.db.Exec(ctx,
			`INSERT INTO audit_logs (org_id, actor_id, resource_type, resource_id, action, diff, ip)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			e.OrgID,
			nullableString(e.ActorID),
			e.ResourceType,
			e.ResourceID,
			string(e.Action),
			nullableJSON(diffJSON),
			nullableString(e.IP),
		)
		if err != nil {
			log.Error().Err(err).Str("action", string(e.Action)).Msg("failed to write audit log")
		}
	}()
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

type Log struct {
	ID           string     `json:"id"`
	ActorID      *string    `json:"actor_id"`
	ActorEmail   *string    `json:"actor_email"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	Action       string     `json:"action"`
	IP           *string    `json:"ip"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (w *Writer) List(ctx context.Context, orgID string, limit, offset int) ([]Log, int, error) {
	var total int
	w.db.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE org_id = $1`, orgID).Scan(&total)

	rows, err := w.db.Query(ctx,
		`SELECT a.id, a.actor_id, u.email, a.resource_type, a.resource_id, a.action, a.ip, a.created_at
		 FROM audit_logs a
		 LEFT JOIN users u ON u.id = a.actor_id
		 WHERE a.org_id = $1
		 ORDER BY a.created_at DESC
		 LIMIT $2 OFFSET $3`,
		orgID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := []Log{}
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorEmail, &l.ResourceType, &l.ResourceID, &l.Action, &l.IP, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}
