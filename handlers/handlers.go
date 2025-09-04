package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"quiz100/database"
	"quiz100/models"
	"quiz100/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	db     *database.Database
	hub    *websocket.Hub
	config *models.Config

	userRepo          *models.UserRepository
	teamRepo          *models.TeamRepository
	eventRepo         *models.EventRepository
	answerRepo        *models.AnswerRepository
	emojiReactionRepo *models.EmojiReactionRepository
	teamAssignmentSvc *models.TeamAssignmentService
	logger            *models.QuizLogger

	currentEvent    *models.Event
	stateManager    *models.EventStateManager
	currentQuestion *models.Question
}

type JoinRequest struct {
	Nickname string `json:"nickname" binding:"required"`
}

type AnswerRequest struct {
	QuestionNumber int `json:"question_number" binding:"required"`
	AnswerIndex    int `json:"answer_index" binding:"required"`
}

type EmojiRequest struct {
	Emoji string `json:"emoji" binding:"required"`
}

type AdminRequest struct {
	Action string `json:"action" binding:"required"`
}

type JumpStateRequest struct {
	State          string `json:"state" binding:"required"`
	QuestionNumber *int   `json:"question_number,omitempty"`
}

func NewHandler(db *database.Database, hub *websocket.Hub, config *models.Config, logger *models.QuizLogger) *Handler {
	userRepo := models.NewUserRepository(db.DB)
	teamRepo := models.NewTeamRepository(db.DB)

	stateManager := models.NewEventStateManager(config.Event.TeamMode, len(config.Questions))

	return &Handler{
		db:                db,
		hub:               hub,
		config:            config,
		userRepo:          userRepo,
		teamRepo:          teamRepo,
		eventRepo:         models.NewEventRepository(db.DB),
		answerRepo:        models.NewAnswerRepository(db.DB),
		emojiReactionRepo: models.NewEmojiReactionRepository(db.DB),
		teamAssignmentSvc: models.NewTeamAssignmentService(userRepo, teamRepo, config),
		logger:            logger,
		stateManager:      stateManager,
	}
}

func (h *Handler) GetParticipantPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Quiz Participant",
	})
}

func (h *Handler) GetAdminPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"title": "Quiz Admin",
	})
}

func (h *Handler) GetScreenPage(c *gin.Context) {
	c.HTML(http.StatusOK, "screen.html", gin.H{
		"title": "Quiz Screen",
	})
}

func (h *Handler) Join(c *gin.Context) {
	var req JoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	existingUser, err := h.userRepo.GetUserBySessionID(sessionID)
	if err != nil {
		log.Printf("Error checking existing user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var user *models.User
	var assignedTeam *models.Team

	// Check if this is a session rejoin attempt
	isRejoinAttempt := req.Nickname == "Rejoining..."

	if existingUser != nil {
		user = existingUser
		h.logger.LogUserReconnect(user.Nickname, sessionID)
	} else {
		// If this is a rejoin attempt but no existing user found, return error
		if isRejoinAttempt {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or not found"})
			return
		}

		user, err = h.userRepo.CreateUser(sessionID, req.Nickname)
		if err != nil {
			h.logger.LogError("creating user", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
		h.logger.LogUserJoin(req.Nickname, sessionID)

		// Check if teams exist and team mode is enabled for automatic team assignment
		if h.config.Event.TeamMode {
			existingTeams, err := h.teamRepo.GetAllTeamsWithMembers()
			if err == nil && len(existingTeams) > 0 {
				// Teams exist, assign new user to available team
				assignedTeam, err = h.teamAssignmentSvc.AssignUserToAvailableTeam(user.ID, user.Nickname)
				if err != nil {
					h.logger.LogError("auto-assigning user to team", err)
					// Continue without team assignment on error
				} else if assignedTeam != nil {
					h.logger.LogUserJoin(fmt.Sprintf("%s (assigned to %s)", req.Nickname, assignedTeam.Name), sessionID)
				}
			}
		}
	}

	err = h.userRepo.UpdateUserConnection(sessionID, true)
	if err != nil {
		h.logger.LogError("updating user connection", err)
	}

	message := websocket.Message{
		Type: "user_joined",
		Data: gin.H{
			"user":          user,
			"nickname":      user.Nickname,
			"assigned_team": assignedTeam,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeAdmin)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

	// If user was assigned to a team, also send team update message
	if assignedTeam != nil {
		teamMessage := websocket.Message{
			Type: "team_member_added",
			Data: gin.H{
				"team": assignedTeam,
				"user": user,
			},
		}
		teamMessageBytes, _ := json.Marshal(teamMessage)
		h.hub.BroadcastToType(teamMessageBytes, websocket.ClientTypeAdmin)
		h.hub.BroadcastToType(teamMessageBytes, websocket.ClientTypeScreen)
	}

	c.JSON(http.StatusOK, gin.H{
		"user":          user,
		"session_id":    sessionID,
		"assigned_team": assignedTeam,
	})
}

func (h *Handler) Answer(c *gin.Context) {
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

	user, err := h.userRepo.GetUserBySessionID(sessionID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	existingAnswer, err := h.answerRepo.GetAnswerByUserAndQuestion(user.ID, req.QuestionNumber)
	if err != nil {
		h.logger.LogError("checking existing answer", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if existingAnswer != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already answered this question"})
		return
	}

	if req.QuestionNumber < 1 || req.QuestionNumber > len(h.config.Questions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question number"})
		return
	}

	question := h.config.Questions[req.QuestionNumber-1]
	// Convert 1-based answer index to 0-based for comparison with 1-based correct answer
	isCorrect := req.AnswerIndex == question.Correct

	err = h.answerRepo.CreateAnswer(user.ID, req.QuestionNumber, req.AnswerIndex, isCorrect)
	if err != nil {
		h.logger.LogError("creating answer", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save answer"})
		return
	}

	h.logger.LogAnswer(user.Nickname, req.QuestionNumber, req.AnswerIndex, isCorrect)

	if isCorrect {
		newScore := user.Score + 1
		err = h.userRepo.UpdateUserScore(user.ID, newScore)
		if err != nil {
			h.logger.LogError("updating user score", err)
		} else {
			user.Score = newScore

			// If team mode is enabled, update team scores
			if h.config.Event.TeamMode && user.TeamID != nil {
				_, err = h.teamAssignmentSvc.CalculateTeamScores()
				if err != nil {
					h.logger.LogError("updating team scores", err)
				}
			}
		}
	}

	message := websocket.Message{
		Type: "answer_received",
		Data: gin.H{
			"user_id":         user.ID,
			"nickname":        user.Nickname,
			"question_number": req.QuestionNumber,
			"answer_index":    req.AnswerIndex,
			"is_correct":      isCorrect,
			"new_score":       user.Score,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeAdmin)

	c.JSON(http.StatusOK, gin.H{
		"is_correct": isCorrect,
		"new_score":  user.Score,
	})
}

func (h *Handler) SendEmoji(c *gin.Context) {
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

	user, err := h.userRepo.GetUserBySessionID(sessionID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	h.logger.LogEmojiReaction(user.Nickname, req.Emoji)

	message := websocket.Message{
		Type: "emoji",
		Data: gin.H{
			"user_id":  user.ID,
			"nickname": user.Nickname,
			"emoji":    req.Emoji,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (h *Handler) ResetSession(c *gin.Context) {
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	user, err := h.userRepo.GetUserBySessionID(sessionID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	// Delete user's answers
	err = h.answerRepo.DeleteAnswersByUserID(user.ID)
	if err != nil {
		h.logger.LogError("deleting user answers", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user answers"})
		return
	}

	// Delete user's emoji reactions
	err = h.emojiReactionRepo.DeleteReactionsByUserID(user.ID)
	if err != nil {
		h.logger.LogError("deleting user emoji reactions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user reactions"})
		return
	}

	// Delete user record
	err = h.userRepo.DeleteUserBySessionID(sessionID)
	if err != nil {
		h.logger.LogError("deleting user", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	h.logger.LogUserSessionReset(user.Nickname, sessionID)

	// Notify admin and screen about user leaving
	message := websocket.Message{
		Type: "user_left",
		Data: gin.H{
			"user_id":  user.ID,
			"nickname": user.Nickname,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeAdmin)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

	c.JSON(http.StatusOK, gin.H{
		"status": "Session reset successfully",
	})
}

func (h *Handler) AdminStart(c *gin.Context) {
	if h.currentEvent != nil && h.currentEvent.Status == "started" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event already started"})
		return
	}

	event, err := h.eventRepo.CreateEvent(h.config.Event.Title, h.config.Event.TeamMode, h.config.Event.TeamSize, h.config.Event.QrCode)
	if err != nil {
		h.logger.LogError("creating event", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	h.currentEvent = event

	err = h.eventRepo.UpdateEventStatus(event.ID, "started")
	if err != nil {
		h.logger.LogError("updating event status", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start event"})
		return
	}

	h.currentEvent.Status = "started"

	// Log event start
	users, _ := h.userRepo.GetAllUsers()
	h.logger.LogEventStart(h.config.Event.Title, h.config.Event.TeamMode, len(users))

	message := websocket.Message{
		Type: "event_started",
		Data: gin.H{
			"event": h.currentEvent,
			"title": h.config.Event.Title,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	c.JSON(http.StatusOK, gin.H{"event": h.currentEvent})
}

func (h *Handler) AdminNext(c *gin.Context) {
	if h.currentEvent == nil || h.currentEvent.Status != "started" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active event"})
		return
	}

	nextQuestion := h.currentEvent.CurrentQuestion + 1
	if nextQuestion > len(h.config.Questions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No more questions"})
		return
	}

	err := h.eventRepo.UpdateCurrentQuestion(h.currentEvent.ID, nextQuestion)
	if err != nil {
		h.logger.LogError("updating current question", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update question"})
		return
	}

	h.currentEvent.CurrentQuestion = nextQuestion
	question := h.config.Questions[nextQuestion-1]

	h.logger.LogQuestionStart(nextQuestion, question.Text)

	message := websocket.Message{
		Type: "question_start",
		Data: gin.H{
			"question_number": nextQuestion,
			"question":        question,
			"total_questions": len(h.config.Questions),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	c.JSON(http.StatusOK, gin.H{
		"question_number": nextQuestion,
		"question":        question,
	})
}

func (h *Handler) AdminAlert(c *gin.Context) {
	if h.currentEvent == nil || h.currentEvent.Status != "started" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active event"})
		return
	}

	h.logger.LogAlert("5秒カウントダウン開始")

	// Start countdown from 5 seconds
	go h.startCountdown()

	c.JSON(http.StatusOK, gin.H{"status": "countdown started"})
}

func (h *Handler) startCountdown() {
	for i := 5; i >= 1; i-- {
		message := websocket.Message{
			Type: "countdown",
			Data: gin.H{
				"seconds_left": i,
			},
		}

		// Send only to screen clients
		messageBytes, _ := json.Marshal(message)
		h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

		// Wait 1 second
		time.Sleep(1 * time.Second)
	}

	// Send question_end message to all clients after countdown
	endMessage := websocket.Message{
		Type: "question_end",
		Data: gin.H{
			"message": "Time's up!",
		},
	}

	endMessageBytes, _ := json.Marshal(endMessage)
	h.hub.Broadcast <- endMessageBytes
}

func (h *Handler) AdminStop(c *gin.Context) {
	if h.currentEvent == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active event"})
		return
	}

	err := h.eventRepo.UpdateEventStatus(h.currentEvent.ID, "finished")
	if err != nil {
		h.logger.LogError("updating event status", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop event"})
		return
	}

	users, err := h.userRepo.GetAllUsers()
	if err != nil {
		h.logger.LogError("getting final results", err)
		users = []models.User{}
	}

	var teams []models.Team
	if h.config.Event.TeamMode {
		teams, err = h.teamAssignmentSvc.CalculateTeamScores()
		if err != nil {
			h.logger.LogError("calculating final team scores", err)
			teams = []models.Team{}
		}

		// Log team results
		for _, team := range teams {
			memberNames := make([]string, len(team.Members))
			for i, member := range team.Members {
				memberNames[i] = member.Nickname
			}
			h.logger.LogTeamResult(team.Name, team.Score, memberNames)
		}
	}

	h.logger.LogEventEnd(h.config.Event.Title, len(users), len(h.config.Questions))

	message := websocket.Message{
		Type: "final_results",
		Data: gin.H{
			"results":   users,
			"teams":     teams,
			"team_mode": h.config.Event.TeamMode,
			"event":     h.currentEvent,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	h.currentEvent.Status = "finished"

	c.JSON(http.StatusOK, gin.H{
		"event":   h.currentEvent,
		"results": users,
		"teams":   teams,
	})
}

func (h *Handler) GetStatus(c *gin.Context) {
	users, _ := h.userRepo.GetAllUsers()
	clientCounts := h.hub.GetClientCount()

	var teams []models.Team
	if h.config.Event.TeamMode {
		teams, _ = h.teamRepo.GetAllTeamsWithMembers()
	}

	c.JSON(http.StatusOK, gin.H{
		"event":         h.currentEvent,
		"users":         users,
		"teams":         teams,
		"client_counts": clientCounts,
		"config": gin.H{
			"team_mode": h.config.Event.TeamMode,
			"team_size": h.config.Event.TeamSize,
			"title":     h.config.Event.Title,
			"questions": h.config.Questions,
		},
	})
}

func (h *Handler) ParticipantWebSocket(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	user, err := h.userRepo.GetUserBySessionID(sessionID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	websocket.ServeWS(h.hub, c.Writer, c.Request, websocket.ClientTypeParticipant, user.ID, sessionID)
}

func (h *Handler) AdminWebSocket(c *gin.Context) {
	websocket.ServeWS(h.hub, c.Writer, c.Request, websocket.ClientTypeAdmin, 0, "admin")
}

func (h *Handler) ScreenWebSocket(c *gin.Context) {
	websocket.ServeWS(h.hub, c.Writer, c.Request, websocket.ClientTypeScreen, 0, "screen")
}

// Team Management API
func (h *Handler) AdminCreateTeams(c *gin.Context) {
	if h.currentEvent == nil || h.currentEvent.Status != "started" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active event"})
		return
	}

	if !h.config.Event.TeamMode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Team mode is not enabled"})
		return
	}

	teams, err := h.teamAssignmentSvc.CreateTeamsAndAssignUsers()
	if err != nil {
		h.logger.LogError("creating teams", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create teams"})
		return
	}

	if len(teams) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No users to assign to teams"})
		return
	}

	message := websocket.Message{
		Type: "team_assignment",
		Data: gin.H{
			"teams": teams,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	h.logger.LogTeamAssignment(len(teams), h.getTotalUsersInTeams(teams))
	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

func (h *Handler) GetTeams(c *gin.Context) {
	teams, err := h.teamRepo.GetAllTeamsWithMembers()
	if err != nil {
		h.logger.LogError("getting teams", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get teams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

func (h *Handler) getTotalUsersInTeams(teams []models.Team) int {
	total := 0
	for _, team := range teams {
		total += len(team.Members)
	}
	return total
}

// Health check endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	healthInfo := gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(time.Now().Add(-time.Duration(int64(time.Since(h.hub.StartTime).Seconds())) * time.Second)),
		"memory": gin.H{
			"alloc":       memStats.Alloc / 1024 / 1024,      // MB
			"total_alloc": memStats.TotalAlloc / 1024 / 1024, // MB
			"sys":         memStats.Sys / 1024 / 1024,        // MB
			"num_gc":      memStats.NumGC,
			"goroutines":  runtime.NumGoroutine(),
		},
		"database": gin.H{
			"connected": h.db != nil,
		},
		"websocket": h.hub.GetClientCount(),
	}

	c.JSON(http.StatusOK, healthInfo)
}

// Screen Quiz Info API
func (h *Handler) GetScreenInfo(c *gin.Context) {
	screenInfo := gin.H{
		"title":         h.config.Event.Title,
		"team_mode":     h.config.Event.TeamMode,
		"team_size":     h.config.Event.TeamSize,
		"qrcode":        h.config.Event.QrCode,
		"current_event": h.currentEvent,
	}

	c.JSON(http.StatusOK, screenInfo)
}

// Debug information endpoint
func (h *Handler) DebugInfo(c *gin.Context) {
	hubStats := h.hub.GetStatistics()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	debugInfo := gin.H{
		"system": gin.H{
			"go_version": runtime.Version(),
			"goroutines": runtime.NumGoroutine(),
			"memory": gin.H{
				"alloc":       memStats.Alloc,
				"total_alloc": memStats.TotalAlloc,
				"sys":         memStats.Sys,
				"heap_alloc":  memStats.HeapAlloc,
				"heap_sys":    memStats.HeapSys,
				"gc_runs":     memStats.NumGC,
			},
		},
		"websocket": hubStats,
		"database": gin.H{
			"connected": h.db != nil,
			"path":      h.db.GetPath(),
		},
		"event": gin.H{
			"current_event": h.currentEvent,
			"config":        h.config.Event,
		},
	}

	c.JSON(http.StatusOK, debugInfo)
}

// New State-Based Admin Action System
func (h *Handler) AdminAction(c *gin.Context) {
	var req AdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.Action {
	case "start_event":
		h.handleStartEvent(c)
	case "show_title":
		h.handleShowTitle(c)
	case "assign_teams":
		h.handleAssignTeams(c)
	case "next_question":
		h.handleNextQuestion(c)
	case "countdown_alert":
		h.handleCountdownAlert(c)
	case "show_answer_stats":
		h.handleShowAnswerStats(c)
	case "reveal_answer":
		h.handleRevealAnswer(c)
	case "show_results":
		h.handleShowResults(c)
	case "celebration":
		h.handleCelebration(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
	}
}

func (h *Handler) GetAvailableActions(c *gin.Context) {
	actions := h.stateManager.GetAvailableActions()
	c.JSON(http.StatusOK, gin.H{
		"available_actions": actions,
		"current_state":     h.stateManager.GetCurrentState(),
		"current_question":  h.stateManager.GetCurrentQuestion(),
	})
}

func (h *Handler) handleStartEvent(c *gin.Context) {
	if err := h.stateManager.TransitionTo(models.StateStarted); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event, err := h.eventRepo.CreateEvent(h.config.Event.Title, h.config.Event.TeamMode, h.config.Event.TeamSize, h.config.Event.QrCode)
	if err != nil {
		h.logger.LogError("creating event", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	h.currentEvent = event
	h.logger.LogEventStart(h.config.Event.Title, h.config.Event.TeamMode, 0)

	message := websocket.Message{
		Type: "event_started",
		Data: gin.H{
			"event": h.currentEvent,
			"title": h.config.Event.Title,
			"state": h.stateManager.GetCurrentState(),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	c.JSON(http.StatusOK, gin.H{
		"message": "イベントを開始しました",
		"event":   h.currentEvent,
		"state":   h.stateManager.GetCurrentState(),
	})
}

func (h *Handler) handleShowTitle(c *gin.Context) {
	if err := h.stateManager.TransitionTo(models.StateTitleDisplay); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := websocket.Message{
		Type: "title_display",
		Data: gin.H{
			"title": h.config.Event.Title,
			"state": h.stateManager.GetCurrentState(),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

	c.JSON(http.StatusOK, gin.H{
		"message": "タイトルを表示しました",
		"state":   h.stateManager.GetCurrentState(),
	})
}

func (h *Handler) handleAssignTeams(c *gin.Context) {
	if err := h.stateManager.TransitionTo(models.StateTeamAssignment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	teams, err := h.teamAssignmentSvc.CreateTeamsAndAssignUsers()
	if err != nil {
		h.logger.LogError("creating teams", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create teams"})
		return
	}

	message := websocket.Message{
		Type: "team_assignment",
		Data: gin.H{
			"teams": teams,
			"state": h.stateManager.GetCurrentState(),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	h.logger.LogTeamAssignment(len(teams), h.getTotalUsersInTeams(teams))

	c.JSON(http.StatusOK, gin.H{
		"message": "チーム分けを実行しました",
		"teams":   teams,
		"state":   h.stateManager.GetCurrentState(),
	})
}

func (h *Handler) handleNextQuestion(c *gin.Context) {
	if err := h.stateManager.NextQuestion(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	questionNum := h.stateManager.GetCurrentQuestion()
	if questionNum > len(h.config.Questions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No more questions"})
		return
	}

	question := h.config.Questions[questionNum-1]
	h.currentQuestion = &question

	if h.currentEvent != nil {
		err := h.eventRepo.UpdateCurrentQuestion(h.currentEvent.ID, questionNum)
		if err != nil {
			h.logger.LogError("updating current question", err)
		} else {
			h.currentEvent.CurrentQuestion = questionNum
		}
	}

	h.logger.LogQuestionStart(questionNum, question.Text)

	message := websocket.Message{
		Type: "question_start",
		Data: gin.H{
			"question_number": questionNum,
			"question":        question,
			"total_questions": len(h.config.Questions),
			"state":           h.stateManager.GetCurrentState(),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	c.JSON(http.StatusOK, gin.H{
		"message":       "次の問題を開始しました",
		"question_data": message.Data,
		"state":         h.stateManager.GetCurrentState(),
	})
}

func (h *Handler) handleCountdownAlert(c *gin.Context) {
	if err := h.stateManager.TransitionTo(models.StateCountdownActive); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.LogAlert("5秒カウントダウン開始")
	go h.startCountdownWithAutoTransition()

	c.JSON(http.StatusOK, gin.H{
		"message": "5秒カウントダウンを開始しました",
		"state":   h.stateManager.GetCurrentState(),
	})
}

func (h *Handler) startCountdownWithAutoTransition() {
	for i := 5; i >= 1; i-- {
		message := websocket.Message{
			Type: "countdown",
			Data: gin.H{
				"seconds_left": i,
			},
		}

		messageBytes, _ := json.Marshal(message)
		h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

		time.Sleep(1 * time.Second)
	}

	endMessage := websocket.Message{
		Type: "question_end",
		Data: gin.H{
			"message": "Time's up!",
		},
	}

	endMessageBytes, _ := json.Marshal(endMessage)
	h.hub.Broadcast <- endMessageBytes

	// 自動遷移
	h.stateManager.TransitionTo(models.StateAnswerStats)
}

func (h *Handler) handleShowAnswerStats(c *gin.Context) {
	// 既にStateAnswerStatsの場合はスキップ
	if h.stateManager.GetCurrentState() != models.StateAnswerStats {
		if err := h.stateManager.TransitionTo(models.StateAnswerStats); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	users, _ := h.userRepo.GetAllUsers()
	answeredCount := 0
	correctCount := 0
	currentQuestionNum := h.stateManager.GetCurrentQuestion()

	for _, user := range users {
		answer, _ := h.answerRepo.GetAnswerByUserAndQuestion(user.ID, currentQuestionNum)
		if answer != nil {
			answeredCount++
			if answer.IsCorrect {
				correctCount++
			}
		}
	}

	message := websocket.Message{
		Type: "answer_stats",
		Data: gin.H{
			"total_participants": len(users),
			"answered_count":     answeredCount,
			"correct_count":      correctCount,
			"correct_rate":       float64(correctCount) / float64(max(answeredCount, 1)) * 100,
			"state":              h.stateManager.GetCurrentState(),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

	c.JSON(http.StatusOK, gin.H{
		"message": "回答状況を表示しました",
		"state":   h.stateManager.GetCurrentState(),
	})
}

func (h *Handler) handleRevealAnswer(c *gin.Context) {
	if err := h.stateManager.TransitionTo(models.StateAnswerReveal); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.currentQuestion == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No current question"})
		return
	}

	message := websocket.Message{
		Type: "answer_reveal",
		Data: gin.H{
			"question":      h.currentQuestion,
			"correct_index": h.currentQuestion.Correct,
			"state":         h.stateManager.GetCurrentState(),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

	c.JSON(http.StatusOK, gin.H{
		"message": "回答を発表しました",
		"state":   h.stateManager.GetCurrentState(),
	})
}

func (h *Handler) handleShowResults(c *gin.Context) {
	if err := h.stateManager.TransitionTo(models.StateResults); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	users, err := h.userRepo.GetAllUsers()
	if err != nil {
		h.logger.LogError("getting final results", err)
		users = []models.User{}
	}

	var teams []models.Team
	if h.config.Event.TeamMode {
		teams, err = h.teamAssignmentSvc.CalculateTeamScores()
		if err != nil {
			h.logger.LogError("calculating final team scores", err)
			teams = []models.Team{}
		}
	}

	message := websocket.Message{
		Type: "final_results",
		Data: gin.H{
			"results":   users,
			"teams":     teams,
			"team_mode": h.config.Event.TeamMode,
			"state":     h.stateManager.GetCurrentState(),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	c.JSON(http.StatusOK, gin.H{
		"message": "結果を発表しました",
		"results": users,
		"teams":   teams,
		"state":   h.stateManager.GetCurrentState(),
	})
}

func (h *Handler) handleCelebration(c *gin.Context) {
	if err := h.stateManager.TransitionTo(models.StateCelebration); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := websocket.Message{
		Type: "celebration",
		Data: gin.H{
			"state": h.stateManager.GetCurrentState(),
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

	// 5秒後に自動的に終了状態に遷移
	go func() {
		time.Sleep(5 * time.Second)
		h.stateManager.TransitionTo(models.StateFinished)

		if h.currentEvent != nil {
			h.eventRepo.UpdateEventStatus(h.currentEvent.ID, "finished")
			h.currentEvent.Status = "finished"
		}

		users, _ := h.userRepo.GetAllUsers()
		h.logger.LogEventEnd(h.config.Event.Title, len(users), len(h.config.Questions))
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "🎉 お疲れ様でした！",
		"state":   h.stateManager.GetCurrentState(),
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Admin State Jump API
func (h *Handler) AdminJumpState(c *gin.Context) {
	var req JumpStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert string to EventState
	var targetState models.EventState
	switch req.State {
	case "waiting":
		targetState = models.StateWaiting
	case "started":
		targetState = models.StateStarted
	case "title_display":
		targetState = models.StateTitleDisplay
	case "team_assignment":
		targetState = models.StateTeamAssignment
	case "question_active":
		targetState = models.StateQuestionActive
	case "countdown_active":
		targetState = models.StateCountdownActive
	case "answer_stats":
		targetState = models.StateAnswerStats
	case "answer_reveal":
		targetState = models.StateAnswerReveal
	case "results":
		targetState = models.StateResults
	case "celebration":
		targetState = models.StateCelebration
	case "finished":
		targetState = models.StateFinished
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state: " + req.State})
		return
	}

	// Set question number if provided
	if req.QuestionNumber != nil {
		if err := h.stateManager.SetCurrentQuestion(*req.QuestionNumber); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question number: " + err.Error()})
			return
		}

		// Also update current event if it exists
		if h.currentEvent != nil {
			err := h.eventRepo.UpdateCurrentQuestion(h.currentEvent.ID, *req.QuestionNumber)
			if err != nil {
				h.logger.LogError("updating current question in database", err)
			} else {
				h.currentEvent.CurrentQuestion = *req.QuestionNumber
			}
		}
	}

	// Perform state jump
	if err := h.stateManager.JumpToState(targetState); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logMessage := fmt.Sprintf("Admin jumped to state: %s", req.State)
	if req.QuestionNumber != nil {
		logMessage += fmt.Sprintf(" (question: %d)", *req.QuestionNumber)
	}
	h.logger.LogAlert(logMessage)

	// Broadcast state change to all clients
	message := websocket.Message{
		Type: "state_changed",
		Data: gin.H{
			"new_state":        h.stateManager.GetCurrentState(),
			"current_question": h.stateManager.GetCurrentQuestion(),
			"jumped":           true,
		},
	}

	// Add current question data if jumping to question-related state
	if req.QuestionNumber != nil && *req.QuestionNumber > 0 && *req.QuestionNumber <= len(h.config.Questions) {
		question := h.config.Questions[*req.QuestionNumber-1]
		h.currentQuestion = &question
		message.Data.(gin.H)["question"] = question
		message.Data.(gin.H)["question_number"] = *req.QuestionNumber
		message.Data.(gin.H)["total_questions"] = len(h.config.Questions)
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	response := gin.H{
		"message":          fmt.Sprintf("ステート '%s' にジャンプしました", req.State),
		"new_state":        targetState,
		"current_question": h.stateManager.GetCurrentQuestion(),
	}

	if req.QuestionNumber != nil {
		response["message"] = fmt.Sprintf("ステート '%s' (問題%d) にジャンプしました", req.State, *req.QuestionNumber)
	}

	c.JSON(http.StatusOK, response)
}

// Get Available States for Jump
func (h *Handler) GetAvailableStates(c *gin.Context) {
	// Define all states with their Japanese display names
	allStates := []gin.H{
		{"value": "waiting", "label": "参加者待ち"},
		{"value": "started", "label": "イベント開始"},
		{"value": "title_display", "label": "タイトル表示"},
		{"value": "team_assignment", "label": "チーム分け"},
		{"value": "question_active", "label": "問題表示中"},
		{"value": "countdown_active", "label": "カウントダウン中"},
		{"value": "answer_stats", "label": "回答状況表示"},
		{"value": "answer_reveal", "label": "回答発表"},
		{"value": "results", "label": "結果発表"},
		{"value": "celebration", "label": "お疲れ様画面"},
		{"value": "finished", "label": "終了"},
	}

	c.JSON(http.StatusOK, gin.H{
		"available_states": allStates,
		"current_state":    h.stateManager.GetCurrentState(),
	})
}
