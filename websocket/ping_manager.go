package websocket

import (
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"time"
)

// PingTracker tracks individual ping requests
type PingTracker struct {
	PingID   string
	SentTime time.Time
	UserID   int
	Timeout  *time.Timer
}

// PingResult represents ping measurement results
type PingResult struct {
	UserID   int    `json:"user_id"`
	Nickname string `json:"nickname"`
	Latency  int64  `json:"latency"`
	Status   string `json:"status"`
}

// PingUserRepository interface for getting user information
type PingUserRepository interface {
	GetUserByID(id int) (*PingUser, error)
}

// PingUser represents a user for nickname lookup
type PingUser struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
}

// PingManager handles ping/pong operations
type PingManager struct {
	hubManager   *HubManager
	userRepo     PingUserRepository
	pingTrackers map[string]*PingTracker
	mutex        sync.RWMutex
	ticker       *time.Ticker
	stopCh       chan struct{}
}

// NewPingManager creates a new ping manager
func NewPingManager(hubManager *HubManager, userRepo PingUserRepository) *PingManager {
	return &PingManager{
		hubManager:   hubManager,
		userRepo:     userRepo,
		pingTrackers: make(map[string]*PingTracker),
		stopCh:       make(chan struct{}),
	}
}

// Start begins the ping monitoring system (10-second intervals)
func (pm *PingManager) Start() {
	pm.ticker = time.NewTicker(10 * time.Second)

	go func() {
		for {
			select {
			case <-pm.ticker.C:
				pm.sendPingToAllParticipants()
			case <-pm.stopCh:
				return
			}
		}
	}()

	log.Println("Ping manager started with 10-second intervals")
}

// Stop stops the ping monitoring system
func (pm *PingManager) Stop() {
	if pm.ticker != nil {
		pm.ticker.Stop()
	}
	close(pm.stopCh)
	log.Println("Ping manager stopped")
}

// sendPingToAllParticipants sends ping to all connected participants
func (pm *PingManager) sendPingToAllParticipants() {
	participants := pm.hubManager.GetClientsByType(ClientTypeParticipant)

	for _, client := range participants {
		pm.sendPingToClient(client)
	}
}

// sendPingToClient sends a ping message to a specific client
func (pm *PingManager) sendPingToClient(client *Client) {
	pingID := pm.generateUniqueID()
	sentTime := time.Now()

	// Create timeout timer (1 second)
	timeout := time.AfterFunc(1*time.Second, func() {
		pm.handlePingTimeout(client.UserID, pingID)
	})

	// Store ping tracker
	pm.mutex.Lock()
	pm.pingTrackers[pingID] = &PingTracker{
		PingID:   pingID,
		SentTime: sentTime,
		UserID:   client.UserID,
		Timeout:  timeout,
	}
	pm.mutex.Unlock()

	// Send ping message
	pingData := map[string]string{
		"ping_id": pingID,
	}

	err := pm.hubManager.BroadcastToUser(MessagePing, pingData, client.UserID)
	if err != nil {
		log.Printf("Failed to send ping to user %d: %v", client.UserID, err)
		// Clean up tracker on send failure
		pm.mutex.Lock()
		if tracker, exists := pm.pingTrackers[pingID]; exists {
			tracker.Timeout.Stop()
			delete(pm.pingTrackers, pingID)
		}
		pm.mutex.Unlock()
	}
}

// HandlePong processes pong responses from clients
func (pm *PingManager) HandlePong(pingID string, userID int) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	tracker, exists := pm.pingTrackers[pingID]
	if !exists {
		// Response received after timeout, ignore
		return
	}

	// Verify user ID matches
	if tracker.UserID != userID {
		log.Printf("User ID mismatch in pong response: expected %d, got %d", tracker.UserID, userID)
		return
	}

	// Calculate latency
	latency := time.Since(tracker.SentTime).Milliseconds()

	// Stop timeout timer
	tracker.Timeout.Stop()

	// Clean up tracker
	delete(pm.pingTrackers, pingID)

	// Determine connection status
	status := pm.determineConnectionStatus(latency)

	// Send result to admin clients
	pm.sendPingResultToAdmins(userID, latency, status)
}

// handlePingTimeout handles ping timeout (1 second)
func (pm *PingManager) handlePingTimeout(userID int, pingID string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Remove tracker (responses after this point are ignored)
	delete(pm.pingTrackers, pingID)

	// Send timeout result to admin clients
	pm.sendPingResultToAdmins(userID, 0, "bad") // Timeout is treated as "bad"
}

// determineConnectionStatus determines connection status based on latency
func (pm *PingManager) determineConnectionStatus(latency int64) string {
	if latency < 300 {
		return "good"
	} else if latency < 1000 {
		return "slow"
	}
	return "bad"
}

// sendPingResultToAdmins sends ping results to admin clients
func (pm *PingManager) sendPingResultToAdmins(userID int, latency int64, status string) {
	// Get user nickname from repository
	nickname := fmt.Sprintf("User%d", userID) // Default fallback
	if pm.userRepo != nil {
		if user, err := pm.userRepo.GetUserByID(userID); err == nil && user != nil {
			nickname = user.Nickname
		}
	}

	pingResult := PingResult{
		UserID:   userID,
		Nickname: nickname,
		Latency:  latency,
		Status:   status,
	}

	err := pm.hubManager.BroadcastToType(MessagePingResult, pingResult, ClientTypeAdmin)
	if err != nil {
		log.Printf("Failed to send ping result to admin: %v", err)
	}
}

// generateUniqueID generates a unique ID for ping requests
func (pm *PingManager) generateUniqueID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
