package services

import (
	"fmt"
	"quiz100/models"
	"quiz100/websocket"
	"time"

	"github.com/gin-gonic/gin"
)

// StateService provides high-level state management operations
type StateService struct {
	stateManager *models.EventStateManager
	hubManager   *websocket.HubManager
	hub          *websocket.Hub
	logger       Logger
	config       *models.Config
	userRepo     *models.UserRepository
	teamRepo     *models.TeamRepository
	answerRepo   *models.AnswerRepository
}

// Logger interface for logging operations
type Logger interface {
	LogStateTransition(from, to models.EventState)
	LogError(context string, err error)
	LogAlert(message string)
}

// StateTransitionResult contains the result of a state transition
type StateTransitionResult struct {
	PreviousState models.EventState `json:"previous_state"`
	NewState      models.EventState `json:"new_state"`
	Success       bool              `json:"success"`
	Message       string            `json:"message,omitempty"`
	Error         error             `json:"-"`
}

// NewStateService creates a new StateService instance
func NewStateService(stateManager *models.EventStateManager, hubManager *websocket.HubManager, hub *websocket.Hub, logger Logger, config *models.Config, userRepo *models.UserRepository, teamRepo *models.TeamRepository, answerRepo *models.AnswerRepository) *StateService {
	ss := &StateService{
		stateManager: stateManager,
		hubManager:   hubManager,
		hub:          hub,
		logger:       logger,
		config:       config,
		userRepo:     userRepo,
		teamRepo:     teamRepo,
		answerRepo:   answerRepo,
	}

	// // Start periodic sync with 15-second interval
	// if hub != nil {
	// 	hub.StartPeriodicSync(15 * time.Second)
	// }

	return ss
}

// GetCurrentState returns the current state
func (ss *StateService) GetCurrentState() models.EventState {
	return ss.stateManager.GetCurrentState()
}

// GetQuestionNumber returns the current question number
func (ss *StateService) GetQuestionNumber() int {
	return ss.stateManager.GetQuestionNumber()
}

// GetAvailableActions returns available actions for the current state
func (ss *StateService) GetAvailableActions() []string {
	return ss.stateManager.GetAvailableActions()
}

// CanTransitionTo checks if transition to target state is allowed
func (ss *StateService) CanTransitionTo(targetState models.EventState) bool {
	return ss.stateManager.CanTransitionTo(targetState)
}

// TransitionTo performs a state transition with validation and logging
func (ss *StateService) TransitionTo(targetState models.EventState) *StateTransitionResult {
	previousState := ss.stateManager.GetCurrentState()

	// Validate transition
	if !ss.stateManager.CanTransitionTo(targetState) {
		result := &StateTransitionResult{
			PreviousState: previousState,
			NewState:      previousState, // No change
			Success:       false,
			Message:       fmt.Sprintf("Invalid transition from %s to %s", previousState, targetState),
			Error:         fmt.Errorf("invalid transition from %s to %s", previousState, targetState),
		}
		ss.logger.LogError("state transition", result.Error)
		return result
	}

	// Perform transition
	if err := ss.stateManager.TransitionTo(targetState); err != nil {
		result := &StateTransitionResult{
			PreviousState: previousState,
			NewState:      previousState, // No change
			Success:       false,
			Message:       fmt.Sprintf("Transition failed: %v", err),
			Error:         err,
		}
		ss.logger.LogError("state transition", err)
		return result
	}

	// Log successful transition
	ss.logger.LogStateTransition(previousState, targetState)

	// // Broadcast state change
	// ss.broadcastStateChange(previousState, targetState)

	// Update Hub event state for synchronization
	ss.UpdateEventState()

	return &StateTransitionResult{
		PreviousState: previousState,
		NewState:      targetState,
		Success:       true,
		Message:       fmt.Sprintf("Successfully transitioned from %s to %s", previousState, targetState),
	}
}

// JumpToState performs an unrestricted state jump (for admin use)
func (ss *StateService) JumpToState(targetState models.EventState) *StateTransitionResult {
	previousState := ss.stateManager.GetCurrentState()

	if err := ss.stateManager.JumpToState(targetState); err != nil {
		result := &StateTransitionResult{
			PreviousState: previousState,
			NewState:      previousState, // No change
			Success:       false,
			Message:       fmt.Sprintf("State jump failed: %v", err),
			Error:         err,
		}
		ss.logger.LogError("state jump", err)
		return result
	}

	// Log state jump
	ss.logger.LogAlert(fmt.Sprintf("Admin jumped from state %s to %s", previousState, targetState))

	// Broadcast state change
	ss.broadcastStateChange(previousState, targetState)

	// Update Hub event state for synchronization
	ss.UpdateEventState()

	return &StateTransitionResult{
		PreviousState: previousState,
		NewState:      targetState,
		Success:       true,
		Message:       fmt.Sprintf("Successfully jumped from %s to %s", previousState, targetState),
	}
}

// SetQuestionNumber sets the current question number
func (ss *StateService) SetQuestionNumber(questionNumber int) error {
	return ss.stateManager.SetQuestionNumber(questionNumber)
}

// NextQuestion advances to the next question
func (ss *StateService) NextQuestion() *StateTransitionResult {
	previousState := ss.stateManager.GetCurrentState()

	if err := ss.stateManager.NextQuestion(); err != nil {
		result := &StateTransitionResult{
			PreviousState: previousState,
			NewState:      previousState, // No change
			Success:       false,
			Message:       fmt.Sprintf("Cannot advance to next question: %v", err),
			Error:         err,
		}
		ss.logger.LogError("next question", err)
		return result
	}

	newState := ss.stateManager.GetCurrentState()

	// // Log transition
	// if newState != previousState {
	// 	ss.logger.LogStateTransition(previousState, newState)

	// 	// Broadcast state change if state actually changed
	// 	ss.broadcastStateChange(previousState, newState)
	// }

	// Update Hub event state for synchronization (question number changed)
	ss.UpdateEventState()

	return &StateTransitionResult{
		PreviousState: previousState,
		NewState:      newState,
		Success:       true,
		Message:       "Successfully advanced to next question",
	}
}

// StartCountdownWithAutoTransition starts countdown and automatically transitions to answer stats
func (ss *StateService) StartCountdownWithAutoTransition() *StateTransitionResult {
	// First transition to countdown state
	result := ss.TransitionTo(models.StateCountdownActive)
	if !result.Success {
		return result
	}

	// Start countdown in a separate goroutine
	go ss.executeCountdownSequence()

	return result
}

// GetStateInfo returns comprehensive state information
func (ss *StateService) GetStateInfo() map[string]any {
	return map[string]any{
		"current_state":     ss.stateManager.GetCurrentState(),
		"question_number":   ss.stateManager.GetQuestionNumber(),
		"available_actions": ss.stateManager.GetAvailableActions(),
		"state_label":       models.GetStateLabel(ss.stateManager.GetCurrentState()),
		"timestamp":         time.Now().UTC(),
	}
}

// ValidateStateTransition validates if a state transition is logically correct
func (ss *StateService) ValidateStateTransition(from, to models.EventState) error {
	// Additional business logic validation can be added here
	// For example: ensure all participants have answered before revealing answers

	// Basic validation using state manager
	if from == to {
		return fmt.Errorf("source and target states are the same: %s", from)
	}

	if !models.IsValidState(from) {
		return fmt.Errorf("invalid source state: %s", from)
	}

	if !models.IsValidState(to) {
		return fmt.Errorf("invalid target state: %s", to)
	}

	return nil
}

// Private helper methods

// broadcastStateChange broadcasts state change notification to all clients
func (ss *StateService) broadcastStateChange(previousState, newState models.EventState) {
	stateData := map[string]any{
		"previous_state":  previousState,
		"new_state":       newState,
		"question_number": ss.stateManager.GetQuestionNumber(),
		"jumped":          false, // This could be parameterized if needed
		"timestamp":       time.Now().UTC(),
	}

	if err := ss.hubManager.BroadcastStateChanged(stateData); err != nil {
		ss.logger.LogError("broadcasting state change", err)
	}
}

// executeCountdownSequence executes the 5-second countdown and auto-transitions
func (ss *StateService) executeCountdownSequence() {
	countdownData := gin.H{
		"seconds_left": 5,
	}
	if err := ss.hubManager.BroadcastCountdown(countdownData); err != nil {
		ss.logger.LogError("broadcasting countdown", err)
	}

	time.Sleep(5500 * time.Millisecond)

	// Send question end message
	endData := map[string]any{}

	if err := ss.hubManager.BroadcastQuestionEnd(endData); err != nil {
		ss.logger.LogError("broadcasting question end", err)
	}

	// Auto-transition to answer stats
	if result := ss.TransitionTo(models.StateAnswerStats); !result.Success {
		ss.logger.LogError("auto-transition to answer stats", result.Error)
	}
}

// AutoTransitionToCelebration automatically transitions to celebration after delay
func (ss *StateService) AutoTransitionToCelebration(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		if result := ss.TransitionTo(models.StateFinished); !result.Success {
			ss.logger.LogError("auto-transition to finished", result.Error)
		}
	}()
}

// State Synchronization Methods

// GenerateEventSyncData creates comprehensive synchronization data for the current state
func (ss *StateService) GenerateEventSyncData() *websocket.EventSyncData {
	currentState := ss.stateManager.GetCurrentState()
	currentQuestion := ss.stateManager.GetQuestionNumber()

	syncData := &websocket.EventSyncData{
		EventState:     string(currentState),
		QuestionNumber: currentQuestion,
		// SyncVersion:     0, // Will be set by Hub
		// Timestamp:       time.Now(),
	}

	// Add question data if in question-related state
	if currentState == models.StateQuestionActive ||
		currentState == models.StateCountdownActive ||
		currentState == models.StateAnswerStats ||
		currentState == models.StateAnswerReveal {
		if ss.config != nil && currentQuestion > 0 && currentQuestion <= len(ss.config.Questions) {
			question := ss.config.Questions[currentQuestion-1]
			syncData.QuestionData = models.Question{
				Type:    question.Type,
				Text:    question.Text,
				Image:   question.Image,
				Choices: question.Choices,
				Correct: 0, // invalid value
			}
		}
	}

	// Add team data if in team mode and relevant state
	if ss.config != nil && ss.config.Event.TeamMode &&
		(currentState == models.StateTeamAssignment ||
			currentState == models.StateQuestionActive ||
			currentState == models.StateResults) {
		if teams, err := ss.teamRepo.GetAllTeamsWithMembers(); err == nil {
			teamData := make([]any, len(teams))
			for i, team := range teams {
				teamData[i] = map[string]any{
					"id":      team.ID,
					"name":    team.Name,
					"score":   team.Score,
					"members": team.Members,
				}
			}
			syncData.TeamData = teamData
		}
	}

	// Add participant data for admin visibility
	if users, err := ss.userRepo.GetAllUsers(); err == nil {
		participantData := make([]any, len(users))
		for i, user := range users {
			participantData[i] = map[string]any{
				"id":        user.ID,
				"nickname":  user.Nickname,
				"team_id":   user.TeamID,
				"score":     user.Score,
				"connected": user.Connected,
			}
		}
		syncData.ParticipantData = participantData
	}

	// Add answer data for current question if in question-related states
	if currentQuestion > 0 &&
		(currentState == models.StateQuestionActive ||
			currentState == models.StateCountdownActive ||
			currentState == models.StateAnswerStats ||
			currentState == models.StateAnswerReveal) {

		answerData := make(map[string]any)
		if users, err := ss.userRepo.GetAllUsers(); err == nil {
			for _, user := range users {
				answer, err := ss.answerRepo.GetAnswerByUserAndQuestion(user.ID, currentQuestion)
				if err == nil && answer != nil {
					answerData[fmt.Sprintf("%d", user.ID)] = answer.AnswerIndex
				}
			}
		}
		syncData.AnswerData = answerData
	}

	return syncData
}

// UpdateEventState updates the Hub's event state and triggers synchronization
func (ss *StateService) UpdateEventState() {
	if ss.hub == nil {
		return
	}

	syncData := ss.GenerateEventSyncData()
	ss.hub.UpdateEventState(syncData)
}

// RequestClientSync manually requests synchronization for a specific client
func (ss *StateService) RequestClientSync(userID int, syncType string) {
	if ss.hub == nil {
		return
	}

	// Find client by userID
	participants := ss.hub.GetClientsByType(websocket.ClientTypeParticipant)
	for _, client := range participants {
		if client.UserID == userID {
			ss.hub.RequestStateSync(client, syncType)
			ss.logger.LogAlert(fmt.Sprintf("Manual sync requested for user %d (type: %s)", userID, syncType))
			return
		}
	}

	ss.logger.LogError("client sync request", fmt.Errorf("client not found for user ID: %d", userID))
}

// GetClientSyncStatus returns synchronization status for all connected clients
func (ss *StateService) GetClientSyncStatus() map[int]*websocket.ClientState {
	if ss.hub == nil {
		return make(map[int]*websocket.ClientState)
	}

	return ss.hub.GetClientSyncStatus()
}

// SyncAllClients forces synchronization for all connected participants
func (ss *StateService) SyncAllClients() {
	if ss.hub == nil {
		return
	}

	participants := ss.hub.GetClientsByType(websocket.ClientTypeParticipant)
	count := 0

	for _, client := range participants {
		ss.hub.RequestStateSync(client, "manual")
		count++
	}

	ss.logger.LogAlert(fmt.Sprintf("Manual sync requested for %d participants", count))
}

// IsClientSynchronized checks if a specific client is synchronized with current state
func (ss *StateService) IsClientSynchronized(userID int) bool {
	if ss.hub == nil {
		return false
	}

	syncStatus := ss.hub.GetClientSyncStatus()
	clientState, exists := syncStatus[userID]
	if !exists || !clientState.IsInitialized {
		return false
	}

	currentState := string(ss.stateManager.GetCurrentState())
	currentQuestion := ss.stateManager.GetQuestionNumber()

	return clientState.LastEventState == currentState &&
		clientState.LastQuestionNum == currentQuestion &&
		time.Since(clientState.LastSyncTime) < 60*time.Second
}

// GetSynchronizationReport generates a report of all client sync statuses
func (ss *StateService) GetSynchronizationReport() map[string]any {
	if ss.hub == nil {
		return map[string]any{
			"error": "Hub not available",
		}
	}

	syncStatus := ss.hub.GetClientSyncStatus()
	currentState := string(ss.stateManager.GetCurrentState())
	questionNumber := ss.stateManager.GetQuestionNumber()

	synchronized := 0
	outdated := 0
	uninitialized := 0

	clientReports := make([]map[string]any, 0)

	for userID, clientState := range syncStatus {
		isSync := clientState.IsInitialized &&
			clientState.LastEventState == currentState &&
			clientState.LastQuestionNum == questionNumber &&
			time.Since(clientState.LastSyncTime) < 60*time.Second

		var status string
		if !clientState.IsInitialized {
			status = "uninitialized"
			uninitialized++
		} else if isSync {
			status = "synchronized"
			synchronized++
		} else {
			status = "outdated"
			outdated++
		}

		clientReports = append(clientReports, map[string]any{
			"user_id":           userID,
			"status":            status,
			"last_sync_time":    clientState.LastSyncTime,
			"last_event_state":  clientState.LastEventState,
			"last_question_num": clientState.LastQuestionNum,
			"sync_version":      clientState.SyncVersion,
		})
	}

	return map[string]any{
		"current_state":   currentState,
		"question_number": questionNumber,
		"total_clients":   len(syncStatus),
		"synchronized":    synchronized,
		"outdated":        outdated,
		"uninitialized":   uninitialized,
		"sync_rate":       float64(synchronized) / float64(len(syncStatus)) * 100,
		"client_details":  clientReports,
		"timestamp":       time.Now(),
	}
}
