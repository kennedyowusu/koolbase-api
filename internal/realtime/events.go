package realtime

import "time"

type EventType string

const (
	EventSubscribed    EventType = "subscribed"
	EventUnsubscribed  EventType = "unsubscribed"
	EventRecordCreated EventType = "db.record.created"
	EventRecordUpdated EventType = "db.record.updated"
	EventRecordDeleted EventType = "db.record.deleted"
	EventError         EventType = "error"
)

type Event struct {
	Type      EventType   `json:"type"`
	Channel   string      `json:"channel,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type RecordEventPayload struct {
	ProjectID  string      `json:"project_id"`
	Collection string      `json:"collection"`
	RecordID   string      `json:"record_id,omitempty"`
	Record     interface{} `json:"record,omitempty"`
}

type SubscribeMessage struct {
	Action     string `json:"action"`
	ProjectID  string `json:"project_id"`
	Collection string `json:"collection"`
}

func ChannelKey(projectID, collection string) string {
	return projectID + ":" + collection
}
