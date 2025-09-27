package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"quiz100/database"
	"quiz100/models"
	"quiz100/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func setupTestHandler() (*ParticipantHandlers, error) {
	// Create test database
	db, err := database.NewDatabase(":memory:")
	if err != nil {
		return nil, err
	}

	// Create test config
	config := &models.Config{
		Event: models.EventConfig{
			Title:    "Test Quiz",
			TeamMode: false,
			TeamSize: 5,
		},
		Questions: []models.Question{
			{
				Type:    "text",
				Text:    "Test question 1?",
				Choices: []string{"A", "B", "C", "D"},
				Correct: 1,
			},
			{
				Type:    "text",
				Text:    "Test question 2?",
				Choices: []string{"X", "Y", "Z"},
				Correct: 2,
			},
		},
	}

	// Create test logger
	logger, err := models.NewQuizLogger("logs")
	if err != nil {
		return nil, err
	}

	// Create WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create WebSocket hub manager
	hubManager := websocket.NewHubManager(hub)

	// Create repositories
	userRepo := models.NewUserRepository(db.DB)
	answerRepo := models.NewAnswerRepository(db.DB)
	emojiReactionRepo := models.NewEmojiReactionRepository(db.DB)

	return NewParticipantHandlers(userRepo, answerRepo, emojiReactionRepo, hubManager, *logger, config), nil
}

func TestHealthCheck(t *testing.T) {
	_, err := setupTestHandler()
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"memory":    "test",
			"websocket": "test",
		})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.NotNil(t, response["memory"])
	assert.NotNil(t, response["websocket"])
}

func TestJoinUser(t *testing.T) {
	handler, err := setupTestHandler()
	assert.NoError(t, err)

	router := gin.New()
	router.POST("/join", handler.Join)

	joinRequest := JoinRequest{
		Nickname: "TestUser",
	}
	jsonData, _ := json.Marshal(joinRequest)

	req, _ := http.NewRequest("POST", "/join", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["user"])
	assert.NotNil(t, response["session_id"])

	user := response["user"].(map[string]interface{})
	assert.Equal(t, "TestUser", user["nickname"])
	assert.Equal(t, float64(0), user["score"]) // JSON numbers are float64
}

func TestGetStatus(t *testing.T) {
	_, err := setupTestHandler()
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"config":        gin.H{},
			"users":         []interface{}{},
			"client_counts": gin.H{},
		})
	})

	req, _ := http.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["config"])
	assert.NotNil(t, response["users"])
	assert.NotNil(t, response["client_counts"])
}

func TestAnswerQuestion(t *testing.T) {
	handler, err := setupTestHandler()
	assert.NoError(t, err)

	router := gin.New()
	router.POST("/join", handler.Join)
	router.POST("/answer", handler.Answer)

	// First join a user
	joinRequest := JoinRequest{
		Nickname: "TestUser",
	}
	jsonData, _ := json.Marshal(joinRequest)

	req, _ := http.NewRequest("POST", "/join", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var joinResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &joinResponse)
	sessionID := joinResponse["session_id"].(string)

	// Then answer a question
	answerRequest := AnswerRequest{
		QuestionNumber: 1,
		AnswerIndex:    1, // Correct answer
	}
	jsonData, _ = json.Marshal(answerRequest)

	req, _ = http.NewRequest("POST", "/answer", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", sessionID)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["is_correct"])
	assert.Equal(t, float64(1), response["new_score"])
}

func TestSendEmoji(t *testing.T) {
	handler, err := setupTestHandler()
	assert.NoError(t, err)

	router := gin.New()
	router.POST("/join", handler.Join)
	router.POST("/emoji", handler.SendEmoji)

	// First join a user
	joinRequest := JoinRequest{
		Nickname: "TestUser",
	}
	jsonData, _ := json.Marshal(joinRequest)

	req, _ := http.NewRequest("POST", "/join", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var joinResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &joinResponse)
	sessionID := joinResponse["session_id"].(string)

	// Then send an emoji
	emojiRequest := EmojiRequest{
		Emoji: "😀",
	}
	jsonData, _ = json.Marshal(emojiRequest)

	req, _ = http.NewRequest("POST", "/emoji", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", sessionID)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "sent", response["status"])
}
