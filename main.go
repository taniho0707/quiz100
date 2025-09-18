package main

import (
	"log"
	"net/http"

	"quiz100/database"
	"quiz100/handlers"
	"quiz100/middleware"
	"quiz100/models"
	"quiz100/services"
	"quiz100/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	config, err := models.LoadConfig("config/quiz.toml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := models.NewQuizLogger("logs")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	db, err := database.NewDatabase("database/quiz.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	userRepo := models.NewUserRepository(db.DB)
	answerRepo := models.NewAnswerRepository(db.DB)
	emojiReactionRepo := models.NewEmojiReactionRepository(db.DB)
	eventRepo := models.NewEventRepository(db.DB)
	teamRepo := models.NewTeamRepository(db.DB)

	// Initialize team assignment service
	teamAssignmentSvc := models.NewTeamAssignmentService(userRepo, teamRepo, config)

	// Initialize WebSocket hub and manager
	hub := websocket.NewHub()
	hubManager := websocket.NewHubManager(hub)

	// Initialize ping manager with adapter
	userRepoAdapter := websocket.NewUserRepositoryAdapter(userRepo)
	pingManager := websocket.NewPingManager(hubManager, userRepoAdapter)
	messageHandler := websocket.NewMessageHandler(pingManager)

	go hub.Run()
	go pingManager.Start()

	// Initialize state manager and service
	stateManager := models.NewEventStateManager(config.Event.TeamMode, len(config.Questions))
	stateService := services.NewStateService(stateManager, hubManager, hub, logger, config, userRepo, teamRepo)

	// Initialize split handlers
	participantHandlers := handlers.NewParticipantHandlers(userRepo, answerRepo, emojiReactionRepo, hubManager, *logger, config)
	adminHandlers := handlers.NewAdminHandlers(eventRepo, userRepo, answerRepo, teamRepo, teamAssignmentSvc, hubManager, stateService, *logger, config)
	websocketHandlers := handlers.NewWebSocketHandlers(hub, hubManager, messageHandler, userRepo, teamRepo, eventRepo, *logger, config, stateService)

	// Initialize current event (empty initially)
	var currentEvent *models.Event = nil
	adminHandlers.SetCurrentEvent(currentEvent)
	websocketHandlers.SetCurrentEvent(currentEvent)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(middleware.LogMACInfo())

	r.LoadHTMLGlob("static/html/*")
	r.Static("/css", "./static/css")
	r.Static("/js", "./static/js")
	r.Static("/images", "./static/images")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")

	// HTML page handlers (temporarily keeping old handler until we create page handlers)
	// TODO: Create dedicated page handlers or add to websocket handlers
	r.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "index.html", gin.H{}) })
	r.GET("/admin", middleware.AdminAuth(), func(c *gin.Context) { c.HTML(http.StatusOK, "admin.html", gin.H{}) })
	r.GET("/show", middleware.ScreenAuth(), func(c *gin.Context) { c.HTML(http.StatusOK, "screen.html", gin.H{}) })

	api := r.Group("/api")
	{
		// Participant API endpoints
		api.POST("/join", participantHandlers.Join)
		api.POST("/answer", participantHandlers.Answer)
		api.POST("/emoji", participantHandlers.SendEmoji)
		api.POST("/reset-session", participantHandlers.ResetSession)

		// General API endpoints
		api.GET("/status", websocketHandlers.GetStatus)
		api.GET("/health", websocketHandlers.HealthCheck)

		admin := api.Group("/admin")
		admin.Use(middleware.AdminAuth())
		{
			// Legacy endpoints (to be deprecated)
			// admin.POST("/start", handler.AdminStart)
			// admin.POST("/next", handler.AdminNext)
			// admin.POST("/alert", handler.AdminAlert)
			// admin.POST("/stop", handler.AdminStop)
			// admin.POST("/teams", handler.AdminCreateTeams)

			admin.GET("/teams", websocketHandlers.GetTeams)
			admin.GET("/debug", websocketHandlers.DebugInfo)

			// New State-Based Action System
			admin.POST("/action", adminHandlers.AdminAction)
			admin.GET("/actions", adminHandlers.GetAvailableActions)

			// Debug State Jump System
			admin.POST("/jump-state", adminHandlers.AdminJumpState)
			admin.GET("/available-states", adminHandlers.GetAvailableStates)
		}

		screen := api.Group("/screen")
		screen.Use(middleware.ScreenAuth())
		{
			screen.GET("/info", websocketHandlers.GetScreenInfo)
		}
	}

	ws := r.Group("/ws")
	{
		ws.GET("/participant", websocketHandlers.ParticipantWebSocket)
		ws.GET("/admin", middleware.AdminAuth(), websocketHandlers.AdminWebSocket)
		ws.GET("/screen", middleware.ScreenAuth(), websocketHandlers.ScreenWebSocket)

		// State synchronization endpoints
		ws.GET("/sync-status", middleware.AdminAuth(), websocketHandlers.GetSyncStatus)
		ws.POST("/sync-client", middleware.AdminAuth(), websocketHandlers.RequestClientSync)
		ws.POST("/sync-all", middleware.AdminAuth(), websocketHandlers.SyncAllClients)
		ws.GET("/sync-check/:user_id", middleware.AdminAuth(), websocketHandlers.CheckClientSync)
	}

	logger.Info("=== QUIZ SYSTEM STARTING ===")
	logger.Info("Event: %s", config.Event.Title)
	logger.Info("Team mode: %v", config.Event.TeamMode)
	logger.Info("Team size: %d", config.Event.TeamSize)
	logger.Info("Questions: %d", len(config.Questions))
	logger.Info("Avoid groups: %v", config.TeamSeparation.AvoidGroups)
	logger.Info("Server starting on :8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
