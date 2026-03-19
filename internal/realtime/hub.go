package realtime

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type ProjectAuthorizer interface {
	AuthorizeProject(projectID, orgID string) (bool, error)
}

type subscription struct {
	client  *Client
	channel string
}

type broadcastMessage struct {
	channel string
	payload []byte
}

type Hub struct {
	mu          sync.RWMutex
	clients     map[*Client]bool
	channels    map[string]map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	subscribe   chan subscription
	unsubscribe chan subscription
	broadcast   chan broadcastMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:     map[*Client]bool{},
		channels:    map[string]map[*Client]bool{},
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		subscribe:   make(chan subscription),
		unsubscribe: make(chan subscription),
		broadcast:   make(chan broadcastMessage, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[realtime] client=%s connected (total=%d)", client.id, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				for channel := range client.channels {
					if subs, exists := h.channels[channel]; exists {
						delete(subs, client)
						if len(subs) == 0 {
							delete(h.channels, channel)
						}
					}
				}
				close(client.send)
				log.Printf("[realtime] client=%s disconnected (total=%d)", client.id, len(h.clients))
			}
			h.mu.Unlock()

		case sub := <-h.subscribe:
			h.mu.Lock()
			if _, ok := h.channels[sub.channel]; !ok {
				h.channels[sub.channel] = map[*Client]bool{}
			}
			h.channels[sub.channel][sub.client] = true
			sub.client.channels[sub.channel] = true
			h.mu.Unlock()

		case sub := <-h.unsubscribe:
			h.mu.Lock()
			if subs, ok := h.channels[sub.channel]; ok {
				delete(subs, sub.client)
				if len(subs) == 0 {
					delete(h.channels, sub.channel)
				}
			}
			delete(sub.client.channels, sub.channel)
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			subs := h.channels[msg.channel]
			h.mu.RUnlock()

			if len(subs) == 0 {
				continue
			}

			log.Printf("[realtime] broadcast channel=%s clients=%d", msg.channel, len(subs))

			for client := range subs {
				select {
				case client.send <- msg.payload:
				default:
					log.Printf("[realtime] client=%s send buffer full, dropping", client.id)
					h.unregister <- client
				}
			}
		}
	}
}

func (h *Hub) Subscribe(client *Client, channel string) {
	h.subscribe <- subscription{client: client, channel: channel}
}

func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.unsubscribe <- subscription{client: client, channel: channel}
}

func (h *Hub) Broadcast(channel string, event Event) {
	event.Channel = channel
	event.Timestamp = time.Now().UTC()

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[realtime] marshal error: %v", err)
		return
	}

	select {
	case h.broadcast <- broadcastMessage{channel: channel, payload: payload}:
	default:
		log.Printf("[realtime] broadcast channel full, dropping event for channel=%s", channel)
	}
}

func (h *Hub) PublishRecordCreated(projectID, collection string, record interface{}) {
	h.Broadcast(ChannelKey(projectID, collection), Event{
		Type: EventRecordCreated,
		Payload: RecordEventPayload{
			ProjectID:  projectID,
			Collection: collection,
			Record:     record,
		},
	})
}

func (h *Hub) PublishRecordUpdated(projectID, collection string, record interface{}) {
	h.Broadcast(ChannelKey(projectID, collection), Event{
		Type: EventRecordUpdated,
		Payload: RecordEventPayload{
			ProjectID:  projectID,
			Collection: collection,
			Record:     record,
		},
	})
}

func (h *Hub) PublishRecordDeleted(projectID, collection, recordID string) {
	h.Broadcast(ChannelKey(projectID, collection), Event{
		Type: EventRecordDeleted,
		Payload: RecordEventPayload{
			ProjectID:  projectID,
			Collection: collection,
			RecordID:   recordID,
		},
	})
}
