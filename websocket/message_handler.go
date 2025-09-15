package websocket

import (
	"encoding/json"
	"log"
)

// MessageHandler handles WebSocket message processing
type MessageHandler struct {
	pingManager *PingManager
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(pingManager *PingManager) *MessageHandler {
	return &MessageHandler{
		pingManager: pingManager,
	}
}

// HandleMessage processes incoming WebSocket messages
func (mh *MessageHandler) HandleMessage(client *Client, message []byte) {
	var msg struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	switch msg.Type {
	case "pong":
		mh.handlePongMessage(client, msg.Data)
	default:
		// Other message types can be added here in the future
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handlePongMessage processes pong responses from participants
func (mh *MessageHandler) handlePongMessage(client *Client, data json.RawMessage) {
	var pongData struct {
		PingID string `json:"ping_id"`
	}

	if err := json.Unmarshal(data, &pongData); err != nil {
		log.Printf("Error unmarshaling pong data: %v", err)
		return
	}

	if pongData.PingID == "" {
		log.Printf("Missing ping_id in pong response from user %d", client.UserID)
		return
	}

	// Only process pong from participant clients
	if client.Type != ClientTypeParticipant {
		log.Printf("Ignoring pong from non-participant client: %s", client.Type)
		return
	}

	// Forward to ping manager for processing
	mh.pingManager.HandlePong(pongData.PingID, client.UserID)
}
