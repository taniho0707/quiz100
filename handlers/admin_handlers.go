package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"quiz100/models"
	"quiz100/services"
	"quiz100/websocket"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminHandlers contains handlers for admin-related operations
type AdminHandlers struct {
	eventRepo         *models.EventRepository
	userRepo          *models.UserRepository
	answerRepo        *models.AnswerRepository
	teamRepo          *models.TeamRepository
	teamAssignmentSvc *models.TeamAssignmentService
	hubManager        *websocket.HubManager
	stateService      *services.StateService
	logger            models.QuizLogger
	config            *models.Config
	currentEvent      *models.Event
	currentQuestion   *models.Question
	dbResetCallback   func() error
}

// AdminRequest represents a general admin action request
type AdminRequest struct {
	Action string `json:"action" binding:"required"`
}

// JumpStateRequest represents a state jump request
type JumpStateRequest struct {
	State          string `json:"state" binding:"required"`
	QuestionNumber *int   `json:"question_number,omitempty"`
}

// NewAdminHandlers creates a new AdminHandlers instance
func NewAdminHandlers(
	eventRepo *models.EventRepository,
	userRepo *models.UserRepository,
	answerRepo *models.AnswerRepository,
	teamRepo *models.TeamRepository,
	teamAssignmentSvc *models.TeamAssignmentService,
	hubManager *websocket.HubManager,
	stateService *services.StateService,
	logger models.QuizLogger,
	config *models.Config,
) *AdminHandlers {
	return &AdminHandlers{
		eventRepo:         eventRepo,
		userRepo:          userRepo,
		answerRepo:        answerRepo,
		teamRepo:          teamRepo,
		teamAssignmentSvc: teamAssignmentSvc,
		hubManager:        hubManager,
		stateService:      stateService,
		logger:            logger,
		config:            config,
	}
}

// AdminAction handles admin action requests using the new state-based system
func (ah *AdminHandlers) AdminAction(c *gin.Context) {
	var req AdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.Action {
	case "start_event":
		ah.handleStartEvent(c)
	case "show_title":
		ah.handleShowTitle(c)
	case "assign_teams":
		ah.handleAssignTeams(c)
	case "next_question":
		ah.handleNextQuestion(c)
	case "countdown_alert":
		ah.handleCountdownAlert(c)
	case "show_answer_stats":
		ah.handleShowAnswerStats(c)
	case "reveal_answer":
		ah.handleRevealAnswer(c)
	case "show_results":
		ah.handleShowResults(c)
	case "celebration":
		ah.handleCelebration(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
	}
}

// GetAvailableActions returns the available actions for the current state
func (ah *AdminHandlers) GetAvailableActions(c *gin.Context) {
	actions := ah.stateService.GetAvailableActions()
	c.JSON(http.StatusOK, gin.H{
		"available_actions": actions,
		"current_state":     ah.stateService.GetCurrentState(),
		"question_number":   ah.stateService.GetQuestionNumber(),
	})
}

// AdminJumpState handles admin state jumping (debug functionality)
func (ah *AdminHandlers) AdminJumpState(c *gin.Context) {
	var req JumpStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert string to EventState using constants
	targetState, err := models.StringToState(req.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set question number if provided
	if req.QuestionNumber != nil {
		if err := ah.stateService.SetQuestionNumber(*req.QuestionNumber); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question number: " + err.Error()})
			return
		}

		// Also update current event if it exists
		if ah.currentEvent != nil {
			err := ah.eventRepo.UpdateQuestionNumber(ah.currentEvent.ID, *req.QuestionNumber)
			if err != nil {
				ah.logger.LogError("updating current question in database", err)
			} else {
				ah.currentEvent.QuestionNumber = *req.QuestionNumber
			}
		}
	}

	// Perform state jump using state service
	result := ah.stateService.JumpToState(targetState)
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	// Broadcast state change with question data if jumping to question-related state
	stateData := gin.H{
		"new_state":       result.NewState,
		"question_number": ah.stateService.GetQuestionNumber(),
		"jumped":          true,
	}

	// Add current question data if jumping to question-related state
	if req.QuestionNumber != nil && *req.QuestionNumber > 0 && *req.QuestionNumber <= len(ah.config.Questions) {
		question := ah.config.Questions[*req.QuestionNumber-1]
		ah.currentQuestion = &question
		stateData["question"] = question
		stateData["question_number"] = *req.QuestionNumber
		stateData["total_questions"] = len(ah.config.Questions)
	}

	if err := ah.hubManager.BroadcastStateChanged(stateData); err != nil {
		ah.logger.LogError("broadcasting state change", err)
	}

	response := gin.H{
		"message":         result.Message,
		"new_state":       result.NewState,
		"question_number": ah.stateService.GetQuestionNumber(),
	}

	c.JSON(http.StatusOK, response)
}

// GetAvailableStates returns all available states for jumping
func (ah *AdminHandlers) GetAvailableStates(c *gin.Context) {
	// Use constants to get all states with their labels
	allStates := models.GetAllStatesWithLabels()

	c.JSON(http.StatusOK, gin.H{
		"available_states": allStates,
		"current_state":    ah.stateService.GetCurrentState(),
	})
}

// Private action handlers

func (ah *AdminHandlers) handleStartEvent(c *gin.Context) {
	result := ah.stateService.TransitionTo(models.StateStarted)
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	event, err := ah.eventRepo.CreateEvent(ah.config.Event.Title, ah.config.Event.TeamMode, ah.config.Event.TeamSize, ah.config.Event.QrCode)
	if err != nil {
		ah.logger.LogError("creating event", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	ah.currentEvent = event
	ah.logger.LogEventStart(ah.config.Event.Title, ah.config.Event.TeamMode, 0)

	eventData := gin.H{
		"event": ah.currentEvent,
		"title": ah.config.Event.Title,
		"state": ah.stateService.GetCurrentState(),
	}

	if err := ah.hubManager.BroadcastEventStarted(eventData); err != nil {
		ah.logger.LogError("broadcasting event started", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "イベントを開始しました",
		"event":   ah.currentEvent,
		"state":   ah.stateService.GetCurrentState(),
	})
}

func (ah *AdminHandlers) handleShowTitle(c *gin.Context) {
	result := ah.stateService.TransitionTo(models.StateTitleDisplay)
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	titleData := gin.H{
		"title": ah.config.Event.Title,
		"state": ah.stateService.GetCurrentState(),
	}

	if err := ah.hubManager.BroadcastTitleDisplay(titleData); err != nil {
		ah.logger.LogError("broadcasting title display", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "タイトルを表示しました",
		"state":   ah.stateService.GetCurrentState(),
	})
}

func (ah *AdminHandlers) handleAssignTeams(c *gin.Context) {
	result := ah.stateService.TransitionTo(models.StateTeamAssignment)
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	teams, err := ah.teamAssignmentSvc.CreateTeamsAndAssignUsers()
	if err != nil {
		ah.logger.LogError("creating teams", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create teams"})
		return
	}

	teamsData := gin.H{
		"teams": teams,
		"state": ah.stateService.GetCurrentState(),
	}

	if err := ah.hubManager.BroadcastTeamAssignment(teamsData); err != nil {
		ah.logger.LogError("broadcasting team assignment", err)
	}

	ah.logger.LogTeamAssignment(len(teams), ah.getTotalUsersInTeams(teams))

	c.JSON(http.StatusOK, gin.H{
		"message": "チーム分けを実行しました",
		"teams":   teams,
		"state":   ah.stateService.GetCurrentState(),
	})
}

func (ah *AdminHandlers) handleNextQuestion(c *gin.Context) {
	result := ah.stateService.NextQuestion()
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	questionNum := ah.stateService.GetQuestionNumber()
	if questionNum > len(ah.config.Questions) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No more questions"})
		return
	}

	question := ah.config.Questions[questionNum-1]
	ah.currentQuestion = &question

	if ah.currentEvent != nil {
		err := ah.eventRepo.UpdateQuestionNumber(ah.currentEvent.ID, questionNum)
		if err != nil {
			ah.logger.LogError("updating current question", err)
		} else {
			ah.currentEvent.QuestionNumber = questionNum
		}
	}

	ah.logger.LogQuestionStart(questionNum, question.Text)

	// websocket 本文
	questionAndAnswerData := gin.H{
		"question_number": questionNum,
		"question":        question,
		"total_questions": len(ah.config.Questions),
		"correct":         question.Correct,
	}

	questionData := gin.H{
		"question_number": questionNum,
		"question": gin.H{
			"type":    question.Type,
			"text":    question.Text,
			"image":   question.Image,
			"choices": question.Choices,
		},
	}

	if err := ah.hubManager.BroadcastQuestionStart(questionData, questionAndAnswerData); err != nil {
		ah.logger.LogError("broadcasting question start", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "次の問題を開始しました",
		"question": questionData,
		"state":    ah.stateService.GetCurrentState(),
	})
}

func (ah *AdminHandlers) handleCountdownAlert(c *gin.Context) {
	result := ah.stateService.StartCountdownWithAutoTransition()
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	ah.logger.LogAlert("5秒カウントダウン開始")

	c.JSON(http.StatusOK, gin.H{
		"message": "5秒カウントダウンを開始しました",
		"state":   ah.stateService.GetCurrentState(),
	})
}

func (ah *AdminHandlers) handleShowAnswerStats(c *gin.Context) {
	// The transition may have already occurred from countdown auto-transition
	if ah.stateService.GetCurrentState() != models.StateAnswerStats {
		result := ah.stateService.TransitionTo(models.StateAnswerStats)
		if !result.Success {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
			return
		}
	}

	users, _ := ah.userRepo.GetAllUsers()
	answeredCount := 0
	correctCount := 0
	currentQuestionNum := ah.stateService.GetQuestionNumber()

	// 現在の問題情報を取得
	if ah.currentQuestion == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No current question"})
		return
	}

	// 各選択肢の回答数をカウント
	choicesCounts := make([]int, len(ah.currentQuestion.Choices))

	for _, user := range users {
		answer, _ := ah.answerRepo.GetAnswerByUserAndQuestion(user.ID, currentQuestionNum)
		if answer != nil {
			answeredCount++
			if answer.IsCorrect {
				correctCount++
			}
			// 回答インデックス（1-based）を0-basedに変換してカウント
			if answer.AnswerIndex >= 1 && answer.AnswerIndex <= len(choicesCounts) {
				choicesCounts[answer.AnswerIndex-1]++
			}
		}
	}

	statsData := gin.H{
		"total_participants": len(users),
		"choices_counts":     choicesCounts,
	}

	if err := ah.hubManager.BroadcastAnswerStats(statsData); err != nil {
		ah.logger.LogError("broadcasting answer stats", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "回答状況を表示しました",
		"state":   ah.stateService.GetCurrentState(),
	})
}

func (ah *AdminHandlers) handleRevealAnswer(c *gin.Context) {
	result := ah.stateService.TransitionTo(models.StateAnswerReveal)
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	if ah.currentQuestion == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No current question"})
		return
	}

	revealData := gin.H{
		"correct": ah.currentQuestion.Correct,
	}

	if err := ah.hubManager.BroadcastAnswerReveal(revealData); err != nil {
		ah.logger.LogError("broadcasting answer reveal", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "回答を発表しました",
		"state":   ah.stateService.GetCurrentState(),
	})
}

func (ah *AdminHandlers) handleShowResults(c *gin.Context) {
	result := ah.stateService.TransitionTo(models.StateResults)
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	users, err := ah.userRepo.GetAllUsers()
	if err != nil {
		ah.logger.LogError("getting final results", err)
		users = []models.User{}
	}

	var teams []models.Team
	if ah.config.Event.TeamMode {
		teams, err = ah.teamAssignmentSvc.CalculateTeamScores()
		if err != nil {
			ah.logger.LogError("calculating final team scores", err)
			teams = []models.Team{}
		}
	}

	resultsData := gin.H{
		"results":   users,
		"teams":     teams,
		"team_mode": ah.config.Event.TeamMode,
	}

	if err := ah.hubManager.BroadcastFinalResults(resultsData); err != nil {
		ah.logger.LogError("broadcasting final results", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "結果を発表しました",
		"results": users,
		"teams":   teams,
		"state":   ah.stateService.GetCurrentState(),
	})
}

func (ah *AdminHandlers) handleCelebration(c *gin.Context) {
	result := ah.stateService.TransitionTo(models.StateCelebration)
	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}

	celebrationData := gin.H{
		"state": ah.stateService.GetCurrentState(),
	}

	if err := ah.hubManager.BroadcastCelebration(celebrationData); err != nil {
		ah.logger.LogError("broadcasting celebration", err)
	}

	// Auto-transition to finished after 5 seconds
	ah.stateService.AutoTransitionToCelebration(5)

	c.JSON(http.StatusOK, gin.H{
		"message": "🎉 お疲れ様でした！",
		"state":   ah.stateService.GetCurrentState(),
	})
}

// Helper methods

func (ah *AdminHandlers) getTotalUsersInTeams(teams []models.Team) int {
	total := 0
	for _, team := range teams {
		total += len(team.Members)
	}
	return total
}

// SetCurrentEvent sets the current event (for handlers that need it)
func (ah *AdminHandlers) SetCurrentEvent(event *models.Event) {
	ah.currentEvent = event
}

// SetDBResetCallback sets the callback function for database reset
func (ah *AdminHandlers) SetDBResetCallback(callback func() error) {
	ah.dbResetCallback = callback
}

// ResetDatabase handles database reset request
func (ah *AdminHandlers) ResetDatabase(c *gin.Context) {
	dbPath := "database/quiz.db"

	// Create backup
	backupDir := "logs"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		ah.logger.LogError("creating backup directory", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backup directory"})
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("quiz_backup_%s.db", timestamp))

	// Copy database file
	if err := copyFile(dbPath, backupPath); err != nil {
		ah.logger.LogError("creating database backup", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backup"})
		return
	}

	ah.logger.Info("Database backup created: %s", backupPath)

	// Broadcast reset notification to all clients
	resetData := gin.H{
		"message": "データベースがリセットされました。ページをリロードしてください。",
	}
	if err := ah.hubManager.BroadcastDatabaseReset(resetData); err != nil {
		ah.logger.LogError("broadcasting database reset", err)
	}

	// Wait a bit for the broadcast to complete
	time.Sleep(500 * time.Millisecond)

	// Execute the reset callback (this will trigger application restart)
	if ah.dbResetCallback != nil {
		go func() {
			// Wait for response to be sent
			time.Sleep(1 * time.Second)
			if err := ah.dbResetCallback(); err != nil {
				ah.logger.LogError("executing database reset", err)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "データベースをリセットしました",
		"backup_path": backupPath,
	})
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}
