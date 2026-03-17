package projectauth

import (
	"context"

	"github.com/kennedyowusu/hatchway-api/platform/email"
)

type MailerAdapter struct {
	provider email.Provider
}

func NewMailerAdapter(p email.Provider) *MailerAdapter {
	return &MailerAdapter{provider: p}
}

func (m *MailerAdapter) SendEmail(ctx context.Context, to, subject, html string) error {
	return m.provider.Send(ctx, email.Message{
		To:      to,
		Subject: subject,
		HTML:    html,
	})
}
