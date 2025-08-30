package handlers

import (
	"encoding/json"
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

	currentEvent *models.Event
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

func NewHandler(db *database.Database, hub *websocket.Hub, config *models.Config, logger *models.QuizLogger) *Handler {
	userRepo := models.NewUserRepository(db.DB)
	teamRepo := models.NewTeamRepository(db.DB)

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
	if existingUser != nil {
		user = existingUser
		h.logger.LogUserReconnect(user.Nickname, sessionID)
	} else {
		user, err = h.userRepo.CreateUser(sessionID, req.Nickname)
		if err != nil {
			h.logger.LogError("creating user", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
		h.logger.LogUserJoin(req.Nickname, sessionID)
	}

	err = h.userRepo.UpdateUserConnection(sessionID, true)
	if err != nil {
		h.logger.LogError("updating user connection", err)
	}

	message := websocket.Message{
		Type: "user_joined",
		Data: gin.H{
			"user":     user,
			"nickname": user.Nickname,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeAdmin)
	h.hub.BroadcastToType(messageBytes, websocket.ClientTypeScreen)

	c.JSON(http.StatusOK, gin.H{
		"user":       user,
		"session_id": sessionID,
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

	err = h.emojiReactionRepo.CreateReaction(user.ID, req.Emoji)
	if err != nil {
		h.logger.LogError("creating emoji reaction", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save emoji reaction"})
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

func (h *Handler) AdminStart(c *gin.Context) {
	if h.currentEvent != nil && h.currentEvent.Status == "started" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event already started"})
		return
	}

	event, err := h.eventRepo.CreateEvent(h.config.Event.Title, h.config.Event.TeamMode)
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

	h.logger.LogAlert("5秒アラート発動")

	message := websocket.Message{
		Type: "time_alert",
		Data: gin.H{
			"message": "残り時間5秒！",
			"seconds": 5,
		},
	}

	messageBytes, _ := json.Marshal(message)
	h.hub.Broadcast <- messageBytes

	c.JSON(http.StatusOK, gin.H{"status": "alert sent"})
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
		"config":        h.config.Event,
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
