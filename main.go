package main

import (
	"log"
	"net/http"

	"quiz100/database"
	"quiz100/handlers"
	"quiz100/middleware"
	"quiz100/models"
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

	// Clean up old logs at startup
	if err := logger.CleanupOldLogs(); err != nil {
		logger.Warning("Failed to cleanup old logs: %v", err)
	}

	db, err := database.NewDatabase("database/quiz.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	hub := websocket.NewHub()
	go hub.Run()

	handler := handlers.NewHandler(db, hub, config, logger)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(middleware.LogMACInfo())

	r.LoadHTMLGlob("static/html/*")
	r.Static("/css", "./static/css")
	r.Static("/js", "./static/js")
	r.Static("/images", "./static/images")

	r.GET("/", handler.GetParticipantPage)
	r.GET("/admin", middleware.AdminAuth(), handler.GetAdminPage)
	r.GET("/show", middleware.ScreenAuth(), handler.GetScreenPage)

	api := r.Group("/api")
	{
		api.POST("/join", handler.Join)
		api.POST("/answer", handler.Answer)
		api.POST("/emoji", handler.SendEmoji)
		api.POST("/reset-session", handler.ResetSession)
		api.GET("/status", handler.GetStatus)
		api.GET("/health", handler.HealthCheck)

		admin := api.Group("/admin")
		admin.Use(middleware.AdminAuth())
		{
			admin.POST("/start", handler.AdminStart)
			admin.POST("/next", handler.AdminNext)
			admin.POST("/alert", handler.AdminAlert)
			admin.POST("/stop", handler.AdminStop)
			admin.POST("/teams", handler.AdminCreateTeams)
			admin.GET("/teams", handler.GetTeams)
			admin.GET("/debug", handler.DebugInfo)
		}
	}

	ws := r.Group("/ws")
	{
		ws.GET("/participant", handler.ParticipantWebSocket)
		ws.GET("/admin", middleware.AdminAuth(), handler.AdminWebSocket)
		ws.GET("/screen", middleware.ScreenAuth(), handler.ScreenWebSocket)
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
