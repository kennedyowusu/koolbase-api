package realtime

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 8192
)

type Client struct {
	id       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	orgID    string
	channels map[string]bool
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, orgID string) *Client {
	return &Client{
		id:       uuid.NewString(),
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		orgID:    orgID,
		channels: map[string]bool{},
	}
}

func (c *Client) sendEvent(event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	select {
	case c.send <- payload:
	default:
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[realtime] write error client=%s: %v", c.id, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump(authorizer ProjectAuthorizer) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("[realtime] read error client=%s: %v", c.id, err)
			break
		}

		var msg SubscribeMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendEvent(Event{
				Type:      EventError,
				Payload:   map[string]string{"message": "invalid message format"},
				Timestamp: time.Now().UTC(),
			})
			continue
		}

		if msg.ProjectID == "" || msg.Collection == "" {
			c.sendEvent(Event{
				Type:      EventError,
				Payload:   map[string]string{"message": "project_id and collection are required"},
				Timestamp: time.Now().UTC(),
			})
			continue
		}

		ok, err := authorizer.AuthorizeProject(msg.ProjectID, c.orgID)
		if err != nil || !ok {
			c.sendEvent(Event{
				Type:      EventError,
				Payload:   map[string]string{"message": "access denied"},
				Timestamp: time.Now().UTC(),
			})
			continue
		}

		channel := ChannelKey(msg.ProjectID, msg.Collection)

		switch msg.Action {
		case "subscribe":
			c.hub.Subscribe(c, channel)
			log.Printf("[realtime] client=%s subscribed to %s", c.id, channel)
			c.sendEvent(Event{
				Type:      EventSubscribed,
				Channel:   channel,
				Timestamp: time.Now().UTC(),
			})
		case "unsubscribe":
			c.hub.Unsubscribe(c, channel)
			log.Printf("[realtime] client=%s unsubscribed from %s", c.id, channel)
			c.sendEvent(Event{
				Type:      EventUnsubscribed,
				Channel:   channel,
				Timestamp: time.Now().UTC(),
			})
		default:
			c.sendEvent(Event{
				Type:      EventError,
				Payload:   map[string]string{"message": "unsupported action"},
				Timestamp: time.Now().UTC(),
			})
		}
	}
}
