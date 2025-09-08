package services

import (
	"fmt"
	"quiz100/models"
	"quiz100/websocket"
	"time"
)

// StateService provides high-level state management operations
type StateService struct {
	stateManager *models.EventStateManager
	hubManager   *websocket.HubManager
	logger       Logger
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
func NewStateService(stateManager *models.EventStateManager, hubManager *websocket.HubManager, logger Logger) *StateService {
	return &StateService{
		stateManager: stateManager,
		hubManager:   hubManager,
		logger:       logger,
	}
}

// GetCurrentState returns the current state
func (ss *StateService) GetCurrentState() models.EventState {
	return ss.stateManager.GetCurrentState()
}

// GetCurrentQuestion returns the current question number
func (ss *StateService) GetCurrentQuestion() int {
	return ss.stateManager.GetCurrentQuestion()
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

	// Broadcast state change
	ss.broadcastStateChange(previousState, targetState)

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

	return &StateTransitionResult{
		PreviousState: previousState,
		NewState:      targetState,
		Success:       true,
		Message:       fmt.Sprintf("Successfully jumped from %s to %s", previousState, targetState),
	}
}

// SetCurrentQuestion sets the current question number
func (ss *StateService) SetCurrentQuestion(questionNumber int) error {
	return ss.stateManager.SetCurrentQuestion(questionNumber)
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

	// Log transition
	if newState != previousState {
		ss.logger.LogStateTransition(previousState, newState)

		// Broadcast state change if state actually changed
		ss.broadcastStateChange(previousState, newState)
	}

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
func (ss *StateService) GetStateInfo() map[string]interface{} {
	return map[string]interface{}{
		"current_state":     ss.stateManager.GetCurrentState(),
		"current_question":  ss.stateManager.GetCurrentQuestion(),
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
	stateData := map[string]interface{}{
		"previous_state":   previousState,
		"new_state":        newState,
		"current_question": ss.stateManager.GetCurrentQuestion(),
		"jumped":           false, // This could be parameterized if needed
		"timestamp":        time.Now().UTC(),
	}

	if err := ss.hubManager.BroadcastStateChanged(stateData); err != nil {
		ss.logger.LogError("broadcasting state change", err)
	}
}

// executeCountdownSequence executes the 5-second countdown and auto-transitions
func (ss *StateService) executeCountdownSequence() {
	// Send countdown messages (5 to 1 seconds)
	for i := 5; i >= 1; i-- {
		countdownData := map[string]interface{}{
			"seconds_left": i,
		}

		if err := ss.hubManager.BroadcastCountdown(countdownData); err != nil {
			ss.logger.LogError("broadcasting countdown", err)
		}

		time.Sleep(1 * time.Second)
	}

	// Send final countdown message with 0 to trigger answer blocking
	finalCountdownData := map[string]interface{}{
		"seconds_left": 0,
	}
	if err := ss.hubManager.BroadcastCountdown(finalCountdownData); err != nil {
		ss.logger.LogError("broadcasting final countdown", err)
	}

	// Send question end message
	endData := map[string]interface{}{
		"message": "Time's up!",
	}

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
