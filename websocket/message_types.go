package websocket

// MessageType represents WebSocket message types
type MessageType string

// WebSocket message type constants
const (
	// Event messages
	MessageEventStarted   MessageType = "event_started"
	MessageTitleDisplay   MessageType = "title_display"
	MessageTeamAssignment MessageType = "team_assignment"
	MessageQuestionStart  MessageType = "question_start"
	MessageQuestionEnd    MessageType = "question_end"
	MessageFinalResults   MessageType = "final_results"
	MessageCelebration    MessageType = "celebration"

	// User interaction messages
	MessageUserJoined      MessageType = "user_joined"
	MessageUserLeft        MessageType = "user_left"
	MessageAnswerReceived  MessageType = "answer_received"
	MessageEmojiReaction   MessageType = "emoji"
	MessageTeamMemberAdded MessageType = "team_member_added"

	// Quiz progress messages
	MessageCountdown    MessageType = "countdown"
	MessageAnswerStats  MessageType = "answer_stats"
	MessageAnswerReveal MessageType = "answer_reveal"
	MessageStateChanged MessageType = "state_changed"

	// Connectivity messages
	MessagePing       MessageType = "ping"
	MessagePong       MessageType = "pong"
	MessagePingResult MessageType = "ping_result"

	// State synchronization messages
	MessageInitialSync  MessageType = "initial_sync"
	MessageStateSync    MessageType = "state_sync"
	MessageSyncRequest  MessageType = "sync_request"
	MessageSyncComplete MessageType = "sync_complete"

	// Legacy/deprecated messages (to be removed)
	MessageTimeAlert MessageType = "time_alert" // DEPRECATED: use countdown instead
)

// AllMessageTypes returns all valid message types
func AllMessageTypes() []MessageType {
	return []MessageType{
		MessageEventStarted,
		MessageTitleDisplay,
		MessageTeamAssignment,
		MessageQuestionStart,
		MessageQuestionEnd,
		MessageFinalResults,
		MessageCelebration,
		MessageUserJoined,
		MessageUserLeft,
		MessageAnswerReceived,
		MessageEmojiReaction,
		MessageTeamMemberAdded,
		MessageCountdown,
		MessageAnswerStats,
		MessageAnswerReveal,
		MessageStateChanged,
		MessagePing,
		MessagePong,
		MessagePingResult,
		MessageInitialSync,
		MessageStateSync,
		MessageSyncRequest,
		MessageSyncComplete,
		MessageTimeAlert, // DEPRECATED
	}
}

// MessageTypeToString converts MessageType to string
func MessageTypeToString(msgType MessageType) string {
	return string(msgType)
}

// StringToMessageType converts string to MessageType with validation
func StringToMessageType(msgTypeStr string) (MessageType, bool) {
	msgType := MessageType(msgTypeStr)

	// Validate the message type
	for _, validType := range AllMessageTypes() {
		if msgType == validType {
			return msgType, true
		}
	}

	return "", false
}

// IsValidMessageType checks if a message type is valid
func IsValidMessageType(msgType MessageType) bool {
	for _, validType := range AllMessageTypes() {
		if msgType == validType {
			return true
		}
	}
	return false
}

// IsDeprecatedMessageType checks if a message type is deprecated
func IsDeprecatedMessageType(msgType MessageType) bool {
	deprecatedTypes := []MessageType{
		MessageTimeAlert,
	}

	for _, deprecatedType := range deprecatedTypes {
		if msgType == deprecatedType {
			return true
		}
	}
	return false
}

// TypedMessage represents a WebSocket message with typed message type
type TypedMessage struct {
	Type MessageType `json:"type"`
	Data any         `json:"data"`
	// UserID *int        `json:"user_id,omitempty"` // Reserved for future user-specific messaging
	Target ClientType `json:"target,omitempty"`
}

// NewTypedMessage creates a new typed message
func NewTypedMessage(msgType MessageType, data any) TypedMessage {
	return TypedMessage{
		Type: msgType,
		Data: data,
	}
}

// NewTypedMessageWithTarget creates a new typed message with target
func NewTypedMessageWithTarget(msgType MessageType, data any, target ClientType) TypedMessage {
	return TypedMessage{
		Type:   msgType,
		Data:   data,
		Target: target,
	}
}

// NewTypedMessageWithUser creates a new typed message with user ID (not implemented yet)
// TODO: Implement user-specific messaging when needed
// func NewTypedMessageWithUser(msgType MessageType, data any, userID int) TypedMessage {
// 	return TypedMessage{
// 		Type:   msgType,
// 		Data:   data,
// 		UserID: userID,
// 	}
// }
