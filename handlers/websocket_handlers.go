package handlers

import (
	"fmt"
	"net/http"
	"quiz100/models"
	"quiz100/services"
	"quiz100/websocket"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// WebSocketHandlers contains handlers for WebSocket connections and utility endpoints
type WebSocketHandlers struct {
	hub            *websocket.Hub
	hubManager     *websocket.HubManager
	messageHandler *websocket.MessageHandler
	userRepo       *models.UserRepository
	teamRepo       *models.TeamRepository
	eventRepo      *models.EventRepository
	logger         models.QuizLogger
	config         *models.Config
	currentEvent   *models.Event
	stateService   *services.StateService
}

// NewWebSocketHandlers creates a new WebSocketHandlers instance
func NewWebSocketHandlers(
	hub *websocket.Hub,
	hubManager *websocket.HubManager,
	messageHandler *websocket.MessageHandler,
	userRepo *models.UserRepository,
	teamRepo *models.TeamRepository,
	eventRepo *models.EventRepository,
	logger models.QuizLogger,
	config *models.Config,
	stateService *services.StateService,
) *WebSocketHandlers {
	return &WebSocketHandlers{
		hub:            hub,
		hubManager:     hubManager,
		messageHandler: messageHandler,
		userRepo:       userRepo,
		teamRepo:       teamRepo,
		eventRepo:      eventRepo,
		logger:         logger,
		config:         config,
		stateService:   stateService,
	}
}

// ParticipantWebSocket handles WebSocket connections for participants
func (wh *WebSocketHandlers) ParticipantWebSocket(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	user, err := wh.userRepo.GetUserBySessionID(sessionID)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	websocket.ServeWS(wh.hub, c.Writer, c.Request, websocket.ClientTypeParticipant, user.ID, sessionID, wh.messageHandler)
}

// AdminWebSocket handles WebSocket connections for admin clients
func (wh *WebSocketHandlers) AdminWebSocket(c *gin.Context) {
	websocket.ServeWS(wh.hub, c.Writer, c.Request, websocket.ClientTypeAdmin, 0, "admin", wh.messageHandler)
}

// ScreenWebSocket handles WebSocket connections for screen display
func (wh *WebSocketHandlers) ScreenWebSocket(c *gin.Context) {
	websocket.ServeWS(wh.hub, c.Writer, c.Request, websocket.ClientTypeScreen, 0, "screen", wh.messageHandler)
}

// GetStatus returns the current system status
func (wh *WebSocketHandlers) GetStatus(c *gin.Context) {
	users, _ := wh.userRepo.GetAllUsers()
	clientCounts := wh.hubManager.GetClientCount()

	var teams []models.Team
	if wh.config.Event.TeamMode {
		teams, _ = wh.teamRepo.GetAllTeamsWithMembers()
	}

	c.JSON(http.StatusOK, gin.H{
		"event":         wh.currentEvent,
		"users":         users,
		"teams":         teams,
		"client_counts": clientCounts,
		"config": gin.H{
			"team_mode": wh.config.Event.TeamMode,
			"team_size": wh.config.Event.TeamSize,
			"title":     wh.config.Event.Title,
			"questions": wh.config.Questions,
		},
	})
}

// GetScreenInfo returns information for the screen display
func (wh *WebSocketHandlers) GetScreenInfo(c *gin.Context) {
	screenInfo := gin.H{
		"title":         wh.config.Event.Title,
		"team_mode":     wh.config.Event.TeamMode,
		"team_size":     wh.config.Event.TeamSize,
		"qrcode":        wh.config.Event.QrCode,
		"current_event": wh.currentEvent,
	}

	c.JSON(http.StatusOK, screenInfo)
}

// HealthCheck returns system health information
func (wh *WebSocketHandlers) HealthCheck(c *gin.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	healthInfo := gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(time.Now().Add(-time.Duration(int64(time.Since(wh.hub.StartTime).Seconds())) * time.Second)),
		"memory": gin.H{
			"alloc":       memStats.Alloc / 1024 / 1024,      // MB
			"total_alloc": memStats.TotalAlloc / 1024 / 1024, // MB
			"sys":         memStats.Sys / 1024 / 1024,        // MB
			"num_gc":      memStats.NumGC,
			"goroutines":  runtime.NumGoroutine(),
		},
		"database": gin.H{
			"connected": true, // Assuming database is connected if we got this far
		},
		"websocket": wh.hubManager.GetClientCount(),
	}

	c.JSON(http.StatusOK, healthInfo)
}

// DebugInfo returns debug information about the system
func (wh *WebSocketHandlers) DebugInfo(c *gin.Context) {
	hubStats := wh.hubManager.GetStatistics()

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
			"connected": true, // Simplified for now
		},
		"event": gin.H{
			"current_event": wh.currentEvent,
			"config":        wh.config.Event,
		},
	}

	c.JSON(http.StatusOK, debugInfo)
}

// GetTeams returns all teams with their members
func (wh *WebSocketHandlers) GetTeams(c *gin.Context) {
	teams, err := wh.teamRepo.GetAllTeamsWithMembers()
	if err != nil {
		wh.logger.LogError("getting teams", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get teams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

// SetCurrentEvent sets the current event (for handlers that need it)
func (wh *WebSocketHandlers) SetCurrentEvent(event *models.Event) {
	wh.currentEvent = event
}

// State Synchronization Endpoints

// GetSyncStatus returns synchronization status for all clients
func (wh *WebSocketHandlers) GetSyncStatus(c *gin.Context) {
	if wh.stateService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "State service not available",
		})
		return
	}

	syncReport := wh.stateService.GetSynchronizationReport()
	c.JSON(http.StatusOK, syncReport)
}

// RequestClientSync manually requests synchronization for a specific client
func (wh *WebSocketHandlers) RequestClientSync(c *gin.Context) {
	if wh.stateService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "State service not available",
		})
		return
	}

	var request struct {
		UserID   int    `json:"user_id" binding:"required"`
		SyncType string `json:"sync_type"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if request.SyncType == "" {
		request.SyncType = "manual"
	}

	wh.stateService.RequestClientSync(request.UserID, request.SyncType)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Sync requested successfully",
		"user_id":   request.UserID,
		"sync_type": request.SyncType,
	})
}

// SyncAllClients forces synchronization for all connected participants
func (wh *WebSocketHandlers) SyncAllClients(c *gin.Context) {
	if wh.stateService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "State service not available",
		})
		return
	}

	wh.stateService.SyncAllClients()

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync requested for all participants",
	})
}

// CheckClientSync checks if a specific client is synchronized
func (wh *WebSocketHandlers) CheckClientSync(c *gin.Context) {
	if wh.stateService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "State service not available",
		})
		return
	}

	userIDStr := c.Param("user_id")
	userID := 0
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	isSynchronized := wh.stateService.IsClientSynchronized(userID)

	c.JSON(http.StatusOK, gin.H{
		"user_id":          userID,
		"synchronized":     isSynchronized,
		"current_state":    wh.stateService.GetCurrentState(),
		"current_question": wh.stateService.GetCurrentQuestion(),
	})
}
