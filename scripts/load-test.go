package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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

type LoadTester struct {
	ServerURL       string
	WebSocketURL    string
	Users           int
	TestDuration    time.Duration
	UnstablePercent int // 通信不安定なユーザーの割合(%)

	// 統計情報
	mu                sync.RWMutex
	ConnectedUsers    int
	DisconnectedUsers int
	TotalMessages     int64
	TotalErrors       int64
	ResponseTimes     []time.Duration

	startTime time.Time
	stopChan  chan bool
}

type UserSession struct {
	ID         int
	Nickname   string
	SessionID  string
	Conn       *websocket.Conn
	IsUnstable bool

	JoinTime      time.Time
	MessagesSent  int64
	ErrorCount    int64
	ResponseTimes []time.Duration
}

type TestMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

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

func NewLoadTester() *LoadTester {
	return &LoadTester{
		ServerURL:       ServerURL,
		WebSocketURL:    WebSocketURL,
		Users:           DefaultUsers,
		TestDuration:    DefaultTestDuration,
		UnstablePercent: 20, // デフォルトで20%のユーザーを不安定状態に
		stopChan:        make(chan bool),
		ResponseTimes:   make([]time.Duration, 0),
	}
}

func (lt *LoadTester) Start() {
	log.Printf("🚀 負荷テスト開始: %d人のユーザーで %v間実行", lt.Users, lt.TestDuration)
	log.Printf("🌐 サーバーURL: %s", lt.ServerURL)
	log.Printf("⚡ 不安定ユーザー割合: %d%%", lt.UnstablePercent)

	lt.startTime = time.Now()

	var wg sync.WaitGroup

	// テスト終了タイマー
	go func() {
		time.Sleep(lt.TestDuration)
		close(lt.stopChan)
	}()

	// ユーザーセッションを並行実行
	for i := 0; i < lt.Users; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			lt.simulateUser(userID)
		}(i + 1)

		// 接続を段階的に増やす（サーバーへの負荷を分散）
		if i%10 == 9 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 統計情報を定期的に出力
	go lt.printStats()

	wg.Wait()

	log.Printf("✅ 負荷テスト完了")
	lt.printFinalReport()
}

func (lt *LoadTester) simulateUser(userID int) {
	nickname := fmt.Sprintf("TestUser%03d", userID)
	isUnstable := rand.Intn(100) < lt.UnstablePercent

	user := &UserSession{
		ID:            userID,
		Nickname:      nickname,
		IsUnstable:    isUnstable,
		JoinTime:      time.Now(),
		ResponseTimes: make([]time.Duration, 0),
	}

	log.Printf("👤 ユーザー %s 開始 (不安定: %v)", nickname, isUnstable)

	// 1. ユーザー登録
	if !lt.joinUser(user) {
		return
	}

	// 2. WebSocket接続
	if !lt.connectWebSocket(user) {
		return
	}
	defer lt.closeConnection(user)

	lt.mu.Lock()
	lt.ConnectedUsers++
	lt.mu.Unlock()

	// 3. テスト期間中、様々なアクションを実行
	lt.runUserActions(user)

	lt.mu.Lock()
	lt.ConnectedUsers--
	lt.DisconnectedUsers++
	lt.mu.Unlock()
}

func (lt *LoadTester) joinUser(user *UserSession) bool {
	joinReq := JoinRequest{Nickname: user.Nickname}
	reqBody, _ := json.Marshal(joinReq)

	startTime := time.Now()
	resp, err := http.Post(lt.ServerURL+"/api/join", "application/json", bytes.NewBuffer(reqBody))
	responseTime := time.Since(startTime)

	if err != nil {
		lt.recordError(user, fmt.Sprintf("join request failed: %v", err))
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		lt.recordError(user, fmt.Sprintf("join failed with status: %d", resp.StatusCode))
		return false
	}

	var joinResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&joinResp); err != nil {
		lt.recordError(user, fmt.Sprintf("join response decode failed: %v", err))
		return false
	}

	user.SessionID = joinResp["session_id"].(string)
	user.ResponseTimes = append(user.ResponseTimes, responseTime)

	lt.mu.Lock()
	lt.ResponseTimes = append(lt.ResponseTimes, responseTime)
	lt.mu.Unlock()

	return true
}

func (lt *LoadTester) connectWebSocket(user *UserSession) bool {
	wsURL := fmt.Sprintf("%s/ws/participant?session_id=%s", lt.WebSocketURL, user.SessionID)

	startTime := time.Now()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	responseTime := time.Since(startTime)

	if err != nil {
		lt.recordError(user, fmt.Sprintf("websocket connection failed: %v", err))
		return false
	}

	user.Conn = conn
	user.ResponseTimes = append(user.ResponseTimes, responseTime)

	lt.mu.Lock()
	lt.ResponseTimes = append(lt.ResponseTimes, responseTime)
	lt.mu.Unlock()

	return true
}

func (lt *LoadTester) runUserActions(user *UserSession) {
	// WebSocketメッセージを受信するgoroutine
	go lt.handleWebSocketMessages(user)

	ticker := time.NewTicker(time.Duration(rand.Intn(3000)+1000) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-lt.stopChan:
			return
		case <-ticker.C:
			// 不安定なユーザーは時々接続を切断・再接続
			if user.IsUnstable && rand.Intn(100) < 15 { // 15%の確率で不安定な動作
				lt.simulateUnstableConnection(user)
				continue
			}

			// ランダムなアクションを実行
			action := rand.Intn(3)
			switch action {
			case 0:
				lt.sendAnswer(user)
			case 1:
				lt.sendEmoji(user)
			case 2:
				// 何もしない（アイドル状態）
			}
		}
	}
}

func (lt *LoadTester) handleWebSocketMessages(user *UserSession) {
	for {
		select {
		case <-lt.stopChan:
			return
		default:
			if user.Conn == nil {
				return
			}

			user.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			_, _, err := user.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					lt.recordError(user, fmt.Sprintf("websocket read error: %v", err))
				}
				return
			}

			// メッセージを処理（ここでは単純にカウント）
			lt.mu.Lock()
			lt.TotalMessages++
			user.MessagesSent++
			lt.mu.Unlock()
		}
	}
}

func (lt *LoadTester) sendAnswer(user *UserSession) {
	if user.Conn == nil {
		return
	}

	answerReq := AnswerRequest{
		QuestionNumber: rand.Intn(5) + 1, // 1-5の問題番号
		AnswerIndex:    rand.Intn(4) + 1, // 1-4の選択肢
	}

	reqBody, _ := json.Marshal(answerReq)
	req, _ := http.NewRequest("POST", lt.ServerURL+"/api/answer", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", user.SessionID)

	startTime := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		lt.recordError(user, fmt.Sprintf("answer request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	user.ResponseTimes = append(user.ResponseTimes, responseTime)
	lt.mu.Lock()
	lt.ResponseTimes = append(lt.ResponseTimes, responseTime)
	lt.mu.Unlock()
}

func (lt *LoadTester) sendEmoji(user *UserSession) {
	emojis := []string{"😀", "😂", "🎉", "👍", "❤️", "🔥", "💪", "🎊"}
	emoji := emojis[rand.Intn(len(emojis))]

	emojiReq := EmojiRequest{Emoji: emoji}
	reqBody, _ := json.Marshal(emojiReq)
	req, _ := http.NewRequest("POST", lt.ServerURL+"/api/emoji", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", user.SessionID)

	startTime := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		lt.recordError(user, fmt.Sprintf("emoji request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	user.ResponseTimes = append(user.ResponseTimes, responseTime)
	lt.mu.Lock()
	lt.ResponseTimes = append(lt.ResponseTimes, responseTime)
	lt.mu.Unlock()
}

func (lt *LoadTester) simulateUnstableConnection(user *UserSession) {
	log.Printf("🔄 %s: 不安定接続をシミュレート", user.Nickname)

	// 接続を一時的に切断
	if user.Conn != nil {
		user.Conn.Close()
		user.Conn = nil
	}

	// 2-10秒間待機（ネットワーク障害をシミュレート）
	time.Sleep(time.Duration(rand.Intn(8000)+2000) * time.Millisecond)

	// 再接続を試行
	lt.connectWebSocket(user)
}

func (lt *LoadTester) recordError(user *UserSession, errorMsg string) {
	log.Printf("❌ %s: %s", user.Nickname, errorMsg)

	lt.mu.Lock()
	lt.TotalErrors++
	user.ErrorCount++
	lt.mu.Unlock()
}

func (lt *LoadTester) closeConnection(user *UserSession) {
	if user.Conn != nil {
		user.Conn.Close()
	}
}

func (lt *LoadTester) printStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-lt.stopChan:
			return
		case <-ticker.C:
			lt.mu.RLock()
			elapsed := time.Since(lt.startTime)
			avgResponseTime := lt.calculateAverageResponseTime()

			log.Printf("📊 [%v経過] 接続中: %d, 切断済み: %d, 総メッセージ: %d, エラー: %d, 平均レスポンス: %v",
				elapsed.Round(time.Second),
				lt.ConnectedUsers,
				lt.DisconnectedUsers,
				lt.TotalMessages,
				lt.TotalErrors,
				avgResponseTime.Round(time.Millisecond))
			lt.mu.RUnlock()
		}
	}
}

func (lt *LoadTester) calculateAverageResponseTime() time.Duration {
	if len(lt.ResponseTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, rt := range lt.ResponseTimes {
		total += rt
	}
	return total / time.Duration(len(lt.ResponseTimes))
}

func (lt *LoadTester) printFinalReport() {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	totalDuration := time.Since(lt.startTime)
	avgResponseTime := lt.calculateAverageResponseTime()

	var minResponseTime, maxResponseTime time.Duration
	if len(lt.ResponseTimes) > 0 {
		minResponseTime = lt.ResponseTimes[0]
		maxResponseTime = lt.ResponseTimes[0]

		for _, rt := range lt.ResponseTimes {
			if rt < minResponseTime {
				minResponseTime = rt
			}
			if rt > maxResponseTime {
				maxResponseTime = rt
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🎯 Quiz100 負荷テスト結果レポート")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("⏱️  テスト実行時間: %v\n", totalDuration.Round(time.Second))
	fmt.Printf("👥 対象ユーザー数: %d人\n", lt.Users)
	fmt.Printf("🌐 不安定ユーザー割合: %d%%\n", lt.UnstablePercent)
	fmt.Println("\n📊 パフォーマンス統計:")
	fmt.Printf("   • 接続成功ユーザー: %d人\n", lt.ConnectedUsers+lt.DisconnectedUsers)
	fmt.Printf("   • 総メッセージ数: %d\n", lt.TotalMessages)
	fmt.Printf("   • エラー総数: %d\n", lt.TotalErrors)
	fmt.Printf("   • エラー率: %.2f%%\n", float64(lt.TotalErrors)/float64(lt.TotalMessages+lt.TotalErrors)*100)
	fmt.Println("\n⚡ レスポンス時間:")
	fmt.Printf("   • 平均: %v\n", avgResponseTime.Round(time.Millisecond))
	fmt.Printf("   • 最小: %v\n", minResponseTime.Round(time.Millisecond))
	fmt.Printf("   • 最大: %v\n", maxResponseTime.Round(time.Millisecond))

	if len(lt.ResponseTimes) > 0 {
		// 95パーセンタイル計算
		sortedTimes := make([]time.Duration, len(lt.ResponseTimes))
		copy(sortedTimes, lt.ResponseTimes)
		// 簡単なソート
		for i := 0; i < len(sortedTimes)-1; i++ {
			for j := i + 1; j < len(sortedTimes); j++ {
				if sortedTimes[i] > sortedTimes[j] {
					sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
				}
			}
		}
		p95Index := int(float64(len(sortedTimes)) * 0.95)
		if p95Index < len(sortedTimes) {
			fmt.Printf("   • 95パーセンタイル: %v\n", sortedTimes[p95Index].Round(time.Millisecond))
		}
	}

	fmt.Println("\n🎯 推奨事項:")
	if lt.TotalErrors > 0 {
		errorRate := float64(lt.TotalErrors) / float64(lt.TotalMessages+lt.TotalErrors) * 100
		if errorRate > 5 {
			fmt.Printf("   ⚠️  エラー率が高めです (%.2f%%) - サーバーリソースの確認を推奨\n", errorRate)
		}
	}

	if avgResponseTime > 1*time.Second {
		fmt.Printf("   ⚠️  平均レスポンス時間が長めです (%v) - パフォーマンス最適化を検討\n", avgResponseTime.Round(time.Millisecond))
	}

	if lt.TotalErrors == 0 && avgResponseTime < 500*time.Millisecond {
		fmt.Println("   ✅ 優秀な結果です！本番環境での使用に適しています")
	}

	fmt.Println(strings.Repeat("=", 60))
}

func main() {
	rand.Seed(time.Now().UnixNano())

	tester := NewLoadTester()

	// コマンドライン引数の簡単な処理
	if len(os.Args) > 1 {
		if users, err := strconv.Atoi(os.Args[1]); err == nil && users > 0 {
			tester.Users = users
		}
	}

	tester.Start()
}
