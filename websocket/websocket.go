package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ClientType string

const (
	ClientTypeParticipant ClientType = "participant"
	ClientTypeAdmin       ClientType = "admin"
	ClientTypeScreen      ClientType = "screen"
)

type Client struct {
	Conn        *websocket.Conn
	Type        ClientType
	UserID      int
	SessionID   string
	Send        chan []byte
	ConnectedAt time.Time
}

type Hub struct {
	Clients        map[*Client]bool
	Broadcast      chan []byte
	Register       chan *Client
	Unregister     chan *Client
	mutex          sync.RWMutex
	StartTime      time.Time
	TotalConnected int64
	MessagesSent   int64
}

type Message struct {
	Type   string     `json:"type"`
	Data   any        `json:"data"`
	UserID int        `json:"user_id,omitempty"`
	Target ClientType `json:"target,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		StartTime:  time.Now(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mutex.Lock()
			h.Clients[client] = true
			h.TotalConnected++
			h.mutex.Unlock()
			log.Printf("Client registered: %s (UserID: %d)", client.Type, client.UserID)

		case client := <-h.Unregister:
			h.mutex.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				log.Printf("Client unregistered: %s (UserID: %d)", client.Type, client.UserID)
			}
			h.mutex.Unlock()

		case message := <-h.Broadcast:
			h.broadcastMessage(message)
			h.mutex.Lock()
			h.MessagesSent++
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) broadcastMessage(message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.Clients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.Clients, client)
		}
	}
}

func (h *Hub) BroadcastToType(message []byte, clientType ClientType) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.Clients {
		if client.Type == clientType {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.Clients, client)
			}
		}
	}
}

func (h *Hub) BroadcastExceptType(message []byte, excludeType ClientType) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.Clients {
		if client.Type != excludeType {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.Clients, client)
			}
		}
	}
}

func (h *Hub) GetClientsByType(clientType ClientType) []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var clients []*Client
	for client := range h.Clients {
		if client.Type == clientType {
			clients = append(clients, client)
		}
	}
	return clients
}

func (h *Hub) GetClientCount() map[ClientType]int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	counts := map[ClientType]int{
		ClientTypeParticipant: 0,
		ClientTypeAdmin:       0,
		ClientTypeScreen:      0,
	}

	for client := range h.Clients {
		counts[client.Type]++
	}

	return counts
}

type HubStatistics struct {
	ActiveConnections  map[ClientType]int `json:"active_connections"`
	TotalConnected     int64              `json:"total_connected"`
	MessagesSent       int64              `json:"messages_sent"`
	UptimeSeconds      int64              `json:"uptime_seconds"`
	AverageConnections float64            `json:"average_connections"`
	ClientDetails      []ClientInfo       `json:"client_details"`
}

type ClientInfo struct {
	Type           ClientType `json:"type"`
	UserID         int        `json:"user_id"`
	SessionID      string     `json:"session_id"`
	ConnectedAt    time.Time  `json:"connected_at"`
	ConnectionTime int64      `json:"connection_time_seconds"`
}

func (h *Hub) GetStatistics() HubStatistics {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	uptime := time.Since(h.StartTime)
	clientCount := len(h.Clients)
	averageConnections := float64(h.TotalConnected) / uptime.Hours()

	clientDetails := make([]ClientInfo, 0, clientCount)
	for client := range h.Clients {
		clientDetails = append(clientDetails, ClientInfo{
			Type:           client.Type,
			UserID:         client.UserID,
			SessionID:      client.SessionID,
			ConnectedAt:    client.ConnectedAt,
			ConnectionTime: int64(time.Since(client.ConnectedAt).Seconds()),
		})
	}

	return HubStatistics{
		ActiveConnections:  h.GetClientCount(),
		TotalConnected:     h.TotalConnected,
		MessagesSent:       h.MessagesSent,
		UptimeSeconds:      int64(uptime.Seconds()),
		AverageConnections: averageConnections,
		ClientDetails:      clientDetails,
	}
}

func (c *Client) ReadPump(hub *Hub, onMessage func(*Client, []byte)) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if onMessage != nil {
			onMessage(c, message)
		}
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for {
		message, ok := <-c.Send
		if !ok {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, clientType ClientType, userID int, sessionID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		Conn:        conn,
		Type:        clientType,
		UserID:      userID,
		SessionID:   sessionID,
		Send:        make(chan []byte, 256),
		ConnectedAt: time.Now(),
	}

	hub.Register <- client

	go client.WritePump()
	go client.ReadPump(hub, nil)
}
