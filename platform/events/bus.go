package events

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type EventType string

const (
	FlagCreated          EventType = "flag.created"
	FlagUpdated          EventType = "flag.updated"
	FlagDeleted          EventType = "flag.deleted"
	ConfigCreated        EventType = "config.created"
	ConfigUpdated        EventType = "config.updated"
	ConfigDeleted        EventType = "config.deleted"
	VersionPolicyUpdated EventType = "version_policy.updated"
	UserSignedUp         EventType = "user.signed_up"
	UserVerifiedEmail    EventType = "user.verified_email"
	UserRequestedReset   EventType = "user.requested_password_reset"
	UserResetPassword    EventType = "user.reset_password"
	OrgCreated           EventType = "org.created"
	ProjectCreated       EventType = "project.created"
	EnvironmentCreated   EventType = "environment.created"
)

type Event struct {
	Type    EventType
	Payload any
}

type SnapshotPayload struct {
	EnvironmentID string
}

type UserSignedUpPayload struct {
	UserID string
	Email  string
	OrgID  string
	Token  string
}

type UserRequestedResetPayload struct {
	UserID string
	Email  string
	Token  string
}

type Handler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

func New() *Bus {
	return &Bus{handlers: make(map[EventType][]Handler)}
}

var Default = New()

func (b *Bus) Subscribe(eventType EventType, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], h)
}

func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[e.Type]))
	copy(handlers, b.handlers[e.Type])
	b.mu.RUnlock()

	for _, h := range handlers {
		h := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().
						Str("event", string(e.Type)).
						Interface("panic", r).
						Msg("event handler panicked")
				}
			}()
			h(e)
		}()
	}
}
