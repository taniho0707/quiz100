package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	DefaultUsers        = 100
	DefaultTestDuration = 5 * time.Minute
	ServerURL           = "http://localhost:8080"
	WebSocketURL        = "ws://localhost:8080"
)

// 通信障害のタイプ
type NetworkIssueType int

const (
	NetworkNormal      NetworkIssueType = iota
	NetworkSlow                         // 遅延
	NetworkUnstable                     // 不安定（パケットロス）
	NetworkInterrupted                  // 断続的切断
	NetworkOffline                      // 完全オフライン
)

type AdvancedLoadTester struct {
	ServerURL    string
	WebSocketURL string
	Users        int
	TestDuration time.Duration

	// 高度な設定
	NetworkProfiles map[NetworkIssueType]NetworkProfile
	UserProfiles    []UserProfile

	// 統計情報
	mu                sync.RWMutex
	ConnectedUsers    int
	DisconnectedUsers int
	TotalMessages     int64
	TotalErrors       int64
	ResponseTimes     []time.Duration
	NetworkEventLog   []NetworkEvent

	startTime time.Time
	stopChan  chan bool
}

type NetworkProfile struct {
	Name           string
	Probability    int           // この状態になる確率(%)
	Latency        time.Duration // 追加遅延
	PacketLoss     int           // パケットロス率(%)
	Disconnection  int           // 切断頻度(%)
	ReconnectDelay time.Duration // 再接続までの待機時間
}

type UserProfile struct {
	ID                int
	Nickname          string
	NetworkType       NetworkIssueType
	DeviceType        string // "mobile", "tablet", "desktop"
	ConnectionQuality string // "excellent", "good", "poor", "terrible"
	BehaviorPattern   string // "active", "passive", "sporadic"
}

type UserSession struct {
	ID        int
	Nickname  string
	SessionID string
	Conn      *websocket.Conn
	Profile   UserProfile

	JoinTime       time.Time
	MessagesSent   int64
	ErrorCount     int64
	ResponseTimes  []time.Duration
	NetworkEvents  []NetworkEvent
	LastActivity   time.Time
	ReconnectCount int
}

type NetworkEvent struct {
	Timestamp   time.Time
	UserID      int
	EventType   string
	Description string
	Latency     time.Duration
}

type TestScenario struct {
	Name         string
	Description  string
	UserProfiles []UserProfile
	Duration     time.Duration
}

// リクエストタイプ
type JoinRequest struct {
	Nickname string `json:"nickname"`
}

type AnswerRequest struct {
	QuestionNumber int `json:"question_number"`
	AnswerIndex    int `json:"answer_index"`
}

type EmojiRequest struct {
	Emoji string `json:"emoji"`
}

func NewAdvancedLoadTester() *AdvancedLoadTester {
	return &AdvancedLoadTester{
		ServerURL:       ServerURL,
		WebSocketURL:    WebSocketURL,
		Users:           DefaultUsers,
		TestDuration:    DefaultTestDuration,
		stopChan:        make(chan bool),
		ResponseTimes:   make([]time.Duration, 0),
		NetworkEventLog: make([]NetworkEvent, 0),
		NetworkProfiles: map[NetworkIssueType]NetworkProfile{
			NetworkNormal: {
				Name:           "正常接続",
				Probability:    60,
				Latency:        0,
				PacketLoss:     0,
				Disconnection:  0,
				ReconnectDelay: 0,
			},
			NetworkSlow: {
				Name:           "低速接続",
				Probability:    20,
				Latency:        500 * time.Millisecond,
				PacketLoss:     5,
				Disconnection:  5,
				ReconnectDelay: 2 * time.Second,
			},
			NetworkUnstable: {
				Name:           "不安定接続",
				Probability:    15,
				Latency:        200 * time.Millisecond,
				PacketLoss:     15,
				Disconnection:  20,
				ReconnectDelay: 3 * time.Second,
			},
			NetworkInterrupted: {
				Name:           "断続的切断",
				Probability:    4,
				Latency:        100 * time.Millisecond,
				PacketLoss:     30,
				Disconnection:  40,
				ReconnectDelay: 5 * time.Second,
			},
			NetworkOffline: {
				Name:           "オフライン",
				Probability:    1,
				Latency:        0,
				PacketLoss:     100,
				Disconnection:  100,
				ReconnectDelay: 10 * time.Second,
			},
		},
	}
}

func (alt *AdvancedLoadTester) GenerateUserProfiles() {
	alt.UserProfiles = make([]UserProfile, alt.Users)

	deviceTypes := []string{"mobile", "tablet", "desktop"}
	connectionQualities := []string{"excellent", "good", "poor", "terrible"}
	behaviorPatterns := []string{"active", "passive", "sporadic"}

	for i := 0; i < alt.Users; i++ {
		// ネットワークタイプを確率に基づいて決定
		networkType := alt.selectNetworkType()

		profile := UserProfile{
			ID:                i + 1,
			Nickname:          fmt.Sprintf("TestUser%03d", i+1),
			NetworkType:       networkType,
			DeviceType:        deviceTypes[rand.Intn(len(deviceTypes))],
			ConnectionQuality: connectionQualities[rand.Intn(len(connectionQualities))],
			BehaviorPattern:   behaviorPatterns[rand.Intn(len(behaviorPatterns))],
		}

		alt.UserProfiles[i] = profile
	}
}

func (alt *AdvancedLoadTester) selectNetworkType() NetworkIssueType {
	rand := rand.Intn(100)
	cumulative := 0

	for networkType, profile := range alt.NetworkProfiles {
		cumulative += profile.Probability
		if rand < cumulative {
			return networkType
		}
	}

	return NetworkNormal
}

func (alt *AdvancedLoadTester) Start() {
	log.Printf("🚀 高度な負荷テスト開始: %d人のユーザーで %v間実行", alt.Users, alt.TestDuration)
	log.Printf("🌐 サーバーURL: %s", alt.ServerURL)

	// ユーザープロファイルを生成
	alt.GenerateUserProfiles()
	alt.printNetworkProfileSummary()

	alt.startTime = time.Now()

	var wg sync.WaitGroup

	// テスト終了タイマー
	go func() {
		time.Sleep(alt.TestDuration)
		close(alt.stopChan)
	}()

	// ユーザーセッションを並行実行
	for i, profile := range alt.UserProfiles {
		wg.Add(1)
		go func(userProfile UserProfile) {
			defer wg.Done()
			alt.simulateAdvancedUser(userProfile)
		}(profile)

		// 接続を段階的に増やす
		if i%5 == 4 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	// 統計情報とネットワークイベントを定期的に出力
	go alt.printAdvancedStats()
	go alt.monitorNetworkEvents()

	wg.Wait()

	log.Printf("✅ 高度な負荷テスト完了")
	alt.printAdvancedFinalReport()
}

func (alt *AdvancedLoadTester) simulateAdvancedUser(profile UserProfile) {
	user := &UserSession{
		ID:            profile.ID,
		Nickname:      profile.Nickname,
		Profile:       profile,
		JoinTime:      time.Now(),
		ResponseTimes: make([]time.Duration, 0),
		NetworkEvents: make([]NetworkEvent, 0),
		LastActivity:  time.Now(),
	}

	networkProfile := alt.NetworkProfiles[profile.NetworkType]
	log.Printf("👤 ユーザー %s 開始 (%s, %s, %s)",
		profile.Nickname, networkProfile.Name, profile.DeviceType, profile.ConnectionQuality)

	// 1. ユーザー登録
	if !alt.joinAdvancedUser(user) {
		return
	}

	// 2. WebSocket接続
	if !alt.connectAdvancedWebSocket(user) {
		return
	}
	defer alt.closeConnection(user)

	alt.mu.Lock()
	alt.ConnectedUsers++
	alt.mu.Unlock()

	// 3. 高度なユーザー行動をシミュレート
	alt.runAdvancedUserActions(user)

	alt.mu.Lock()
	alt.ConnectedUsers--
	alt.DisconnectedUsers++
	alt.mu.Unlock()
}

func (alt *AdvancedLoadTester) joinAdvancedUser(user *UserSession) bool {
	networkProfile := alt.NetworkProfiles[user.Profile.NetworkType]

	// ネットワーク遅延をシミュレート
	if networkProfile.Latency > 0 {
		time.Sleep(networkProfile.Latency)
	}

	// パケットロスをシミュレート
	if rand.Intn(100) < networkProfile.PacketLoss {
		alt.recordNetworkEvent(user, "PacketLoss", "Join request packet lost", networkProfile.Latency)
		alt.recordError(user, "join request failed: packet loss")
		return false
	}

	joinReq := JoinRequest{Nickname: user.Nickname}
	reqBody, _ := json.Marshal(joinReq)

	startTime := time.Now()

	// カスタムHTTPクライアント（タイムアウト設定）
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 接続レベルでの遅延をシミュレート
				if networkProfile.Latency > 0 {
					time.Sleep(networkProfile.Latency / 2)
				}
				return (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext(ctx, network, addr)
			},
		},
	}

	resp, err := client.Post(alt.ServerURL+"/api/join", "application/json", bytes.NewBuffer(reqBody))
	responseTime := time.Since(startTime)

	if err != nil {
		alt.recordNetworkEvent(user, "ConnectionError", fmt.Sprintf("Join failed: %v", err), responseTime)
		alt.recordError(user, fmt.Sprintf("join request failed: %v", err))
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		alt.recordError(user, fmt.Sprintf("join failed with status: %d", resp.StatusCode))
		return false
	}

	var joinResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&joinResp); err != nil {
		alt.recordError(user, fmt.Sprintf("join response decode failed: %v", err))
		return false
	}

	user.SessionID = joinResp["session_id"].(string)
	user.ResponseTimes = append(user.ResponseTimes, responseTime)
	user.LastActivity = time.Now()

	alt.mu.Lock()
	alt.ResponseTimes = append(alt.ResponseTimes, responseTime)
	alt.mu.Unlock()

	alt.recordNetworkEvent(user, "JoinSuccess", "User joined successfully", responseTime)
	return true
}

func (alt *AdvancedLoadTester) connectAdvancedWebSocket(user *UserSession) bool {
	networkProfile := alt.NetworkProfiles[user.Profile.NetworkType]
	wsURL := fmt.Sprintf("%s/ws/participant?session_id=%s", alt.WebSocketURL, user.SessionID)

	// ネットワーク遅延をシミュレート
	if networkProfile.Latency > 0 {
		time.Sleep(networkProfile.Latency)
	}

	startTime := time.Now()

	// カスタムダイヤラーでネットワーク状況をシミュレート
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 接続失敗をシミュレート
			if rand.Intn(100) < networkProfile.Disconnection {
				return nil, fmt.Errorf("simulated connection failure")
			}

			return (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext(ctx, network, addr)
		},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	responseTime := time.Since(startTime)

	if err != nil {
		alt.recordNetworkEvent(user, "WebSocketConnectionError", fmt.Sprintf("Connection failed: %v", err), responseTime)
		alt.recordError(user, fmt.Sprintf("websocket connection failed: %v", err))
		return false
	}

	user.Conn = conn
	user.ResponseTimes = append(user.ResponseTimes, responseTime)
	user.LastActivity = time.Now()

	alt.mu.Lock()
	alt.ResponseTimes = append(alt.ResponseTimes, responseTime)
	alt.mu.Unlock()

	alt.recordNetworkEvent(user, "WebSocketConnected", "WebSocket connection established", responseTime)
	return true
}

func (alt *AdvancedLoadTester) runAdvancedUserActions(user *UserSession) {
	// WebSocketメッセージを受信するgoroutine
	go alt.handleAdvancedWebSocketMessages(user)

	// ユーザーの行動パターンに基づいてアクション間隔を調整
	var actionInterval time.Duration
	switch user.Profile.BehaviorPattern {
	case "active":
		actionInterval = time.Duration(rand.Intn(2000)+500) * time.Millisecond
	case "passive":
		actionInterval = time.Duration(rand.Intn(8000)+3000) * time.Millisecond
	case "sporadic":
		actionInterval = time.Duration(rand.Intn(10000)+1000) * time.Millisecond
	default:
		actionInterval = time.Duration(rand.Intn(3000)+1000) * time.Millisecond
	}

	ticker := time.NewTicker(actionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-alt.stopChan:
			return
		case <-ticker.C:
			networkProfile := alt.NetworkProfiles[user.Profile.NetworkType]

			// 切断イベントをシミュレート
			if rand.Intn(100) < networkProfile.Disconnection {
				alt.simulateAdvancedNetworkIssue(user)
				continue
			}

			// ランダムなアクションを実行
			alt.performRandomAction(user)
		}
	}
}

func (alt *AdvancedLoadTester) handleAdvancedWebSocketMessages(user *UserSession) {
	for {
		select {
		case <-alt.stopChan:
			return
		default:
			if user.Conn == nil {
				return
			}

			user.Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, message, err := user.Conn.ReadMessage()

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					alt.recordNetworkEvent(user, "WebSocketReadError", fmt.Sprintf("Read error: %v", err), 0)
					alt.recordError(user, fmt.Sprintf("websocket read error: %v", err))
				}
				return
			}

			// メッセージを処理
			alt.mu.Lock()
			alt.TotalMessages++
			user.MessagesSent++
			user.LastActivity = time.Now()
			alt.mu.Unlock()

			alt.recordNetworkEvent(user, "MessageReceived", fmt.Sprintf("Received %d bytes", len(message)), 0)
		}
	}
}

func (alt *AdvancedLoadTester) performRandomAction(user *UserSession) {
	networkProfile := alt.NetworkProfiles[user.Profile.NetworkType]

	// ネットワーク遅延をシミュレート
	if networkProfile.Latency > 0 {
		time.Sleep(networkProfile.Latency)
	}

	actions := []string{"answer", "emoji", "idle"}
	weights := map[string]int{
		"answer": 40,
		"emoji":  30,
		"idle":   30,
	}

	// 行動パターンに基づいて重み調整
	switch user.Profile.BehaviorPattern {
	case "active":
		weights["answer"] = 60
		weights["emoji"] = 30
		weights["idle"] = 10
	case "passive":
		weights["answer"] = 20
		weights["emoji"] = 10
		weights["idle"] = 70
	}

	action := alt.weightedRandomChoice(actions, weights)

	switch action {
	case "answer":
		alt.sendAdvancedAnswer(user)
	case "emoji":
		alt.sendAdvancedEmoji(user)
	case "idle":
		// アイドル状態
		alt.recordNetworkEvent(user, "Idle", "User idle", 0)
	}
}

func (alt *AdvancedLoadTester) weightedRandomChoice(choices []string, weights map[string]int) string {
	totalWeight := 0
	for _, choice := range choices {
		totalWeight += weights[choice]
	}

	randomValue := rand.Intn(totalWeight)
	cumulative := 0

	for _, choice := range choices {
		cumulative += weights[choice]
		if randomValue < cumulative {
			return choice
		}
	}

	return choices[0]
}

func (alt *AdvancedLoadTester) sendAdvancedAnswer(user *UserSession) {
	networkProfile := alt.NetworkProfiles[user.Profile.NetworkType]

	// パケットロスをシミュレート
	if rand.Intn(100) < networkProfile.PacketLoss {
		alt.recordNetworkEvent(user, "PacketLoss", "Answer request packet lost", 0)
		alt.recordError(user, "answer request failed: packet loss")
		return
	}

	answerReq := AnswerRequest{
		QuestionNumber: rand.Intn(5) + 1,
		AnswerIndex:    rand.Intn(4) + 1,
	}

	reqBody, _ := json.Marshal(answerReq)
	req, _ := http.NewRequest("POST", alt.ServerURL+"/api/answer", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", user.SessionID)

	startTime := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		alt.recordNetworkEvent(user, "AnswerRequestError", fmt.Sprintf("Failed: %v", err), responseTime)
		alt.recordError(user, fmt.Sprintf("answer request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	user.ResponseTimes = append(user.ResponseTimes, responseTime)
	user.LastActivity = time.Now()

	alt.mu.Lock()
	alt.ResponseTimes = append(alt.ResponseTimes, responseTime)
	alt.mu.Unlock()

	alt.recordNetworkEvent(user, "AnswerSent", fmt.Sprintf("Answer sent (Q%d, A%d)", answerReq.QuestionNumber, answerReq.AnswerIndex), responseTime)
}

func (alt *AdvancedLoadTester) sendAdvancedEmoji(user *UserSession) {
	networkProfile := alt.NetworkProfiles[user.Profile.NetworkType]

	// パケットロスをシミュレート
	if rand.Intn(100) < networkProfile.PacketLoss {
		alt.recordNetworkEvent(user, "PacketLoss", "Emoji request packet lost", 0)
		return
	}

	emojis := []string{"😀", "😂", "🎉", "👍", "❤️", "🔥", "💪", "🎊", "🤔", "😅"}
	emoji := emojis[rand.Intn(len(emojis))]

	emojiReq := EmojiRequest{Emoji: emoji}
	reqBody, _ := json.Marshal(emojiReq)
	req, _ := http.NewRequest("POST", alt.ServerURL+"/api/emoji", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", user.SessionID)

	startTime := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		alt.recordNetworkEvent(user, "EmojiRequestError", fmt.Sprintf("Failed: %v", err), responseTime)
		alt.recordError(user, fmt.Sprintf("emoji request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	user.ResponseTimes = append(user.ResponseTimes, responseTime)
	user.LastActivity = time.Now()

	alt.mu.Lock()
	alt.ResponseTimes = append(alt.ResponseTimes, responseTime)
	alt.mu.Unlock()

	alt.recordNetworkEvent(user, "EmojiSent", fmt.Sprintf("Emoji sent: %s", emoji), responseTime)
}

func (alt *AdvancedLoadTester) simulateAdvancedNetworkIssue(user *UserSession) {
	networkProfile := alt.NetworkProfiles[user.Profile.NetworkType]

	log.Printf("🔄 %s: %s - ネットワーク問題をシミュレート", user.Nickname, networkProfile.Name)

	// 接続を切断
	if user.Conn != nil {
		user.Conn.Close()
		user.Conn = nil
	}

	alt.recordNetworkEvent(user, "NetworkDisconnection", fmt.Sprintf("%s triggered disconnection", networkProfile.Name), 0)
	user.ReconnectCount++

	// 再接続遅延
	time.Sleep(networkProfile.ReconnectDelay)

	// 再接続を試行
	if alt.connectAdvancedWebSocket(user) {
		alt.recordNetworkEvent(user, "NetworkReconnection", fmt.Sprintf("Reconnected after %v", networkProfile.ReconnectDelay), 0)
	}
}

func (alt *AdvancedLoadTester) recordNetworkEvent(user *UserSession, eventType, description string, latency time.Duration) {
	event := NetworkEvent{
		Timestamp:   time.Now(),
		UserID:      user.ID,
		EventType:   eventType,
		Description: description,
		Latency:     latency,
	}

	alt.mu.Lock()
	alt.NetworkEventLog = append(alt.NetworkEventLog, event)
	alt.mu.Unlock()

	user.NetworkEvents = append(user.NetworkEvents, event)
}

func (alt *AdvancedLoadTester) recordError(user *UserSession, errorMsg string) {
	log.Printf("❌ %s: %s", user.Nickname, errorMsg)

	alt.mu.Lock()
	alt.TotalErrors++
	user.ErrorCount++
	alt.mu.Unlock()
}

func (alt *AdvancedLoadTester) closeConnection(user *UserSession) {
	if user.Conn != nil {
		user.Conn.Close()
	}
}

func (alt *AdvancedLoadTester) printNetworkProfileSummary() {
	log.Println("📊 ネットワークプロファイル分布:")

	profileCounts := make(map[NetworkIssueType]int)
	for _, user := range alt.UserProfiles {
		profileCounts[user.NetworkType]++
	}

	for networkType, count := range profileCounts {
		profile := alt.NetworkProfiles[networkType]
		log.Printf("   %s: %d人 (%.1f%%)", profile.Name, count, float64(count)/float64(alt.Users)*100)
	}
}

func (alt *AdvancedLoadTester) printAdvancedStats() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-alt.stopChan:
			return
		case <-ticker.C:
			alt.mu.RLock()
			elapsed := time.Since(alt.startTime)
			avgResponseTime := alt.calculateAverageResponseTime()
			recentEvents := len(alt.NetworkEventLog)

			log.Printf("📊 [%v経過] 接続中: %d, 総メッセージ: %d, エラー: %d, 平均レスポンス: %v, ネットワークイベント: %d",
				elapsed.Round(time.Second),
				alt.ConnectedUsers,
				alt.TotalMessages,
				alt.TotalErrors,
				avgResponseTime.Round(time.Millisecond),
				recentEvents)
			alt.mu.RUnlock()
		}
	}
}

func (alt *AdvancedLoadTester) monitorNetworkEvents() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-alt.stopChan:
			return
		case <-ticker.C:
			alt.analyzeNetworkEvents()
		}
	}
}

func (alt *AdvancedLoadTester) analyzeNetworkEvents() {
	alt.mu.RLock()
	events := alt.NetworkEventLog[:]
	alt.mu.RUnlock()

	if len(events) == 0 {
		return
	}

	// 最近5分間のイベントを分析
	cutoff := time.Now().Add(-5 * time.Minute)
	recentEvents := make([]NetworkEvent, 0)
	eventCounts := make(map[string]int)

	for _, event := range events {
		if event.Timestamp.After(cutoff) {
			recentEvents = append(recentEvents, event)
			eventCounts[event.EventType]++
		}
	}

	if len(recentEvents) > 0 {
		log.Printf("📈 過去5分間のネットワークイベント:")
		for eventType, count := range eventCounts {
			log.Printf("   %s: %d回", eventType, count)
		}
	}
}

func (alt *AdvancedLoadTester) calculateAverageResponseTime() time.Duration {
	if len(alt.ResponseTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, rt := range alt.ResponseTimes {
		total += rt
	}
	return total / time.Duration(len(alt.ResponseTimes))
}

func (alt *AdvancedLoadTester) printAdvancedFinalReport() {
	alt.mu.RLock()
	defer alt.mu.RUnlock()

	totalDuration := time.Since(alt.startTime)
	avgResponseTime := alt.calculateAverageResponseTime()

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🎯 Quiz100 高度な負荷テスト結果レポート")
	fmt.Println(strings.Repeat("=", 80))

	// 基本統計
	fmt.Printf("⏱️  テスト実行時間: %v\n", totalDuration.Round(time.Second))
	fmt.Printf("👥 対象ユーザー数: %d人\n", alt.Users)
	fmt.Printf("📊 総メッセージ数: %d\n", alt.TotalMessages)
	fmt.Printf("❌ エラー総数: %d\n", alt.TotalErrors)

	if alt.TotalMessages+alt.TotalErrors > 0 {
		errorRate := float64(alt.TotalErrors) / float64(alt.TotalMessages+alt.TotalErrors) * 100
		fmt.Printf("📈 エラー率: %.2f%%\n", errorRate)
	}

	fmt.Printf("⚡ 平均レスポンス時間: %v\n", avgResponseTime.Round(time.Millisecond))

	// ネットワークプロファイル別統計
	fmt.Println("\n🌐 ネットワークプロファイル別結果:")
	profileStats := make(map[NetworkIssueType]struct {
		Users      int
		Errors     int64
		Messages   int64
		Reconnects int
	})

	for _, user := range alt.UserProfiles {
		stats := profileStats[user.NetworkType]
		stats.Users++
		// この統計は簡略化されています
		profileStats[user.NetworkType] = stats
	}

	for networkType, stats := range profileStats {
		profile := alt.NetworkProfiles[networkType]
		fmt.Printf("   %s: %d人\n", profile.Name, stats.Users)
	}

	// ネットワークイベント統計
	fmt.Println("\n📈 ネットワークイベント統計:")
	eventCounts := make(map[string]int)
	for _, event := range alt.NetworkEventLog {
		eventCounts[event.EventType]++
	}

	for eventType, count := range eventCounts {
		fmt.Printf("   %s: %d回\n", eventType, count)
	}

	// 推奨事項
	fmt.Println("\n🎯 分析結果と推奨事項:")

	if alt.TotalErrors == 0 {
		fmt.Println("   ✅ エラーなし - システムは安定しています")
	} else {
		errorRate := float64(alt.TotalErrors) / float64(alt.TotalMessages+alt.TotalErrors) * 100
		if errorRate > 10 {
			fmt.Printf("   ⚠️  エラー率が高めです (%.2f%%) - システム負荷の調整が必要\n", errorRate)
		} else if errorRate > 5 {
			fmt.Printf("   ⚠️  エラー率が少し高めです (%.2f%%) - 監視を継続\n", errorRate)
		}
	}

	if avgResponseTime > 2*time.Second {
		fmt.Printf("   ⚠️  レスポンス時間が長めです (%v) - パフォーマンス最適化を検討\n", avgResponseTime.Round(time.Millisecond))
	} else if avgResponseTime < 500*time.Millisecond {
		fmt.Printf("   ✅ 優秀なレスポンス時間です (%v)\n", avgResponseTime.Round(time.Millisecond))
	}

	disconnectionEvents := eventCounts["NetworkDisconnection"]
	reconnectionEvents := eventCounts["NetworkReconnection"]

	if disconnectionEvents > 0 {
		reconnectionRate := float64(reconnectionEvents) / float64(disconnectionEvents) * 100
		fmt.Printf("   📶 再接続率: %.1f%% (%d/%d)\n", reconnectionRate, reconnectionEvents, disconnectionEvents)

		if reconnectionRate > 90 {
			fmt.Println("   ✅ 高い再接続率 - 回復力が優秀です")
		} else if reconnectionRate < 70 {
			fmt.Println("   ⚠️  再接続率が低い - ネットワーク処理の改善を検討")
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}

func main() {
	rand.Seed(time.Now().UnixNano())

	tester := NewAdvancedLoadTester()

	// コマンドライン引数の処理
	if len(os.Args) > 1 {
		if users, err := strconv.Atoi(os.Args[1]); err == nil && users > 0 {
			tester.Users = users
		}
	}

	tester.Start()
}
