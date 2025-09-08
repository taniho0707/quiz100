package models

import "fmt"

// EventState represents the state of a quiz event
type EventState string

// Event state constants
const (
	StateWaiting         EventState = "waiting"
	StateStarted         EventState = "started"
	StateTitleDisplay    EventState = "title_display"
	StateTeamAssignment  EventState = "team_assignment"
	StateQuestionActive  EventState = "question_active"
	StateCountdownActive EventState = "countdown_active"
	StateAnswerStats     EventState = "answer_stats"
	StateAnswerReveal    EventState = "answer_reveal"
	StateResults         EventState = "results"
	StateCelebration     EventState = "celebration"
	StateFinished        EventState = "finished"
)

// StateLabels provides Japanese labels for each state
var StateLabels = map[EventState]string{
	StateWaiting:         "参加者待ち",
	StateStarted:         "イベント開始",
	StateTitleDisplay:    "タイトル表示",
	StateTeamAssignment:  "チーム分け",
	StateQuestionActive:  "問題表示中",
	StateCountdownActive: "カウントダウン中",
	StateAnswerStats:     "回答状況表示",
	StateAnswerReveal:    "回答発表",
	StateResults:         "結果発表",
	StateCelebration:     "お疲れ様画面",
	StateFinished:        "終了",
}

// AllStates returns all valid states
func AllStates() []EventState {
	return []EventState{
		StateWaiting,
		StateStarted,
		StateTitleDisplay,
		StateTeamAssignment,
		StateQuestionActive,
		StateCountdownActive,
		StateAnswerStats,
		StateAnswerReveal,
		StateResults,
		StateCelebration,
		StateFinished,
	}
}

// StateToString converts EventState to string
func StateToString(state EventState) string {
	return string(state)
}

// StringToState converts string to EventState with validation
func StringToState(stateStr string) (EventState, error) {
	state := EventState(stateStr)

	// Validate the state
	for _, validState := range AllStates() {
		if state == validState {
			return state, nil
		}
	}

	return "", fmt.Errorf("invalid state: %s", stateStr)
}

// GetStateLabel returns the Japanese label for a state
func GetStateLabel(state EventState) string {
	if label, exists := StateLabels[state]; exists {
		return label
	}
	return string(state)
}

// GetAllStatesWithLabels returns all states with their Japanese labels
func GetAllStatesWithLabels() []map[string]string {
	states := make([]map[string]string, 0, len(AllStates()))

	for _, state := range AllStates() {
		states = append(states, map[string]string{
			"value": StateToString(state),
			"label": GetStateLabel(state),
		})
	}

	return states
}

// IsValidState checks if a state is valid
func IsValidState(state EventState) bool {
	for _, validState := range AllStates() {
		if state == validState {
			return true
		}
	}
	return false
}
