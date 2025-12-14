package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	TotalUsers     = 80
	ServerURL      = "http://localhost:8080"
	WebSocketURL   = "ws://localhost:8080"
	TotalQuestions = 14 // クイズの総問題数
)

// AnswerPattern は各ユーザーの回答パターンを定義
type AnswerPattern string

const (
	AllCorrect      AnswerPattern = "all_correct"       // 全問正解
	AllWrong        AnswerPattern = "all_wrong"         // 全問不正解
	OnlyQuestion1   AnswerPattern = "only_q1"           // 1問目だけ正解
	OnlyQuestion2   AnswerPattern = "only_q2"           // 2問目だけ正解
	OnlyQuestion3   AnswerPattern = "only_q3"           // 3問目だけ正解
	OnlyQuestion4   AnswerPattern = "only_q4"           // 4問目だけ正解
	OnlyQuestion5   AnswerPattern = "only_q5"           // 5問目だけ正解
	AnswerOnlyFirst AnswerPattern = "answer_only_first" // 1問目だけ回答（以降回答しない）
	SkipFirst       AnswerPattern = "skip_first"        // 1問目スキップ、2問目以降正解
	ChangeAnswer    AnswerPattern = "change_answer"     // 回答を変更する（最初は不正解を選び、1秒後に正解に変更）
	Random          AnswerPattern = "random"            // ランダムに回答
)

// UserProfile はユーザーのプロファイル
type UserProfile struct {
	ID              int
	Nickname        string
	Pattern         AnswerPattern
	ExpectedScore   int // 期待される獲得点数
	ActualScore     int // 実際の獲得点数
	SessionID       string
	Conn            *websocket.Conn
	CurrentQuestion int
	Answers         map[int]int // questionNumber -> answerIndex
}

// Question は問題情報
type Question struct {
	Number  int
	Correct int // 正解の選択肢インデックス（1-based）
	Point   int // 配点
}

// SequenceLoadTester はシーケンス負荷テスター
type SequenceLoadTester struct {
	Users     []*UserProfile
	Questions []Question
	mu        sync.RWMutex

	currentQuestionNumber int
	allUsersConnected     bool
	testCompleted         bool
}

func NewSequenceLoadTester() *SequenceLoadTester {
	return &SequenceLoadTester{
		Users:     make([]*UserProfile, 0, TotalUsers),
		Questions: generateQuestions(),
	}
}

// generateQuestions は問題データを生成（quiz.tomlから読み取る代わりにハードコード）
func generateQuestions() []Question {
	questions := []Question{
		{Number: 1, Correct: 1, Point: 0},
		{Number: 2, Correct: 2, Point: 1},
		{Number: 3, Correct: 3, Point: 1},
		{Number: 4, Correct: 3, Point: 1},
		{Number: 5, Correct: 1, Point: 1},
		{Number: 6, Correct: 1, Point: 1},
		{Number: 7, Correct: 1, Point: 1},
		{Number: 8, Correct: 1, Point: 1},
		{Number: 9, Correct: 1, Point: 1},
		{Number: 10, Correct: 1, Point: 1},
		{Number: 11, Correct: 1, Point: 1},
		{Number: 12, Correct: 1, Point: 1},
		{Number: 13, Correct: 1, Point: 1},
		{Number: 14, Correct: 1, Point: 2},
	}
	return questions
}

// generateUserProfiles はユーザープロファイルを生成
func (slt *SequenceLoadTester) generateUserProfiles() {
	patterns := []AnswerPattern{
		AllCorrect, AllWrong,
		OnlyQuestion1, OnlyQuestion2, OnlyQuestion3, OnlyQuestion4, OnlyQuestion5,
		AnswerOnlyFirst, SkipFirst, ChangeAnswer, Random,
	}

	// 各パターンを均等に割り当て
	for i := 0; i < TotalUsers; i++ {
		pattern := patterns[i%len(patterns)]
		user := &UserProfile{
			ID:       i + 1,
			Nickname: fmt.Sprintf("User%03d", i+1),
			Pattern:  pattern,
			Answers:  make(map[int]int),
		}

		// 期待スコアを計算
		user.ExpectedScore = slt.calculateExpectedScore(user.Pattern)
		slt.Users = append(slt.Users, user)
	}

	log.Printf("✅ %d人のユーザープロファイルを生成しました", len(slt.Users))
	slt.printPatternDistribution()
}

// calculateExpectedScore はパターンに基づいて期待スコアを計算
func (slt *SequenceLoadTester) calculateExpectedScore(pattern AnswerPattern) int {
	score := 0
	switch pattern {
	case AllCorrect, ChangeAnswer:
		// 全問正解
		for _, q := range slt.Questions {
			score += q.Point
		}
	case AllWrong, Random:
		// 全問不正解またはランダム（期待値0）
		score = 0
	case OnlyQuestion1:
		score = slt.Questions[0].Point
	case OnlyQuestion2:
		score = slt.Questions[1].Point
	case OnlyQuestion3:
		score = slt.Questions[2].Point
	case OnlyQuestion4:
		score = slt.Questions[3].Point
	case OnlyQuestion5:
		score = slt.Questions[4].Point
	case AnswerOnlyFirst:
		score = slt.Questions[0].Point
	case SkipFirst:
		// 1問目以外全正解
		for i := 1; i < len(slt.Questions); i++ {
			score += slt.Questions[i].Point
		}
	}
	return score
}

// printPatternDistribution はパターン分布を表示
func (slt *SequenceLoadTester) printPatternDistribution() {
	patternCounts := make(map[AnswerPattern]int)
	for _, user := range slt.Users {
		patternCounts[user.Pattern]++
	}

	log.Println("📊 回答パターン分布:")
	for pattern, count := range patternCounts {
		log.Printf("   %s: %d人", pattern, count)
	}
}

// Start はテストを開始
func (slt *SequenceLoadTester) Start() {
	log.Println("🚀 シーケンス負荷テスト開始")
	log.Printf("👥 ユーザー数: %d人", TotalUsers)
	log.Printf("📝 問題数: %d問", len(slt.Questions))

	// ユーザープロファイルを生成
	slt.generateUserProfiles()

	// Phase 1: ユーザー接続（30秒間で段階的に接続）
	log.Println("\n=== Phase 1: ユーザー接続 ===")
	slt.connectUsers()

	waitForEnter("ユーザー接続が完了しました。イベントを開始してください")

	// Phase 2: クイズシーケンス
	log.Println("\n=== Phase 2: クイズシーケンス ===")
	for qNum := 1; qNum <= len(slt.Questions); qNum++ {
		slt.currentQuestionNumber = qNum
		log.Printf("\n--- 問題 %d/%d ---", qNum, len(slt.Questions))

		waitForEnter(fmt.Sprintf("問題%dを出題してください", qNum))

		// WebSocketメッセージを監視して、question_startを検知
		log.Println("⏳ 問題出題を待機中...")
		time.Sleep(2 * time.Second) // 問題が配信されるまで待機

		// 回答送信（10秒間で分散）
		slt.submitAnswers(qNum)

		waitForEnter(fmt.Sprintf("問題%dの回答受付を終了してください", qNum))

		// リアクション送信
		slt.sendReactions()

		if qNum < len(slt.Questions) {
			waitForEnter("次の問題に進む準備ができたらエンターキーを押してください")
		}
	}

	// Phase 3: 結果確認
	log.Println("\n=== Phase 3: 結果確認 ===")
	waitForEnter("最終結果を発表してください")

	time.Sleep(3 * time.Second) // 結果メッセージが配信されるまで待機

	// 結果を検証
	slt.verifyResults()

	// Phase 4: クリーンアップ
	log.Println("\n=== Phase 4: クリーンアップ ===")
	slt.cleanup()

	log.Println("\n✅ シーケンス負荷テスト完了")
}

// connectUsers はユーザーを段階的に接続
func (slt *SequenceLoadTester) connectUsers() {
	var wg sync.WaitGroup
	connectionInterval := 30 * time.Second / time.Duration(TotalUsers)

	for _, user := range slt.Users {
		wg.Add(1)
		go func(u *UserProfile) {
			defer wg.Done()
			slt.connectUser(u)
		}(user)

		time.Sleep(connectionInterval)
	}

	wg.Wait()
	slt.allUsersConnected = true
	log.Printf("✅ 全ユーザー（%d人）の接続が完了しました", len(slt.Users))
}

// connectUser は個別のユーザーを接続
func (slt *SequenceLoadTester) connectUser(user *UserProfile) {
	// 1. 参加登録
	if !slt.joinUser(user) {
		log.Printf("❌ %s: 参加登録失敗", user.Nickname)
		return
	}

	// 2. WebSocket接続
	if !slt.connectWebSocket(user) {
		log.Printf("❌ %s: WebSocket接続失敗", user.Nickname)
		return
	}

	// 3. WebSocketメッセージを監視
	go slt.handleWebSocketMessages(user)

	log.Printf("✅ %s: 接続完了 (パターン: %s, 期待スコア: %d点)", user.Nickname, user.Pattern, user.ExpectedScore)
}

// joinUser はユーザーを参加登録
func (slt *SequenceLoadTester) joinUser(user *UserProfile) bool {
	joinReq := map[string]string{"nickname": user.Nickname}
	reqBody, _ := json.Marshal(joinReq)

	resp, err := http.Post(ServerURL+"/api/join", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("❌ %s: 参加登録エラー: %v", user.Nickname, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ %s: 参加登録失敗 (status: %d)", user.Nickname, resp.StatusCode)
		return false
	}

	var joinResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&joinResp); err != nil {
		log.Printf("❌ %s: レスポンス解析エラー: %v", user.Nickname, err)
		return false
	}

	user.SessionID = joinResp["session_id"].(string)
	return true
}

// connectWebSocket はWebSocketに接続
func (slt *SequenceLoadTester) connectWebSocket(user *UserProfile) bool {
	wsURL := fmt.Sprintf("%s/ws/participant?session_id=%s", WebSocketURL, user.SessionID)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("❌ %s: WebSocket接続エラー: %v", user.Nickname, err)
		return false
	}

	user.Conn = conn
	return true
}

// handleWebSocketMessages はWebSocketメッセージを処理
func (slt *SequenceLoadTester) handleWebSocketMessages(user *UserProfile) {
	defer func() {
		if user.Conn != nil {
			user.Conn.Close()
		}
	}()

	for {
		_, message, err := user.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("⚠️  %s: WebSocket切断: %v", user.Nickname, err)
			}
			return
		}

		// メッセージを解析
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// 最終結果メッセージを検知
		if msgType, ok := msg["type"].(string); ok && msgType == "final_results" {
			slt.handleFinalResults(user, msg)
		}
	}
}

// handleFinalResults は最終結果を処理
func (slt *SequenceLoadTester) handleFinalResults(user *UserProfile, msg map[string]interface{}) {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return
	}

	// チーム戦の場合
	if teamMode, ok := data["team_mode"].(bool); ok && teamMode {
		// チーム戦の処理（今回は個人戦を想定）
		return
	}

	// 個人戦の結果
	if results, ok := data["results"].([]interface{}); ok {
		for _, r := range results {
			result := r.(map[string]interface{})
			if nickname, ok := result["nickname"].(string); ok && nickname == user.Nickname {
				if score, ok := result["score"].(float64); ok {
					user.ActualScore = int(score)
					log.Printf("📊 %s: 実際のスコア = %d点", user.Nickname, user.ActualScore)
				}
			}
		}
	}
}

// submitAnswers は回答を送信
func (slt *SequenceLoadTester) submitAnswers(questionNumber int) {
	log.Printf("📝 問題%d: 回答送信開始（10秒間で分散）", questionNumber)

	var wg sync.WaitGroup
	submitInterval := 10 * time.Second / time.Duration(len(slt.Users))

	for _, user := range slt.Users {
		wg.Add(1)
		go func(u *UserProfile) {
			defer wg.Done()
			slt.submitUserAnswer(u, questionNumber)
		}(user)

		// 回答を分散させる
		time.Sleep(submitInterval)
	}

	wg.Wait()
	log.Printf("✅ 問題%d: 全ユーザーの回答送信完了", questionNumber)
}

// submitUserAnswer はユーザーの回答を送信
func (slt *SequenceLoadTester) submitUserAnswer(user *UserProfile, questionNumber int) {
	// パターンに基づいて回答を決定
	answerIndex := slt.determineAnswer(user, questionNumber)

	if answerIndex == 0 {
		// 回答しない
		return
	}

	// 回答変更パターンの場合、最初に間違った回答を送信
	if user.Pattern == ChangeAnswer {
		wrongAnswer := answerIndex
		if wrongAnswer == 1 {
			wrongAnswer = 2
		} else {
			wrongAnswer = 1
		}

		slt.sendAnswer(user, questionNumber, wrongAnswer)
		time.Sleep(1 * time.Second) // 1秒待機
	}

	// 正しい回答（またはパターンに応じた回答）を送信
	slt.sendAnswer(user, questionNumber, answerIndex)
	user.Answers[questionNumber] = answerIndex
}

// determineAnswer はパターンに基づいて回答を決定
func (slt *SequenceLoadTester) determineAnswer(user *UserProfile, questionNumber int) int {
	question := slt.Questions[questionNumber-1]

	switch user.Pattern {
	case AllCorrect, ChangeAnswer:
		return question.Correct
	case AllWrong:
		// 不正解を返す
		wrongAnswer := question.Correct + 1
		if wrongAnswer > 4 {
			wrongAnswer = 1
		}
		return wrongAnswer
	case OnlyQuestion1:
		if questionNumber == 1 {
			return question.Correct
		}
		return 0 // 回答しない
	case OnlyQuestion2:
		if questionNumber == 2 {
			return question.Correct
		}
		return 0
	case OnlyQuestion3:
		if questionNumber == 3 {
			return question.Correct
		}
		return 0
	case OnlyQuestion4:
		if questionNumber == 4 {
			return question.Correct
		}
		return 0
	case OnlyQuestion5:
		if questionNumber == 5 {
			return question.Correct
		}
		return 0
	case AnswerOnlyFirst:
		if questionNumber == 1 {
			return question.Correct
		}
		return 0 // 以降は回答しない
	case SkipFirst:
		if questionNumber == 1 {
			return 0 // 1問目はスキップ
		}
		return question.Correct
	case Random:
		return rand.Intn(4) + 1 // 1-4のランダム
	default:
		return question.Correct
	}
}

// sendAnswer は回答を送信
func (slt *SequenceLoadTester) sendAnswer(user *UserProfile, questionNumber, answerIndex int) {
	answerReq := map[string]int{
		"question_number": questionNumber,
		"answer_index":    answerIndex,
	}
	reqBody, _ := json.Marshal(answerReq)

	req, _ := http.NewRequest("POST", ServerURL+"/api/answer", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", user.SessionID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("⚠️  %s: 回答送信エラー (Q%d, A%d): %v", user.Nickname, questionNumber, answerIndex, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️  %s: 回答送信失敗 (Q%d, A%d, status: %d)", user.Nickname, questionNumber, answerIndex, resp.StatusCode)
	}
}

// sendReactions はリアクションを送信
func (slt *SequenceLoadTester) sendReactions() {
	log.Println("😄 リアクション送信中...")

	emojis := []string{"👍", "❤️", "😂", "😮", "😢", "🔥", "✨", "💪"}

	// ランダムに30-50人がリアクションを送信
	reactionCount := rand.Intn(21) + 30

	for i := 0; i < reactionCount && i < len(slt.Users); i++ {
		user := slt.Users[rand.Intn(len(slt.Users))]
		emoji := emojis[rand.Intn(len(emojis))]

		emojiReq := map[string]string{"emoji": emoji}
		reqBody, _ := json.Marshal(emojiReq)

		req, _ := http.NewRequest("POST", ServerURL+"/api/emoji", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Session-ID", user.SessionID)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}

		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	}

	log.Printf("✅ リアクション送信完了（約%d件）", reactionCount)
}

// verifyResults は結果を検証
func (slt *SequenceLoadTester) verifyResults() {
	log.Println("\n📊 結果検証中...")
	time.Sleep(2 * time.Second) // 結果が全ユーザーに配信されるまで待機

	var matchCount, mismatchCount, noDataCount int
	var mismatches []string

	for _, user := range slt.Users {
		if user.ActualScore == 0 && user.ExpectedScore == 0 {
			// スコアが0の場合は区別できないので一致とみなす
			matchCount++
		} else if user.ActualScore == user.ExpectedScore {
			matchCount++
		} else if user.ActualScore == 0 && user.ExpectedScore != 0 {
			noDataCount++
			mismatches = append(mismatches, fmt.Sprintf("  %s: データなし (期待: %d点)", user.Nickname, user.ExpectedScore))
		} else {
			mismatchCount++
			mismatches = append(mismatches, fmt.Sprintf("  %s: 期待 %d点 → 実際 %d点", user.Nickname, user.ExpectedScore, user.ActualScore))
		}
	}

	fmt.Println("\n" + "===========================================")
	fmt.Println("🎯 結果検証レポート")
	fmt.Println("===========================================")
	fmt.Printf("✅ 一致: %d人 (%.1f%%)\n", matchCount, float64(matchCount)/float64(len(slt.Users))*100)
	fmt.Printf("❌ 不一致: %d人 (%.1f%%)\n", mismatchCount, float64(mismatchCount)/float64(len(slt.Users))*100)
	fmt.Printf("⚠️  データなし: %d人 (%.1f%%)\n", noDataCount, float64(noDataCount)/float64(len(slt.Users))*100)

	if len(mismatches) > 0 {
		fmt.Println("\n不一致・データなしの詳細:")
		for _, msg := range mismatches {
			fmt.Println(msg)
		}
	}

	fmt.Println("===========================================")

	if mismatchCount == 0 && noDataCount == 0 {
		log.Println("\n🎉 全ユーザーのスコアが期待値と一致しました！")
	} else {
		log.Printf("\n⚠️  %d人のユーザーでスコアの不一致またはデータ欠落が検出されました", mismatchCount+noDataCount)
	}
}

// cleanup はリソースをクリーンアップ
func (slt *SequenceLoadTester) cleanup() {
	log.Println("🧹 クリーンアップ中...")

	for _, user := range slt.Users {
		if user.Conn != nil {
			user.Conn.Close()
		}
	}

	log.Println("✅ クリーンアップ完了")
}

// waitForEnter はエンターキー入力を待機
func waitForEnter(message string) {
	fmt.Printf("\n⏸️  %s [Enterキーを押してください]\n", message)
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func main() {
	rand.Seed(time.Now().UnixNano())

	tester := NewSequenceLoadTester()
	tester.Start()
}
