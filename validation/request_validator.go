package validation

import (
	"fmt"
	"quiz100/errors"
	"quiz100/models"
	"regexp"
	"strings"
	"unicode/utf8"
)

// RequestValidator provides validation for all quiz application requests
type RequestValidator struct {
	config *models.Config
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(config *models.Config) *RequestValidator {
	return &RequestValidator{
		config: config,
	}
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid   bool
	Errors  []*errors.QuizError
	Details map[string]string
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(err *errors.QuizError) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, err)
}

// AddFieldError adds a field-specific error
func (vr *ValidationResult) AddFieldError(field, message string) {
	if vr.Details == nil {
		vr.Details = make(map[string]string)
	}
	vr.Details[field] = message

	err := errors.NewValidationError(
		fmt.Sprintf("Field '%s': %s", field, message),
		fmt.Sprintf("Field: %s, Message: %s", field, message),
	)
	vr.AddError(err)
}

// GetCombinedError returns a single error combining all validation errors
func (vr *ValidationResult) GetCombinedError() *errors.QuizError {
	if vr.Valid {
		return nil
	}

	if len(vr.Errors) == 1 {
		return vr.Errors[0]
	}

	var messages []string
	for _, err := range vr.Errors {
		messages = append(messages, err.Message)
	}

	return errors.NewValidationError(
		"複数の検証エラーがあります",
		strings.Join(messages, "; "),
	)
}

// NewValidationResult creates a new validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:   true,
		Errors:  make([]*errors.QuizError, 0),
		Details: make(map[string]string),
	}
}

// Nickname validation

// ValidateNickname validates a user nickname
func (rv *RequestValidator) ValidateNickname(nickname string) *ValidationResult {
	result := NewValidationResult()

	// Required field check
	if strings.TrimSpace(nickname) == "" {
		result.AddFieldError("nickname", "ニックネームが入力されていません")
		return result
	}

	// Length validation
	if utf8.RuneCountInString(nickname) < 1 {
		result.AddFieldError("nickname", "ニックネームが短すぎます")
	}

	if utf8.RuneCountInString(nickname) > 20 {
		result.AddFieldError("nickname", "ニックネームが長すぎます（20文字以内）")
	}

	// Character validation
	if containsInvalidChars(nickname) {
		result.AddFieldError("nickname", "ニックネームに使用できない文字が含まれています")
	}

	// Profanity check (basic implementation)
	if containsProfanity(nickname) {
		result.AddFieldError("nickname", "不適切な表現が含まれています")
	}

	return result
}

// Answer validation

// ValidateAnswer validates an answer submission
func (rv *RequestValidator) ValidateAnswer(questionNumber, answerIndex int, userID int) *ValidationResult {
	result := NewValidationResult()

	// Question number validation
	if questionNumber < 1 {
		result.AddFieldError("question_number", "問題番号が無効です")
	}

	if questionNumber > len(rv.config.Questions) {
		result.AddFieldError("question_number", "存在しない問題番号です")
		return result // Early return as other validations depend on this
	}

	// Answer index validation
	question := rv.config.Questions[questionNumber-1]
	if answerIndex < 0 || answerIndex >= len(question.Choices) {
		result.AddFieldError("answer_index", "無効な選択肢です")
	}

	// User ID validation
	if userID <= 0 {
		result.AddFieldError("user_id", "ユーザーIDが無効です")
	}

	return result
}

// State validation

// ValidateStateTransition validates a state transition request
func (rv *RequestValidator) ValidateStateTransition(from, to models.EventState) *ValidationResult {
	result := NewValidationResult()

	// Basic state validation
	if !models.IsValidState(from) {
		result.AddFieldError("from_state", "遷移元の状態が無効です")
	}

	if !models.IsValidState(to) {
		result.AddFieldError("to_state", "遷移先の状態が無効です")
	}

	// Same state check
	if from == to {
		result.AddFieldError("state_transition", "同じ状態への遷移は不要です")
	}

	// Business logic validation would be handled by EventStateManager
	// This validator focuses on input format validation

	return result
}

// ValidateQuestionNumber validates a question number
func (rv *RequestValidator) ValidateQuestionNumber(questionNumber int) *ValidationResult {
	result := NewValidationResult()

	if questionNumber < 0 {
		result.AddFieldError("question_number", "問題番号は0以上である必要があります")
	}

	if questionNumber > len(rv.config.Questions) {
		result.AddFieldError("question_number", fmt.Sprintf("問題番号は%d以下である必要があります", len(rv.config.Questions)))
	}

	return result
}

// Session validation

// ValidateSessionID validates a session ID format
func (rv *RequestValidator) ValidateSessionID(sessionID string) *ValidationResult {
	result := NewValidationResult()

	if strings.TrimSpace(sessionID) == "" {
		result.AddFieldError("session_id", "セッションIDが指定されていません")
		return result
	}

	// UUID format validation (basic)
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(sessionID) {
		result.AddFieldError("session_id", "セッションIDの形式が無効です")
	}

	return result
}

// Action validation

// ValidateAdminAction validates an admin action request
func (rv *RequestValidator) ValidateAdminAction(action string) *ValidationResult {
	result := NewValidationResult()

	if strings.TrimSpace(action) == "" {
		result.AddFieldError("action", "アクションが指定されていません")
		return result
	}

	validActions := map[string]bool{
		"start_event":       true,
		"show_title":        true,
		"assign_teams":      true,
		"next_question":     true,
		"countdown_alert":   true,
		"show_answer_stats": true,
		"reveal_answer":     true,
		"show_results":      true,
		"celebration":       true,
	}

	if !validActions[action] {
		result.AddFieldError("action", "無効なアクションです")
	}

	return result
}

// Emoji validation

// ValidateEmoji validates an emoji reaction
func (rv *RequestValidator) ValidateEmoji(emoji string) *ValidationResult {
	result := NewValidationResult()

	if strings.TrimSpace(emoji) == "" {
		result.AddFieldError("emoji", "絵文字が指定されていません")
		return result
	}

	// Allow common emojis (basic validation)
	allowedEmojis := map[string]bool{
		"❤️": true, "👏": true, "😊": true, "😮": true, "🤔": true, "😅": true,
		"👍": true, "👎": true, "🔥": true, "💯": true, "😄": true, "😢": true,
		"😱": true, "🤯": true, "👌": true, "🙌": true, "💖": true, "⭐": true,
	}

	if !allowedEmojis[emoji] {
		result.AddFieldError("emoji", "サポートされていない絵文字です")
	}

	return result
}

// Team validation

// ValidateTeamOperation validates team-related operations
func (rv *RequestValidator) ValidateTeamOperation() *ValidationResult {
	result := NewValidationResult()

	if !rv.config.Event.TeamMode {
		result.AddError(errors.ErrNotInTeamMode.WithDetails("チーム戦モードが有効になっていません"))
	}

	return result
}

// Complex validation methods

// ValidateUserContext validates user-related context
func (rv *RequestValidator) ValidateUserContext(userID int, sessionID string) *ValidationResult {
	result := NewValidationResult()

	if userID <= 0 {
		result.AddFieldError("user_id", "ユーザーIDが無効です")
	}

	sessionResult := rv.ValidateSessionID(sessionID)
	if !sessionResult.Valid {
		for _, err := range sessionResult.Errors {
			result.AddError(err)
		}
	}

	return result
}

// ValidateQuestionContext validates question-related context
func (rv *RequestValidator) ValidateQuestionContext(questionNumber int, userID int) *ValidationResult {
	result := NewValidationResult()

	// Question validation
	questionResult := rv.ValidateQuestionNumber(questionNumber)
	if !questionResult.Valid {
		for _, err := range questionResult.Errors {
			result.AddError(err)
		}
	}

	// User validation
	if userID <= 0 {
		result.AddFieldError("user_id", "ユーザーIDが無効です")
	}

	return result
}

// Batch validation methods

// ValidateJoinRequest validates a complete join request
func (rv *RequestValidator) ValidateJoinRequest(nickname, sessionID string) *ValidationResult {
	result := NewValidationResult()

	// Nickname validation
	nicknameResult := rv.ValidateNickname(nickname)
	if !nicknameResult.Valid {
		for _, err := range nicknameResult.Errors {
			result.AddError(err)
		}
	}

	// Session ID validation (if provided)
	if sessionID != "" {
		sessionResult := rv.ValidateSessionID(sessionID)
		if !sessionResult.Valid {
			for _, err := range sessionResult.Errors {
				result.AddError(err)
			}
		}
	}

	return result
}

// ValidateAnswerRequest validates a complete answer submission request
func (rv *RequestValidator) ValidateAnswerRequest(userID, questionNumber, answerIndex int, sessionID string) *ValidationResult {
	result := NewValidationResult()

	// User context validation
	userResult := rv.ValidateUserContext(userID, sessionID)
	if !userResult.Valid {
		for _, err := range userResult.Errors {
			result.AddError(err)
		}
	}

	// Answer validation
	answerResult := rv.ValidateAnswer(questionNumber, answerIndex, userID)
	if !answerResult.Valid {
		for _, err := range answerResult.Errors {
			result.AddError(err)
		}
	}

	return result
}

// ValidateEmojiRequest validates a complete emoji reaction request
func (rv *RequestValidator) ValidateEmojiRequest(userID int, sessionID, emoji string) *ValidationResult {
	result := NewValidationResult()

	// User context validation
	userResult := rv.ValidateUserContext(userID, sessionID)
	if !userResult.Valid {
		for _, err := range userResult.Errors {
			result.AddError(err)
		}
	}

	// Emoji validation
	emojiResult := rv.ValidateEmoji(emoji)
	if !emojiResult.Valid {
		for _, err := range emojiResult.Errors {
			result.AddError(err)
		}
	}

	return result
}

// Helper functions

// containsInvalidChars checks for invalid characters in nickname
func containsInvalidChars(nickname string) bool {
	// Disallow control characters, but allow most Unicode
	for _, r := range nickname {
		if r < 32 && r != 9 && r != 10 && r != 13 { // Allow tab, LF, CR
			return true
		}
		// Disallow some specific problematic characters
		if strings.ContainsRune("<>&\"'", r) {
			return true
		}
	}
	return false
}

// containsProfanity checks for basic profanity (simplified implementation)
func containsProfanity(text string) bool {
	// Basic profanity filter - in a real application, this would be more sophisticated
	lowered := strings.ToLower(text)

	// Japanese profanity (very basic list)
	profanityWords := []string{
		"ばか", "あほ", "きちく", "しね", "ころす",
		// Add more as needed, but be careful with false positives
	}

	for _, word := range profanityWords {
		if strings.Contains(lowered, word) {
			return true
		}
	}

	return false
}

// Sanitization helpers

// SanitizeNickname sanitizes a nickname for safe use
func SanitizeNickname(nickname string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(nickname)

	// Remove or replace problematic characters
	sanitized = strings.ReplaceAll(sanitized, "<", "&lt;")
	sanitized = strings.ReplaceAll(sanitized, ">", "&gt;")
	sanitized = strings.ReplaceAll(sanitized, "&", "&amp;")
	sanitized = strings.ReplaceAll(sanitized, "\"", "&quot;")
	sanitized = strings.ReplaceAll(sanitized, "'", "&#39;")

	return sanitized
}

// Data integrity validation

// ValidateConfigIntegrity validates the configuration integrity
func (rv *RequestValidator) ValidateConfigIntegrity() *ValidationResult {
	result := NewValidationResult()

	if rv.config == nil {
		result.AddError(errors.NewValidationError("設定が読み込まれていません", "Config is nil"))
		return result
	}

	// Event configuration validation
	if rv.config.Event.Title == "" {
		result.AddFieldError("event.title", "イベントタイトルが設定されていません")
	}

	if rv.config.Event.TeamMode && rv.config.Event.TeamSize <= 0 {
		result.AddFieldError("event.team_size", "チームモードではチームサイズを1以上に設定する必要があります")
	}

	// Questions validation
	if len(rv.config.Questions) == 0 {
		result.AddFieldError("questions", "問題が1つも設定されていません")
		return result
	}

	for i, question := range rv.config.Questions {
		if question.Text == "" {
			result.AddFieldError(fmt.Sprintf("questions[%d].text", i), "問題文が設定されていません")
		}

		if len(question.Choices) < 2 {
			result.AddFieldError(fmt.Sprintf("questions[%d].choices", i), "選択肢は2つ以上必要です")
		}

		if question.Correct < 0 || question.Correct >= len(question.Choices) {
			result.AddFieldError(fmt.Sprintf("questions[%d].correct", i), "正解番号が選択肢の範囲外です")
		}
	}

	return result
}

// Rate limiting validation (basic implementation)

// ValidateRateLimit validates if an operation is within rate limits
func (rv *RequestValidator) ValidateRateLimit(userID int, operation string) *ValidationResult {
	result := NewValidationResult()

	// This is a placeholder for rate limiting logic
	// In a real implementation, you might use Redis or in-memory storage
	// to track request rates per user and operation

	// For now, just return valid
	return result
}

// State consistency validation

// ValidateSystemState validates the overall system state consistency
func (rv *RequestValidator) ValidateSystemState(currentState models.EventState, questionNumber int) *ValidationResult {
	result := NewValidationResult()

	// Validate state-question consistency
	switch currentState {
	case models.StateQuestionActive:
		if questionNumber <= 0 || questionNumber > len(rv.config.Questions) {
			result.AddError(errors.NewStateError(
				errors.ErrCodeInvalidQuestionNum,
				"問題表示中ですが、有効な問題番号が設定されていません",
				fmt.Sprintf("State: %s, QuestionNumber: %d", currentState, questionNumber),
			))
		}
	case models.StateAnswerReveal:
		if questionNumber <= 0 || questionNumber > len(rv.config.Questions) {
			result.AddError(errors.NewStateError(
				errors.ErrCodeInvalidQuestionNum,
				"回答発表中ですが、有効な問題番号が設定されていません",
				fmt.Sprintf("State: %s, QuestionNumber: %d", currentState, questionNumber),
			))
		}
	}

	return result
}
