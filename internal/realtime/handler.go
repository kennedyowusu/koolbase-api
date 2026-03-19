package realtime

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/kennedyowusu/hatchway-api/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Handler struct {
	hub        *Hub
	authorizer ProjectAuthorizer
	authSvc    *auth.Service
}

func NewHandler(hub *Hub, authorizer ProjectAuthorizer, authSvc *auth.Service) *Handler {
	return &Handler{hub: hub, authorizer: authorizer, authSvc: authSvc}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	raw, err := h.authSvc.ValidateSession(context.Background(), token)
	if err != nil || raw == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, ok := raw.(*auth.User)
	if !ok || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(h.hub, conn, user.ID, user.OrgID)
	h.hub.register <- client

	go client.WritePump()
	go client.ReadPump(h.authorizer)
}
