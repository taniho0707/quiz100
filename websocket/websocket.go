package websocket

import (
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"quiz100/models"
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

	// State synchronization
	ClientStates   map[*Client]*ClientState
	LastEventState *EventSyncData
	StateSync      chan *StateSyncRequest
}

// ClientState tracks individual client synchronization state
type ClientState struct {
	LastSyncTime    time.Time
	SyncVersion     int
	IsInitialized   bool
	LastEventState  string
	LastQuestionNum int
}

// EventSyncData contains all data needed for state synchronization
type EventSyncData struct {
	EventState      string           `json:"event_state"`
	QuestionNumber  int              `json:"question_number"`
	QuestionData    models.Question  `json:"question"`
	TeamData        []any            `json:"team,omitempty"`             // only sending to admin
	ParticipantData []map[string]any `json:"participant_data,omitempty"` // only sending to admin
	AnswerData      map[string]any   `json:"answer_data,omitempty"`      // user_id(string) -> answer_index
	// SyncVersion     int             `json:"sync_version"`
	// Timestamp       time.Time       `json:"timestamp"`
}

// StateSyncRequest represents a state synchronization request
type StateSyncRequest struct {
	Client      *Client
	SyncType    string // "initial", "reconnect", "periodic"
	RequestData map[string]interface{}
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
		Clients:      make(map[*Client]bool),
		Broadcast:    make(chan []byte),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		StartTime:    time.Now(),
		ClientStates: make(map[*Client]*ClientState),
		StateSync:    make(chan *StateSyncRequest, 100),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mutex.Lock()
			h.Clients[client] = true
			h.TotalConnected++
			// Initialize client state for all client types
			h.ClientStates[client] = &ClientState{
				LastSyncTime:    time.Now(),
				SyncVersion:     0,
				IsInitialized:   false,
				LastEventState:  "",
				LastQuestionNum: 0,
			}
			h.mutex.Unlock()
			log.Printf("Client registered: %s (UserID: %d)", client.Type, client.UserID)

			// Trigger initial sync for all client types
			go func() {
				time.Sleep(100 * time.Millisecond) // Wait for connection establishment
				h.StateSync <- &StateSyncRequest{
					Client:   client,
					SyncType: "initial",
				}
			}()

		case client := <-h.Unregister:
			h.mutex.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				delete(h.ClientStates, client) // Clean up client state
				close(client.Send)
				log.Printf("Client unregistered: %s (UserID: %d)", client.Type, client.UserID)
			}
			h.mutex.Unlock()

		case message := <-h.Broadcast:
			h.broadcastMessage(message)
			h.mutex.Lock()
			h.MessagesSent++
			h.mutex.Unlock()

		case syncRequest := <-h.StateSync:
			h.handleStateSyncRequest(syncRequest)
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

func (h *Hub) BroadcastToUser(message []byte, userID int) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.Clients {
		if client.UserID == userID {
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

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, clientType ClientType, userID int, sessionID string, messageHandler *MessageHandler) {
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

	// Set up message handling function
	var onMessage func(*Client, []byte)
	if messageHandler != nil {
		onMessage = messageHandler.HandleMessage
	}

	go client.WritePump()
	go client.ReadPump(hub, onMessage)
}

// State synchronization methods

// handleStateSyncRequest processes state synchronization requests
func (h *Hub) handleStateSyncRequest(request *StateSyncRequest) {
	h.mutex.Lock()
	clientState, exists := h.ClientStates[request.Client]
	if !exists {
		h.mutex.Unlock()
		return
	}

	// Get latest event state
	// eventState := h.LastEventState
	h.mutex.Unlock()

	if h.LastEventState == nil {
		log.Printf("No event state available for sync, skipping client %s (UserID: %d)", request.Client.Type, request.Client.UserID)
		return
	}

	// Check if sync is needed
	if request.SyncType == "initial" ||
		clientState.LastEventState != h.LastEventState.EventState ||
		clientState.LastQuestionNum != h.LastEventState.QuestionNumber ||
		!clientState.IsInitialized {

		var reducedEventState EventSyncData
		reducedEventState.EventState = h.LastEventState.EventState
		reducedEventState.QuestionNumber = h.LastEventState.QuestionNumber
		reducedEventState.QuestionData = models.Question{
			Type:    h.LastEventState.QuestionData.Type,
			Text:    h.LastEventState.QuestionData.Text,
			Image:   h.LastEventState.QuestionData.Image,
			Choices: h.LastEventState.QuestionData.Choices,
			Correct: h.LastEventState.QuestionData.Correct,
		}
		reducedEventState.TeamData = h.LastEventState.TeamData
		reducedEventState.ParticipantData = h.LastEventState.ParticipantData
		reducedEventState.AnswerData = make(map[string]any)
		maps.Copy(reducedEventState.AnswerData, h.LastEventState.AnswerData)

		switch request.Client.Type {
		case ClientTypeParticipant:
			reducedEventState.QuestionData.Correct = 0 // invalid data
			reducedEventState.TeamData = nil
			reducedEventState.ParticipantData = nil
			// reducedEventState.AnswerData から該当ユーザーのみのデータに絞る
			// FIXME:
			for _, v := range h.LastEventState.ParticipantData {
				if v["id"] == request.Client.UserID {
					reducedEventState.ParticipantData = append(reducedEventState.ParticipantData, v)
				}
			}
		case ClientTypeScreen:
			reducedEventState.QuestionData.Correct = 0 // invalid data
		case ClientTypeAdmin:
			// do nothing
		default:
			return
		}
		h.sendInitialSync(request.Client, &reducedEventState)

		// Update client state
		h.mutex.Lock()
		clientState.LastSyncTime = time.Now()
		// clientState.SyncVersion = eventState.SyncVersion
		clientState.IsInitialized = true
		clientState.LastEventState = h.LastEventState.EventState
		clientState.LastQuestionNum = h.LastEventState.QuestionNumber
		h.mutex.Unlock()

		log.Printf("State sync completed for client %s (UserID: %d, SyncType: %s)", request.Client.Type, request.Client.UserID, request.SyncType)
	}
}

// sendInitialSync sends initial synchronization data to a client
func (h *Hub) sendInitialSync(client *Client, eventState *EventSyncData) {
	message := Message{
		Type: string(MessageInitialSync),
		Data: eventState,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling initial sync data: %v", err)
		return
	}

	select {
	case client.Send <- jsonData:
		log.Printf("Initial sync sent to client %d", client.UserID)
	default:
		log.Printf("Failed to send initial sync to client %d (channel full)", client.UserID)
	}
}

// UpdateEventState updates the global event state for synchronization
func (h *Hub) UpdateEventState(eventState *EventSyncData) {
	h.mutex.Lock()
	// eventState.SyncVersion++
	// eventState.Timestamp = time.Now()
	h.LastEventState = eventState
	h.mutex.Unlock()

	log.Printf("Event state updated: %s (Question: %d)", eventState.EventState, eventState.QuestionNumber)
}

// RequestStateSync allows external components to request state synchronization
func (h *Hub) RequestStateSync(client *Client, syncType string) {
	select {
	case h.StateSync <- &StateSyncRequest{
		Client:   client,
		SyncType: syncType,
	}:
	default:
		log.Printf("StateSync channel full, dropping sync request for client %s (UserID: %d)", client.Type, client.UserID)
	}
}

// StartPeriodicSync starts a goroutine that periodically checks for clients needing sync
func (h *Hub) StartPeriodicSync(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			h.checkAndSyncOutdatedClients()
		}
	}()
	log.Printf("Periodic sync started with interval: %v", interval)
}

// checkAndSyncOutdatedClients checks for clients that need state synchronization
func (h *Hub) checkAndSyncOutdatedClients() {
	h.mutex.RLock()
	currentEventState := h.LastEventState

	var outdatedClients []*Client
	for client, state := range h.ClientStates {
		if client.Type == ClientTypeParticipant {
			// Check if client needs sync (state mismatch or too old)
			if currentEventState != nil &&
				(state.LastEventState != currentEventState.EventState ||
					state.LastQuestionNum != currentEventState.QuestionNumber ||
					time.Since(state.LastSyncTime) > 30*time.Second) {
				outdatedClients = append(outdatedClients, client)
			}
		}
	}
	h.mutex.RUnlock()

	// Trigger sync for outdated clients
	for _, client := range outdatedClients {
		h.RequestStateSync(client, "periodic")
	}

	if len(outdatedClients) > 0 {
		log.Printf("Triggered periodic sync for %d outdated clients", len(outdatedClients))
	}
}

// GetClientSyncStatus returns synchronization status for all clients
func (h *Hub) GetClientSyncStatus() map[int]*ClientState {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	status := make(map[int]*ClientState)
	for client, state := range h.ClientStates {
		if client.Type == ClientTypeParticipant {
			// Create a copy to avoid race conditions
			status[client.UserID] = &ClientState{
				LastSyncTime:    state.LastSyncTime,
				SyncVersion:     state.SyncVersion,
				IsInitialized:   state.IsInitialized,
				LastEventState:  state.LastEventState,
				LastQuestionNum: state.LastQuestionNum,
			}
		}
	}

	return status
}
