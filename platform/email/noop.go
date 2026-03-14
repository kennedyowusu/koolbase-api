package email

import (
	"context"

	"github.com/rs/zerolog/log"
)

type NoopProvider struct{}

func (p *NoopProvider) Send(_ context.Context, msg Message) error {
	log.Info().
		Str("to", msg.To).
		Str("subject", msg.Subject).
		Msg("[noop email] would have sent email")
	return nil
}
