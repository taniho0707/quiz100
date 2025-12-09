package websocket

import (
	"encoding/json"
	"fmt"
	"log"
)

// HubManager provides high-level WebSocket hub management operations
type HubManager struct {
	hub *Hub
}

// NewHubManager creates a new HubManager instance
func NewHubManager(hub *Hub) *HubManager {
	return &HubManager{
		hub: hub,
	}
}

// BroadcastMessage sends a message to all connected clients
func (hm *HubManager) BroadcastMessage(msgType MessageType, data any) error {
	message := NewTypedMessage(msgType, data)
	return hm.broadcastTypedMessage(message)
}

// BroadcastToType sends a message to clients of a specific type
func (hm *HubManager) BroadcastToType(msgType MessageType, data any, clientType ClientType) error {
	message := NewTypedMessageWithTarget(msgType, data, clientType)
	return hm.broadcastTypedMessageToType(message, clientType)
}

// BroadcastToUser sends a message to a specific user
func (hm *HubManager) BroadcastToUser(msgType MessageType, data any, userID int) error {
	message := NewTypedMessage(msgType, data)
	return hm.broadcastTypedMessageToUser(message, userID)
}

// BroadcastEventStarted sends event started message to all clients
func (hm *HubManager) BroadcastEventStarted(eventData any) error {
	return hm.BroadcastMessage(MessageEventStarted, eventData)
}

// BroadcastTitleDisplay sends title display message to screen clients
func (hm *HubManager) BroadcastTitleDisplay(titleData any) error {
	return hm.BroadcastToType(MessageTitleDisplay, titleData, ClientTypeScreen)
}

// BroadcastTeamAssignment sends team assignment message to all clients
func (hm *HubManager) BroadcastTeamAssignment(teamsData any) error {
	return hm.BroadcastMessage(MessageTeamAssignment, teamsData)
}

// BroadcastQuestionStart sends question start message to all clients
func (hm *HubManager) BroadcastQuestionStart(questionData any, questionAndAnswerData any) error {
	if err := hm.BroadcastToType(MessageQuestionStart, questionAndAnswerData, ClientTypeAdmin); err != nil {
		return err
	}
	if err := hm.BroadcastToType(MessageQuestionStart, questionData, ClientTypeScreen); err != nil {
		return err
	}
	return hm.BroadcastToType(MessageQuestionStart, questionData, ClientTypeParticipant)
}

// BroadcastQuestionEnd sends question end message to all clients
func (hm *HubManager) BroadcastQuestionEnd(endData any) error {
	return hm.BroadcastMessage(MessageQuestionEnd, endData)
}

// BroadcastCountdown sends countdown message to all clients
func (hm *HubManager) BroadcastCountdown(countdownData any) error {
	// Send to admin clients
	if err := hm.BroadcastToType(MessageCountdown, countdownData, ClientTypeAdmin); err != nil {
		return err
	}
	// Send to screen clients
	return hm.BroadcastToType(MessageCountdown, countdownData, ClientTypeScreen)
}

// BroadcastAnswerStats sends answer statistics to screen clients
func (hm *HubManager) BroadcastAnswerStats(statsData any) error {
	// Send to admin clients
	if err := hm.BroadcastToType(MessageAnswerStats, statsData, ClientTypeAdmin); err != nil {
		return err
	}
	// Send to screen clients
	return hm.BroadcastToType(MessageAnswerStats, statsData, ClientTypeScreen)
}

// BroadcastAnswerReveal sends answer reveal to all clients
func (hm *HubManager) BroadcastAnswerReveal(revealData any) error {
	return hm.BroadcastMessage(MessageAnswerReveal, revealData)
}

// BroadcastFinalResults sends final results to all clients
func (hm *HubManager) BroadcastFinalResults(resultsData any) error {
	return hm.BroadcastMessage(MessageFinalResults, resultsData)
}

// BroadcastCelebration sends celebration message to screen clients
func (hm *HubManager) BroadcastCelebration(celebrationData any) error {
	return hm.BroadcastToType(MessageCelebration, celebrationData, ClientTypeScreen)
}

// BroadcastUserJoined sends user joined notification to admin and screen
func (hm *HubManager) BroadcastUserJoined(userData any) error {
	// Send to admin clients
	if err := hm.BroadcastToType(MessageUserJoined, userData, ClientTypeAdmin); err != nil {
		return err
	}
	// Send to screen clients
	return hm.BroadcastToType(MessageUserJoined, userData, ClientTypeScreen)
}

// BroadcastUserLeft sends user left notification to admin and screen
func (hm *HubManager) BroadcastUserLeft(userData any) error {
	// Send to admin clients
	if err := hm.BroadcastToType(MessageUserLeft, userData, ClientTypeAdmin); err != nil {
		return err
	}
	// Send to screen clients
	return hm.BroadcastToType(MessageUserLeft, userData, ClientTypeScreen)
}

// BroadcastAnswerReceived sends answer received notification to admin
func (hm *HubManager) BroadcastAnswerReceived(answerData any) error {
	return hm.BroadcastToType(MessageAnswerReceived, answerData, ClientTypeAdmin)
}

// BroadcastEmojiReaction sends emoji reaction to screen clients
func (hm *HubManager) BroadcastEmojiReaction(emojiData any) error {
	return hm.BroadcastToType(MessageEmojiReaction, emojiData, ClientTypeScreen)
}

// BroadcastTeamMemberAdded sends team member added notification
func (hm *HubManager) BroadcastTeamMemberAdded(teamData any) error {
	// Send to admin clients
	if err := hm.BroadcastToType(MessageTeamMemberAdded, teamData, ClientTypeAdmin); err != nil {
		return err
	}
	// Send to screen clients
	return hm.BroadcastToType(MessageTeamMemberAdded, teamData, ClientTypeScreen)
}

// BroadcastStateChanged sends state change notification to all clients
func (hm *HubManager) BroadcastStateChanged(stateData any) error {
	return hm.BroadcastMessage(MessageStateChanged, stateData)
}

// BroadcastDatabaseReset sends database reset notification to all clients
func (hm *HubManager) BroadcastDatabaseReset(resetData any) error {
	return hm.BroadcastMessage(MessageDatabaseReset, resetData)
}

// GetClientCount returns the count of clients by type
func (hm *HubManager) GetClientCount() map[string]int {
	hubCounts := hm.hub.GetClientCount()
	stringCounts := make(map[string]int)

	stringCounts["participant"] = hubCounts[ClientTypeParticipant]
	stringCounts["admin"] = hubCounts[ClientTypeAdmin]
	stringCounts["screen"] = hubCounts[ClientTypeScreen]

	return stringCounts
}

// GetStatistics returns hub statistics
func (hm *HubManager) GetStatistics() map[string]any {
	stats := hm.hub.GetStatistics()

	// Convert the struct to map[string]any
	result := make(map[string]any)
	result["active_connections"] = stats.ActiveConnections
	result["total_connected"] = stats.TotalConnected
	result["messages_sent"] = stats.MessagesSent
	result["uptime_seconds"] = stats.UptimeSeconds
	result["average_connections"] = stats.AverageConnections
	result["client_details"] = stats.ClientDetails

	return result
}

// GetClientsByType returns clients of a specific type
func (hm *HubManager) GetClientsByType(clientType ClientType) []*Client {
	return hm.hub.GetClientsByType(clientType)
}

// Private helper methods

// broadcastTypedMessage broadcasts a typed message to all clients
func (hm *HubManager) broadcastTypedMessage(message TypedMessage) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling typed message: %v", err)
		return err
	}

	hm.hub.Broadcast <- messageBytes
	return nil
}

// broadcastTypedMessageToType broadcasts a typed message to clients of a specific type
func (hm *HubManager) broadcastTypedMessageToType(message TypedMessage, clientType ClientType) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling typed message: %v", err)
		return err
	}

	hm.hub.BroadcastToType(messageBytes, clientType)
	return nil
}

// broadcastTypedMessageToUser broadcasts a typed message to a specific user
func (hm *HubManager) broadcastTypedMessageToUser(message TypedMessage, userID int) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling typed message: %v", err)
		return err
	}

	hm.hub.BroadcastToUser(messageBytes, userID)
	return nil
}

// SendDeprecationWarning sends a warning about deprecated message types
func (hm *HubManager) SendDeprecationWarning(msgType MessageType) {
	if IsDeprecatedMessageType(msgType) {
		log.Printf("Warning: Using deprecated message type '%s'. Consider migrating to newer alternatives.", msgType)
	}
}

// ValidateMessageType validates a message type before sending
func (hm *HubManager) ValidateMessageType(msgType MessageType) error {
	if !IsValidMessageType(msgType) {
		return fmt.Errorf("invalid message type: %s", msgType)
	}

	// Log deprecation warning
	hm.SendDeprecationWarning(msgType)

	return nil
}
