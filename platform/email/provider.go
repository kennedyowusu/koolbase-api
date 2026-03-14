package email

import "context"

type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

type Provider interface {
	Send(ctx context.Context, msg Message) error
}
