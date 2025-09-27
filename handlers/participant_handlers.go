package handlers

import (
	"net/http"
	"quiz100/models"
	"quiz100/services"
	"quiz100/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ParticipantHandlers contains handlers for participant-related operations
type ParticipantHandlers struct {
	userRepo          *models.UserRepository
	teamRepo          *models.TeamRepository
	answerRepo        *models.AnswerRepository
	emojiReactionRepo *models.EmojiReactionRepository
	hubManager        *websocket.HubManager
	stateService      *services.StateService
	logger            models.QuizLogger
	config            *models.Config
}

// JoinRequest represents a join request from a participant
type JoinRequest struct {
	Nickname string `json:"nickname" binding:"required"`
}

// AnswerRequest represents an answer submission from a participant
type AnswerRequest struct {
	QuestionNumber int `json:"question_number" binding:"required"`
	AnswerIndex    int `json:"answer_index" binding:"required"`
}

// EmojiRequest represents an emoji reaction from a participant
type EmojiRequest struct {
	Emoji string `json:"emoji" binding:"required"`
}

// NewParticipantHandlers creates a new ParticipantHandlers instance
func NewParticipantHandlers(
	userRepo *models.UserRepository,
	teamRepo *models.TeamRepository,
	answerRepo *models.AnswerRepository,
	emojiReactionRepo *models.EmojiReactionRepository,
	hubManager *websocket.HubManager,
	stateService *services.StateService,
	logger models.QuizLogger,
	config *models.Config,
) *ParticipantHandlers {
	return &ParticipantHandlers{
		userRepo:          userRepo,
		teamRepo:          teamRepo,
		answerRepo:        answerRepo,
		emojiReactionRepo: emojiReactionRepo,
		hubManager:        hubManager,
		stateService:      stateService,
		logger:            logger,
		config:            config,
	}
}

// Join handles participant joining the quiz
func (ph *ParticipantHandlers) Join(c *gin.Context) {
	var req JoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	existingUser, err := ph.userRepo.GetUserBySessionID(sessionID)
	if err != nil {
		ph.logger.LogError("checking existing user", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var user *models.User
	var assignedTeam *models.Team

	// Check if this is a session rejoin attempt
	isRejoinAttempt := req.Nickname == "Rejoining..."

	if existingUser != nil {
		user = existingUser
		ph.logger.LogUserReconnect(user.Nickname, sessionID)

		assignedTeam, err = ph.teamRepo.GetTeamByID(user.ID)
		if err != nil {
			ph.logger.LogError("error during acquiring team by id", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Team database error"})
		}
	} else {
		// If this is a rejoin attempt but no existing user found, return error
		if isRejoinAttempt {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or not found"})
			return
		}

		user, err = ph.userRepo.CreateUser(sessionID, req.Nickname)
		if err != nil {
			ph.logger.LogError("creating user", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
		ph.logger.LogUserJoin(req.Nickname, sessionID)

		// Check if teams exist and team mode is enabled for automatic team assignment
		if ph.config.Event.TeamMode {
			// Auto-assignment logic would be implemented here if needed
			// For now, we'll skip this as it requires team assignment service
		}
	}

	err = ph.userRepo.UpdateUserConnection(sessionID, true)
	if err != nil {
		ph.logger.LogError("updating user connection", err)
	}

	teamname := ""
	if assignedTeam != nil {
		teamname = assignedTeam.Name
	}
	// Broadcast user joined notification
	userData := gin.H{
		"user":     user,
		"nickname": user.Nickname,
		"teamname": teamname,
	}

	if err := ph.hubManager.BroadcastUserJoined(userData); err != nil {
		ph.logger.LogError("broadcasting user joined", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"user":          user,
		"session_id":    sessionID,
		"assigned_team": assignedTeam,
	})
}

// Answer handles participant answer submission
func (ph *ParticipantHandlers) Answer(c *gin.Context) {
	var req AnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	user, err := ph.userRepo.GetUserBySessionID(sessionID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	if req.QuestionNumber < 1 || req.QuestionNumber > len(ph.config.Questions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question number"})
		return
	}

	// 現在回答受付中かどうか判断する
	currentState := ph.stateService.GetCurrentState()
	currentQuestion := ph.stateService.GetCurrentQuestion()
	if currentState != models.StateQuestionActive && currentState != models.StateCountdownActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not currently accepting answers"})
		return
	}
	if currentQuestion != req.QuestionNumber { // XXX: 数字あっているか？
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not currently acception answers number"})
		return
	}

	existingAnswer, err := ph.answerRepo.GetAnswerByUserAndQuestion(user.ID, req.QuestionNumber)
	if err != nil {
		ph.logger.LogError("checking existing answer", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	question := ph.config.Questions[req.QuestionNumber-1]
	// Convert 1-based answer index to 0-based for comparison with 1-based correct answer
	isCorrect := req.AnswerIndex == question.Correct

	if existingAnswer == nil {
		// 新規回答
		err = ph.answerRepo.CreateAnswer(user.ID, req.QuestionNumber, req.AnswerIndex, isCorrect)
		if err != nil {
			ph.logger.LogError("creating answer", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save answer"})
			return
		}
	} else {
		// 回答済み、選択肢変更
		err = ph.answerRepo.ChangeAnswer(user.ID, req.QuestionNumber, req.AnswerIndex, isCorrect)
		if err != nil {
			ph.logger.LogError("changing answer", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change answer"})
			return
		}
	}

	ph.logger.LogAnswer(user.Nickname, req.QuestionNumber, req.AnswerIndex, isCorrect)

	// Broadcast answer received notification
	answerData := gin.H{
		"nickname":        user.Nickname,
		"question_number": req.QuestionNumber,
		"answer":          req.AnswerIndex,
	}

	if err := ph.hubManager.BroadcastAnswerReceived(answerData); err != nil {
		ph.logger.LogError("broadcasting answer received", err)
	}

	c.JSON(http.StatusOK, gin.H{})
}

// SendEmoji handles participant emoji reactions
func (ph *ParticipantHandlers) SendEmoji(c *gin.Context) {
	var req EmojiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	user, err := ph.userRepo.GetUserBySessionID(sessionID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	ph.logger.LogEmojiReaction(user.Nickname, req.Emoji)

	// Broadcast emoji reaction
	emojiData := gin.H{
		"nickname": user.Nickname,
		"emoji":    req.Emoji,
	}

	if err := ph.hubManager.BroadcastEmojiReaction(emojiData); err != nil {
		ph.logger.LogError("broadcasting emoji reaction", err)
	}

	if err = ph.emojiReactionRepo.CreateReaction(user.ID, req.Emoji); err != nil {
		ph.logger.LogError("Emoji create failed", err)
	}

	c.JSON(http.StatusOK, gin.H{})
}

// ResetSession handles participant session reset
func (ph *ParticipantHandlers) ResetSession(c *gin.Context) {
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	user, err := ph.userRepo.GetUserBySessionID(sessionID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	// Delete user's answers
	err = ph.answerRepo.DeleteAnswersByUserID(user.ID)
	if err != nil {
		ph.logger.LogError("deleting user answers", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user answers"})
		return
	}

	// Delete user's emoji reactions
	err = ph.emojiReactionRepo.DeleteReactionsByUserID(user.ID)
	if err != nil {
		ph.logger.LogError("deleting user emoji reactions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user reactions"})
		return
	}

	// Delete user record
	err = ph.userRepo.DeleteUserBySessionID(sessionID)
	if err != nil {
		ph.logger.LogError("deleting user", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	ph.logger.LogUserSessionReset(user.Nickname, sessionID)

	// Broadcast user left notification
	userData := gin.H{
		"user_id":  user.ID,
		"nickname": user.Nickname,
	}

	if err := ph.hubManager.BroadcastUserLeft(userData); err != nil {
		ph.logger.LogError("broadcasting user left", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "Session reset successfully",
	})
}
