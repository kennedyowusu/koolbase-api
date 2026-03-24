package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kennedyowusu/hatchway-api/platform/email"
	"github.com/rs/zerolog/log"
)

const alertThreshold = 0.80
const clearThreshold = 0.70

type AlertChecker struct {
	db     *pgxpool.Pool
	repo   *PGRepository
	mailer email.Provider
	appURL string
}

func NewAlertChecker(db *pgxpool.Pool, repo *PGRepository, mailer email.Provider, appURL string) *AlertChecker {
	return &AlertChecker{db: db, repo: repo, mailer: mailer, appURL: appURL}
}

type orgRow struct {
	id    string
	email string
	name  string
	plan  string
}

func (a *AlertChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	log.Info().Msg("usage alert checker started")
	a.check(ctx)
	for {
		select {
		case <-ticker.C:
			a.check(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (a *AlertChecker) check(ctx context.Context) {
	rows, err := a.db.Query(ctx,
		`SELECT o.id, u.email, o.name, o.plan
		 FROM organizations o
		 JOIN users u ON u.org_id = o.id AND u.role = 'owner'
		 LIMIT 1000`)
	if err != nil {
		log.Error().Err(err).Msg("usage alert: failed to fetch orgs")
		return
	}
	defer rows.Close()

	var orgs []orgRow
	for rows.Next() {
		var o orgRow
		if err := rows.Scan(&o.id, &o.email, &o.name, &o.plan); err != nil {
			continue
		}
		orgs = append(orgs, o)
	}
	for _, org := range orgs {
		a.checkOrg(ctx, org)
	}
}

func (a *AlertChecker) checkOrg(ctx context.Context, org orgRow) {
	resources := map[string]func() (int, error){
		"environments": func() (int, error) { return a.repo.CountEnvironments(ctx, org.id) },
		"flags":        func() (int, error) { return a.repo.CountFlags(ctx, org.id) },
		"configs":      func() (int, error) { return a.repo.CountConfigs(ctx, org.id) },
		"members":      func() (int, error) { return a.repo.CountMembers(ctx, org.id) },
	}

	for resource, countFn := range resources {
		limit := GetLimit(org.plan, resource)
		if limit <= 0 {
			continue
		}
		current, err := countFn()
		if err != nil {
			log.Error().Err(err).Str("org", org.id).Str("resource", resource).Msg("usage alert: count error")
			continue
		}
		pct := float64(current) / float64(limit)

		if pct >= alertThreshold {
			var exists bool
			err := a.db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM org_usage_alerts WHERE org_id = $1 AND resource = $2)`,
				org.id, resource,
			).Scan(&exists)
			if err != nil || exists {
				continue
			}
			if err := a.sendAlert(ctx, org, resource, current, limit); err != nil {
				log.Error().Err(err).Str("org", org.id).Str("resource", resource).Msg("usage alert: send failed")
				continue
			}
			_, err = a.db.Exec(ctx,
				`INSERT INTO org_usage_alerts (org_id, resource) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				org.id, resource,
			)
			if err != nil {
				log.Error().Err(err).Msg("usage alert: failed to record alert")
			}
			log.Info().Str("org", org.id).Str("resource", resource).
				Int("current", current).Int("limit", limit).
				Msg("usage alert sent")

		} else if pct < clearThreshold {
			_, _ = a.db.Exec(ctx,
				`DELETE FROM org_usage_alerts WHERE org_id = $1 AND resource = $2`,
				org.id, resource,
			)
		}
	}
}

func (a *AlertChecker) sendAlert(ctx context.Context, org orgRow, resource string, current, limit int) error {
	pct := int(float64(current) / float64(limit) * 100)
	subject := fmt.Sprintf("You've used %d%% of your %s limit — Koolbase", pct, resource)
	html := alertEmailHTML(org.name, resource, current, limit, pct, a.appURL)
	return a.mailer.Send(ctx, email.Message{
		To:      org.email,
		Subject: subject,
		HTML:    html,
	})
}
