package services

import (
	"fmt"
	"quiz100/models"
	"quiz100/websocket"
	"sync"
	"time"
)

// RobustnessService provides system robustness and recovery mechanisms
type RobustnessService struct {
	stateManager *models.EventStateManager
	hubManager   *websocket.HubManager
	userRepo     *models.UserRepository
	eventRepo    *models.EventRepository
	logger       models.QuizLogger
	config       *models.Config

	// Recovery state
	lastKnownState     models.EventState
	lastQuestionNumber int
	lastEventSnapshot  *EventSnapshot

	// Monitoring
	isMonitoring       bool
	monitoringTicker   *time.Ticker
	monitoringInterval time.Duration

	// Locks
	mu sync.RWMutex
}

// EventSnapshot represents a snapshot of the current system state
type EventSnapshot struct {
	Timestamp      time.Time           `json:"timestamp"`
	State          models.EventState   `json:"state"`
	QuestionNumber int                 `json:"question_number"`
	Event          *models.Event       `json:"event,omitempty"`
	ConnectedUsers int                 `json:"connected_users"`
	ClientCounts   map[string]int      `json:"client_counts"`
	SystemHealth   SystemHealthMetrics `json:"system_health"`
}

// SystemHealthMetrics represents system health indicators
type SystemHealthMetrics struct {
	WebSocketConnections int           `json:"websocket_connections"`
	DatabaseConnected    bool          `json:"database_connected"`
	MemoryUsage          int64         `json:"memory_usage_mb"`
	Uptime               time.Duration `json:"uptime"`
	ErrorCount           int           `json:"error_count"`
	LastError            string        `json:"last_error,omitempty"`
}

// RecoveryAction represents a recovery action
type RecoveryAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
}

// NewRobustnessService creates a new robustness service
func NewRobustnessService(
	stateManager *models.EventStateManager,
	hubManager *websocket.HubManager,
	userRepo *models.UserRepository,
	eventRepo *models.EventRepository,
	logger models.QuizLogger,
	config *models.Config,
) *RobustnessService {
	return &RobustnessService{
		stateManager:       stateManager,
		hubManager:         hubManager,
		userRepo:           userRepo,
		eventRepo:          eventRepo,
		logger:             logger,
		config:             config,
		monitoringInterval: 30 * time.Second,
		lastKnownState:     models.StateWaiting,
	}
}

// StartMonitoring starts the system health monitoring
func (rs *RobustnessService) StartMonitoring() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.isMonitoring {
		return
	}

	rs.isMonitoring = true
	rs.monitoringTicker = time.NewTicker(rs.monitoringInterval)

	go rs.monitoringLoop()
	rs.logger.Info("System robustness monitoring started")
}

// StopMonitoring stops the system health monitoring
func (rs *RobustnessService) StopMonitoring() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if !rs.isMonitoring {
		return
	}

	rs.isMonitoring = false
	if rs.monitoringTicker != nil {
		rs.monitoringTicker.Stop()
		rs.monitoringTicker = nil
	}

	rs.logger.Info("System robustness monitoring stopped")
}

// monitoringLoop runs the main monitoring loop
func (rs *RobustnessService) monitoringLoop() {
	defer func() {
		if r := recover(); r != nil {
			rs.logger.LogError("robustness monitoring", fmt.Errorf("monitoring loop panic: %v", r))
			// Restart monitoring after a short delay
			time.Sleep(5 * time.Second)
			rs.StartMonitoring()
		}
	}()

	for rs.isMonitoring {
		select {
		case <-rs.monitoringTicker.C:
			rs.performHealthCheck()
		}
	}
}

// performHealthCheck performs a comprehensive system health check
func (rs *RobustnessService) performHealthCheck() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Create current snapshot
	snapshot := rs.createEventSnapshot()

	// Check for inconsistencies
	if rs.lastEventSnapshot != nil {
		rs.detectAndFixInconsistencies(rs.lastEventSnapshot, snapshot)
	}

	// Update last known state
	rs.lastEventSnapshot = snapshot
	rs.lastKnownState = snapshot.State
	rs.lastQuestionNumber = snapshot.QuestionNumber

	// Log health status periodically (every 5 minutes)
	if time.Since(rs.lastEventSnapshot.Timestamp).Minutes() >= 5 {
		rs.logSystemHealth(snapshot)
	}
}

// createEventSnapshot creates a snapshot of the current system state
func (rs *RobustnessService) createEventSnapshot() *EventSnapshot {
	return &EventSnapshot{
		Timestamp:      time.Now(),
		State:          rs.stateManager.GetCurrentState(),
		QuestionNumber: rs.stateManager.GetQuestionNumber(),
		ConnectedUsers: rs.getConnectedUserCount(),
		ClientCounts:   rs.hubManager.GetClientCount(),
		SystemHealth:   rs.getSystemHealthMetrics(),
	}
}

// detectAndFixInconsistencies detects and fixes system inconsistencies
func (rs *RobustnessService) detectAndFixInconsistencies(previous, current *EventSnapshot) {
	var actions []RecoveryAction

	// Check for state inconsistencies
	if stateActions := rs.checkStateConsistency(previous, current); len(stateActions) > 0 {
		actions = append(actions, stateActions...)
	}

	// Check for WebSocket connection issues
	if wsActions := rs.checkWebSocketHealth(previous, current); len(wsActions) > 0 {
		actions = append(actions, wsActions...)
	}

	// Check for user synchronization issues
	if userActions := rs.checkUserSynchronization(previous, current); len(userActions) > 0 {
		actions = append(actions, userActions...)
	}

	// Execute recovery actions
	for _, action := range actions {
		rs.executeRecoveryAction(action)
	}
}

// checkStateConsistency checks for state consistency issues
func (rs *RobustnessService) checkStateConsistency(previous, current *EventSnapshot) []RecoveryAction {
	var actions []RecoveryAction

	// Check for invalid state transitions
	if previous.State != current.State {
		// Log state changes (this is normal)
		rs.logger.Info("State changed from %s to %s", previous.State, current.State)

		// Broadcast state sync to all clients
		actions = append(actions, RecoveryAction{
			Type:        "state_sync",
			Description: "Synchronize state across all clients",
			Data: map[string]interface{}{
				"state":           current.State,
				"question_number": current.QuestionNumber,
			},
			Timestamp: time.Now(),
		})
	}

	// Check for question number inconsistencies
	if current.State == models.StateQuestionActive && (current.QuestionNumber <= 0 || current.QuestionNumber > len(rs.config.Questions)) {
		actions = append(actions, RecoveryAction{
			Type:        "fix_question_number",
			Description: "Fix invalid question number",
			Data: map[string]interface{}{
				"question_number": current.QuestionNumber,
				"max_questions":   len(rs.config.Questions),
			},
			Timestamp: time.Now(),
		})
	}

	return actions
}

// checkWebSocketHealth checks for WebSocket connection health
func (rs *RobustnessService) checkWebSocketHealth(previous, current *EventSnapshot) []RecoveryAction {
	var actions []RecoveryAction

	// Check for significant connection drops
	prevTotal := previous.ClientCounts["admin"] + previous.ClientCounts["participant"] + previous.ClientCounts["screen"]
	currentTotal := current.ClientCounts["admin"] + current.ClientCounts["participant"] + current.ClientCounts["screen"]

	if prevTotal > 0 && currentTotal == 0 {
		// All connections lost - this might indicate a problem
		actions = append(actions, RecoveryAction{
			Type:        "websocket_recovery",
			Description: "All WebSocket connections lost - attempting recovery",
			Data: map[string]interface{}{
				"previous_connections": prevTotal,
				"current_connections":  currentTotal,
			},
			Timestamp: time.Now(),
		})
	}

	// Check for admin connection loss
	if previous.ClientCounts["admin"] > 0 && current.ClientCounts["admin"] == 0 {
		actions = append(actions, RecoveryAction{
			Type:        "admin_reconnect_notice",
			Description: "Admin connection lost - system may need manual intervention",
			Data: map[string]interface{}{
				"lost_admin_connections": previous.ClientCounts["admin"],
			},
			Timestamp: time.Now(),
		})
	}

	return actions
}

// checkUserSynchronization checks for user data synchronization issues
func (rs *RobustnessService) checkUserSynchronization(previous, current *EventSnapshot) []RecoveryAction {
	var actions []RecoveryAction

	// Check for significant user count changes without state changes
	if abs(previous.ConnectedUsers-current.ConnectedUsers) > 5 && previous.State == current.State {
		// Significant user count change without state change might indicate sync issues
		actions = append(actions, RecoveryAction{
			Type:        "user_sync_check",
			Description: "Significant user count change detected",
			Data: map[string]interface{}{
				"previous_users": previous.ConnectedUsers,
				"current_users":  current.ConnectedUsers,
				"state":          current.State,
			},
			Timestamp: time.Now(),
		})
	}

	return actions
}

// executeRecoveryAction executes a recovery action
func (rs *RobustnessService) executeRecoveryAction(action RecoveryAction) {
	rs.logger.Info("Executing recovery action: %s - %s", action.Type, action.Description)

	var err error

	switch action.Type {
	case "state_sync":
		err = rs.syncStateAcrossClients(action.Data)
	case "fix_question_number":
		err = rs.fixQuestionNumberInconsistency(action.Data)
	case "websocket_recovery":
		err = rs.attemptWebSocketRecovery(action.Data)
	case "admin_reconnect_notice":
		err = rs.notifyAdminReconnectionNeeded(action.Data)
	case "user_sync_check":
		err = rs.performUserSyncCheck(action.Data)
	default:
		err = fmt.Errorf("unknown recovery action type: %s", action.Type)
	}

	if err != nil {
		action.Success = false
		action.Error = err.Error()
		rs.logger.LogError("recovery action", err)
	} else {
		action.Success = true
	}
}

// syncStateAcrossClients synchronizes state across all connected clients
func (rs *RobustnessService) syncStateAcrossClients(data map[string]interface{}) error {
	state, ok := data["state"].(models.EventState)
	if !ok {
		return fmt.Errorf("invalid state in sync data")
	}

	questionNumber, ok := data["question_number"].(int)
	if !ok {
		return fmt.Errorf("invalid question_number in sync data")
	}

	syncData := map[string]interface{}{
		"state":           state,
		"question_number": questionNumber,
		"sync_type":       "robustness_recovery",
		"timestamp":       time.Now().Unix(),
	}

	return rs.hubManager.BroadcastStateChanged(syncData)
}

// fixQuestionNumberInconsistency fixes question number inconsistencies
func (rs *RobustnessService) fixQuestionNumberInconsistency(data map[string]interface{}) error {
	currentQuestion, ok := data["question_number"].(int)
	if !ok {
		return fmt.Errorf("invalid question_number in data")
	}

	maxQuestions, ok := data["max_questions"].(int)
	if !ok {
		return fmt.Errorf("invalid max_questions in data")
	}

	// Fix the question number
	var newQuestionNumber int
	if currentQuestion <= 0 {
		newQuestionNumber = 1
	} else if currentQuestion > maxQuestions {
		newQuestionNumber = maxQuestions
	} else {
		return nil // No fix needed
	}

	err := rs.stateManager.SetQuestionNumber(newQuestionNumber)
	if err != nil {
		return fmt.Errorf("failed to fix question number: %w", err)
	}

	rs.logger.Info("Fixed question number from %d to %d", currentQuestion, newQuestionNumber)
	return nil
}

// attemptWebSocketRecovery attempts to recover from WebSocket issues
func (rs *RobustnessService) attemptWebSocketRecovery(data map[string]interface{}) error {
	rs.logger.Warning("WebSocket recovery attempted - manual intervention may be required")

	// For now, just log the issue. In a more sophisticated system,
	// we might restart the WebSocket hub or take other recovery actions

	return nil
}

// notifyAdminReconnectionNeeded notifies that admin reconnection is needed
func (rs *RobustnessService) notifyAdminReconnectionNeeded(data map[string]interface{}) error {
	rs.logger.Warning("Admin connection lost - system may require manual intervention")

	// In a production system, this might send notifications to administrators
	// via email, SMS, or other alerting mechanisms

	return nil
}

// performUserSyncCheck performs a user synchronization check
func (rs *RobustnessService) performUserSyncCheck(data map[string]interface{}) error {
	// Get current user count from database
	users, err := rs.userRepo.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get user count: %w", err)
	}

	dbUserCount := len(users)
	wsUserCount := rs.getConnectedUserCount()

	// If there's a significant discrepancy, log it
	if abs(dbUserCount-wsUserCount) > 2 {
		rs.logger.Warning("User count discrepancy detected - DB: %d, WebSocket: %d", dbUserCount, wsUserCount)
	}

	return nil
}

// Recovery methods for external use

// RecoverFromWebSocketDisconnection handles WebSocket disconnection recovery
func (rs *RobustnessService) RecoverFromWebSocketDisconnection(clientType websocket.ClientType, userID int, sessionID string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.logger.Info("Attempting recovery for disconnected client: type=%s, userID=%d", clientType, userID)

	// Update user connection status
	if userID > 0 {
		err := rs.userRepo.UpdateUserConnection(sessionID, false)
		if err != nil {
			rs.logger.LogError("updating user connection status", err)
		}
	}

	// Create recovery action
	action := RecoveryAction{
		Type:        "client_disconnect_recovery",
		Description: fmt.Sprintf("Handle disconnection of %s client", clientType),
		Data: map[string]interface{}{
			"client_type": string(clientType),
			"user_id":     userID,
			"session_id":  sessionID,
		},
		Timestamp: time.Now(),
	}

	rs.executeRecoveryAction(action)
	return nil
}

// RecoverFromStateInconsistency handles state inconsistency recovery
func (rs *RobustnessService) RecoverFromStateInconsistency(expectedState, actualState models.EventState) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.logger.Warning("State inconsistency detected: expected=%s, actual=%s", expectedState, actualState)

	// Force sync to actual state
	syncData := map[string]interface{}{
		"state":           actualState,
		"question_number": rs.stateManager.GetQuestionNumber(),
	}

	return rs.syncStateAcrossClients(syncData)
}

// Emergency recovery methods

// EmergencyReset performs an emergency system reset
func (rs *RobustnessService) EmergencyReset() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.logger.Warning("Performing emergency system reset")

	// Reset state to waiting
	err := rs.stateManager.JumpToState(models.StateWaiting)
	if err != nil {
		return fmt.Errorf("failed to reset state: %w", err)
	}

	// Reset question number
	err = rs.stateManager.SetQuestionNumber(0)
	if err != nil {
		return fmt.Errorf("failed to reset question number: %w", err)
	}

	// Broadcast reset to all clients
	resetData := map[string]interface{}{
		"state":           models.StateWaiting,
		"question_number": 0,
		"reset_type":      "emergency",
		"timestamp":       time.Now().Unix(),
	}

	err = rs.hubManager.BroadcastStateChanged(resetData)
	if err != nil {
		return fmt.Errorf("failed to broadcast reset: %w", err)
	}

	rs.logger.Info("Emergency reset completed successfully")
	return nil
}

// Utility methods

// getConnectedUserCount gets the count of connected users
func (rs *RobustnessService) getConnectedUserCount() int {
	users, err := rs.userRepo.GetAllUsers()
	if err != nil {
		rs.logger.LogError("getting connected user count", err)
		return 0
	}

	count := 0
	for _, user := range users {
		if user.Connected {
			count++
		}
	}
	return count
}

// getSystemHealthMetrics gets current system health metrics
func (rs *RobustnessService) getSystemHealthMetrics() SystemHealthMetrics {
	clientCounts := rs.hubManager.GetClientCount()
	totalConnections := 0
	for _, count := range clientCounts {
		totalConnections += count
	}

	return SystemHealthMetrics{
		WebSocketConnections: totalConnections,
		DatabaseConnected:    true, // Simplified - in reality, this would check DB connectivity
		MemoryUsage:          rs.getMemoryUsage(),
		Uptime:               time.Since(time.Now().Add(-time.Hour)), // Placeholder
		ErrorCount:           0,                                      // Would track error count in a real implementation
	}
}

// getMemoryUsage gets current memory usage (simplified)
func (rs *RobustnessService) getMemoryUsage() int64 {
	// This is a placeholder - in a real implementation,
	// you would use runtime.ReadMemStats() to get actual memory usage
	return 0
}

// logSystemHealth logs system health information
func (rs *RobustnessService) logSystemHealth(snapshot *EventSnapshot) {
	rs.logger.Info("System Health - State: %s, Question: %d, Users: %d, WS Connections: %d",
		snapshot.State,
		snapshot.QuestionNumber,
		snapshot.ConnectedUsers,
		snapshot.SystemHealth.WebSocketConnections,
	)
}

// GetSystemStatus returns current system status for monitoring
func (rs *RobustnessService) GetSystemStatus() *EventSnapshot {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	return rs.createEventSnapshot()
}

// Helper functions

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
