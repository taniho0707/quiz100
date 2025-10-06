package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
	// API用拡張データ
	APIData *APIExtensions `json:"-"`
}

// API拡張データ構造
type APIExtensions struct {
	Users     []User     `json:"users,omitempty"`
	Teams     []Team     `json:"teams,omitempty"`
	Questions []Question `json:"questions,omitempty"`
	Event     *Event     `json:"event,omitempty"`
}

// データ構造定義（元の実装に合わせる）
type User struct {
	ID        int       `json:"id"`
	SessionID string    `json:"session_id"`
	Nickname  string    `json:"nickname"`
	TeamID    *int      `json:"team_id"`
	Score     int       `json:"score"`
	Connected bool      `json:"connected"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Team struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Members   []User    `json:"members,omitempty"`
}

type Question struct {
	Type    string   `json:"type"`
	Text    string   `json:"text"`
	Image   string   `json:"image,omitempty"`
	Choices []string `json:"choices"`
	Correct int      `json:"correct"`
}

type Event struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	IsActive       bool      `json:"is_active"`
	QuestionNumber int       `json:"question_number"`
	TeamMode       bool      `json:"team_mode"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// 静的設定
type StaticConfig struct {
	Title    string `json:"title"`
	TeamMode bool   `json:"team_mode"`
	TeamSize int    `json:"team_size"`
	QRCode   string `json:"qrcode"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var currentMessageIndex = 0

// 静的設定
var staticConfig = StaticConfig{
	Title:    "🎉 クイズ大会",
	TeamMode: true,
	TeamSize: 5,
	QRCode:   "/images/qr_test.png",
}

// 現在の状態（APIレスポンス用）
var currentState = struct {
	Users       []User         `json:"users"`
	Teams       []Team         `json:"teams"`
	Event       *Event         `json:"event"`
	Questions   []Question     `json:"questions"`
	ClientCount map[string]int `json:"client_count"`
}{
	Users:       []User{},
	Teams:       []Team{},
	Event:       nil,
	Questions:   []Question{},
	ClientCount: map[string]int{"admin": 0, "participant": 0, "screen": 0},
}

// テスト用メッセージ配列（API拡張データ付き）
var testMessages = []Message{
	// 2. ユーザー参加（少人数）
	{
		Type: "user_joined",
		Data: map[string]any{
			"assigned_team": nil,
			"nickname":      "太郎",
			"user": User{
				ID:        1,
				SessionID: "1",
				Nickname:  "",
				TeamID:    nil,
				Score:     0,
				Connected: false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		// APIData: &APIExtensions{
		// 	Users: []User{
		// 		{ID: 1, SessionID: "session1", Nickname: "太郎", Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 	},
		// },
	},

	// 3. ユーザー参加（中人数）
	{
		Type: "user_joined",
		Data: map[string]any{
			"assigned_team": nil,
			"nickname":      "花子",
			"user": User{
				ID:        2,
				SessionID: "2",
				Nickname:  "",
				TeamID:    nil,
				Score:     0,
				Connected: false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	},
	{
		Type: "user_joined",
		Data: map[string]any{
			"assigned_team": nil,
			"nickname":      "花子2",
			"user": User{
				ID:        3,
				SessionID: "3",
				Nickname:  "",
				TeamID:    nil,
				Score:     0,
				Connected: false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}, {
		Type: "user_joined",
		Data: map[string]any{
			"assigned_team": nil,
			"nickname":      "花子3",
			"user": User{
				ID:        4,
				SessionID: "4",
				Nickname:  "",
				TeamID:    nil,
				Score:     0,
				Connected: false,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	},

	// 4. タイトル表示
	{
		Type: "title_display",
		Data: map[string]any{
			"state": "title_display",
			"title": "🎉 テスト宴会クイズ大会へようこそ！",
		},
	},

	// 5. チーム分け結果（少人数）
	{
		Type: "team_assignment",
		Data: map[string]any{
			"teams": []map[string]any{
				{
					"name":    "チーム1",
					"members": []string{"太郎", "花子"},
				},
				{
					"name":    "チーム2",
					"members": []string{"花子2", "花子3"},
				},
			},
		},
		// APIData: &APIExtensions{
		// 	Teams: []Team{
		// 		{ID: 1, Name: "チーム1", Score: 0, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		// 			Members: []User{
		// 				{ID: 1, SessionID: "session1", Nickname: "太郎", TeamID: intPtr(1), Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 				{ID: 2, SessionID: "session2", Nickname: "花子", TeamID: intPtr(1), Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 			}},
		// 		{ID: 2, Name: "チーム2", Score: 0, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		// 			Members: []User{
		// 				{ID: 3, SessionID: "session3", Nickname: "次郎", TeamID: intPtr(2), Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 				{ID: 4, SessionID: "session4", Nickname: "三郎", TeamID: intPtr(2), Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 			}},
		// 	},
		// 	Users: []User{
		// 		{ID: 1, SessionID: "session1", Nickname: "太郎", TeamID: intPtr(1), Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 		{ID: 2, SessionID: "session2", Nickname: "花子", TeamID: intPtr(1), Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 		{ID: 3, SessionID: "session3", Nickname: "次郎", TeamID: intPtr(2), Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 		{ID: 4, SessionID: "session4", Nickname: "三郎", TeamID: intPtr(2), Score: 0, Connected: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		// 	},
		// },
	},

	// 6. 問題開始（テキスト問題）
	{
		Type: "question_start",
		Data: map[string]any{
			"question_number": 1,
			"question": map[string]any{
				"Type":    "text",
				"Text":    "Go言語の作者は誰ですか？",
				"Choices": []string{"Rob Pike", "Linus Torvalds", "Dennis Ritchie", "Ken Thompson"},
				"Correct": 1,
			},
		},
	},

	// 8. カウントダウン開始（5秒）
	{
		Type: "countdown",
		Data: map[string]any{
			"seconds_left": 5,
		},
	},

	// 9. カウントダウン（4秒）
	{
		Type: "countdown",
		Data: map[string]any{
			"seconds_left": 4,
		},
	},

	// 10. カウントダウン（3秒）
	{
		Type: "countdown",
		Data: map[string]any{
			"seconds_left": 3,
		},
	},

	// 11. カウントダウン（2秒）
	{
		Type: "countdown",
		Data: map[string]any{
			"seconds_left": 2,
		},
	},

	// 12. カウントダウン（1秒）
	{
		Type: "countdown",
		Data: map[string]any{
			"seconds_left": 1,
		},
	},

	// 13. 問題終了
	{
		Type: "question_end",
		Data: map[string]any{
			"message": "Time's up!",
		},
	},

	// 14. 回答状況表示（少人数、正解者少ない）
	{
		Type: "answer_stats",
		Data: map[string]any{
			"total_participants": 4,
			"answered_count":     3,
			"correct_count":      1,
			"correct_rate":       33.3,
			"answer_breakdown":   []int{1, 1, 1, 0},
		},
	},

	// 15. 回答発表
	{
		Type: "answer_reveal",
		Data: map[string]any{
			"correct_answer": 0,
			"explanation":    "Rob Pikeは、Go言語の主要設計者の一人です。",
		},
	},

	// 16. 問題開始（画像問題、大人数想定）
	{
		Type: "question_start",
		Data: map[string]any{
			"question_number": 2,
			"question": map[string]any{
				"Text":    "このロゴマークはどの企業のものですか？",
				"Image":   "test_logo.png",
				"Choices": []string{"Google", "Microsoft", "Apple", "Amazon"},
			},
		},
	},

	// 17. 回答状況表示（大人数、正解者多い）
	{
		Type: "answer_stats",
		Data: map[string]any{
			"total_participants": 100,
			"answered_count":     95,
			"correct_count":      80,
			"correct_rate":       84.2,
			"answer_breakdown":   []int{5, 80, 8, 2},
		},
	},

	// 18. 回答発表
	{
		Type: "answer_reveal",
		Data: map[string]any{
			"correct_answer": 1,
			"explanation":    "Microsoftのロゴマークです。",
		},
	},

	// 19. 絵文字リアクション
	{
		Type: "emoji",
		Data: map[string]any{
			"emoji":         "😊",
			"user_nickname": "太郎",
		},
	},

	// 20. 絵文字リアクション
	{
		Type: "emoji",
		Data: map[string]any{
			"emoji":         "🎉",
			"user_nickname": "花子",
		},
	},

	// 21. 最終結果（個人戦）
	{
		Type: "final_results",
		Data: map[string]any{
			"results": []map[string]any{
				{"nickname": "太郎", "score": 85, "rank": 1},
				{"nickname": "花子", "score": 80, "rank": 2},
				{"nickname": "次郎", "score": 75, "rank": 3},
				{"nickname": "三郎", "score": 70, "rank": 4},
			},
			"teams":     []map[string]any{},
			"team_mode": false,
		},
	},

	// 22. お疲れ様画面（クラッカー演出）
	{
		Type: "celebration",
		Data: map[string]any{},
	},

	// 23. チーム戦バージョン（大人数チーム分け）
	{
		Type: "team_assignment",
		Data: map[string]any{
			"teams": []map[string]any{
				{
					"name":    "チーム赤組",
					"members": []string{"太郎", "花子", "次郎", "三郎", "五郎"},
				},
				{
					"name":    "チーム青組",
					"members": []string{"一郎", "二郎", "三郎", "四郎", "五郎"},
				},
				{
					"name":    "チーム黄組",
					"members": []string{"A子", "B子", "C子", "D子", "E子"},
				},
				{
					"name":    "チーム緑組",
					"members": []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"},
				},
			},
		},
	},

	// 24. 最終結果（チーム戦）
	{
		Type: "final_results",
		Data: map[string]any{
			"results": []map[string]any{
				{"nickname": "太郎", "score": 85, "rank": 1},
				{"nickname": "花子", "score": 80, "rank": 2},
				{"nickname": "次郎", "score": 75, "rank": 3},
			},
			"teams": []map[string]any{
				{"name": "チーム赤組", "score": 320, "rank": 1},
				{"name": "チーム青組", "score": 315, "rank": 2},
				{"name": "チーム黄組", "score": 310, "rank": 3},
				{"name": "チーム緑組", "score": 305, "rank": 4},
			},
			"team_mode": true,
		},
	},

	// 25. 回答状況表示（回答者0人）
	{
		Type: "answer_stats",
		Data: map[string]any{
			"total_participants": 10,
			"answered_count":     0,
			"correct_count":      0,
			"correct_rate":       0.0,
			"answer_breakdown":   []int{0, 0, 0, 0},
		},
	},

	// 26. 回答状況表示（全員正解）
	{
		Type: "answer_stats",
		Data: map[string]any{
			"total_participants": 10,
			"answered_count":     10,
			"correct_count":      10,
			"correct_rate":       100.0,
			"answer_breakdown":   []int{10, 0, 0, 0},
		},
	},
}

func main() {
	// 静的ファイル配信
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("../../static/css/"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("../../static/js/"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("../../static/images/"))))

	// screen.htmlを/showにマッピング
	http.HandleFunc("/show", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../../static/html/screen.html")
	})

	// WebSocketエンドポイント
	http.HandleFunc("/ws/screen", handleWebSocket)

	// REST APIエンドポイント
	http.HandleFunc("/api/screen/info", handleScreenInfo)
	http.HandleFunc("/api/status", handleStatus)

	// ルートページ（操作説明）
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>Quiz Screen Test Server</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .command { background: #f5f5f5; padding: 10px; margin: 10px 0; border-left: 4px solid #007acc; }
        .current { background: #e8f4fd; font-weight: bold; }
        .note { background: #fff3cd; padding: 10px; border: 1px solid #ffeaa7; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>🎮 Quiz Screen Test Server</h1>
    <p>スクリーン表示のテスト用サーバーです。コンソールでエンターキーを押してメッセージを送信できます。</p>

    <h2>📖 使用方法</h2>
    <ol>
        <li><a href="/show" target="_blank">スクリーン表示ページ</a>を開く</li>
        <li>サーバーを起動したコンソールでエンターキーを押す</li>
        <li>メッセージが順次送信され、画面が変化します</li>
    </ol>

    <div class="note">
        <strong>💡 ヒント:</strong> 複数のタブで /show を開いて同時に確認できます
    </div>

    <h2>📋 送信可能メッセージ一覧</h2>
    <div>現在のメッセージ: <span class="current" id="current">%d / %d</span></div>
    <div id="messages">%s</div>

    <script>
        // 5秒ごとに現在のメッセージ番号を更新
        setInterval(() => {
            fetch('/current')
                .then(r => r.text())
                .then(index => {
                    document.getElementById('current').textContent = index + ' / %d';
                });
        }, 5000);
    </script>
</body>
</html>
`, currentMessageIndex, len(testMessages), generateMessageList(), len(testMessages))
	})

	// 現在のメッセージインデックスを返すAPI
	http.HandleFunc("/current", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%d", currentMessageIndex)
	})

	// キーボード入力監視を別ゴルーチンで開始
	go handleKeyboardInput()

	fmt.Println("🚀 Quiz Screen Test Server starting...")
	fmt.Println("📺 スクリーン表示: http://localhost:8080/show")
	fmt.Println("📋 操作ページ: http://localhost:8080")
	fmt.Println("⌨️  エンターキーを押してメッセージを送信してください")
	fmt.Printf("📊 利用可能メッセージ数: %d\n", len(testMessages))
	fmt.Println(strings.Repeat("-", 50))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// APIハンドラー
func handleScreenInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]any{
		"title":         staticConfig.Title,
		"team_mode":     staticConfig.TeamMode,
		"team_size":     staticConfig.TeamSize,
		"qrcode":        staticConfig.QRCode,
		"current_event": currentState.Event,
	}

	json.NewEncoder(w).Encode(response)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// クライアント数を更新
	currentState.ClientCount["screen"] = len(clients)

	response := map[string]any{
		"event":         currentState.Event,
		"users":         currentState.Users,
		"teams":         currentState.Teams,
		"client_counts": currentState.ClientCount,
		"config": map[string]any{
			"team_mode": staticConfig.TeamMode,
			"team_size": staticConfig.TeamSize,
			"title":     staticConfig.Title,
			"questions": currentState.Questions,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// データ同期関数
func updateCurrentState(msg Message) {
	if msg.APIData == nil {
		return
	}

	// Users更新
	if len(msg.APIData.Users) > 0 {
		currentState.Users = msg.APIData.Users
	}

	// Teams更新
	if len(msg.APIData.Teams) > 0 {
		currentState.Teams = msg.APIData.Teams
	}

	// Event更新
	if msg.APIData.Event != nil {
		currentState.Event = msg.APIData.Event
	}

	// Questions更新
	if len(msg.APIData.Questions) > 0 {
		currentState.Questions = msg.APIData.Questions
	}

	fmt.Printf("   🔄 API状態を更新: Users=%d, Teams=%d, Questions=%d\n",
		len(currentState.Users), len(currentState.Teams), len(currentState.Questions))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true
	fmt.Printf("📱 新しいクライアントが接続しました (合計: %d)\n", len(clients))

	// 接続時に初期メッセージを送信
	if currentMessageIndex > 0 && currentMessageIndex <= len(testMessages) {
		msg := testMessages[currentMessageIndex-1]
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("初期メッセージ送信エラー: %v", err)
			delete(clients, conn)
			return
		}
	}

	// 接続維持のためのメッセージ読み取りループ
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("📱 クライアントが切断しました (残り: %d)\n", len(clients)-1)
			delete(clients, conn)
			break
		}
	}
}

func handleKeyboardInput() {
	fmt.Println("⌨️  エンターキーを押してメッセージを送信してください...")

	for {
		var input string
		fmt.Scanln(&input)

		if currentMessageIndex >= len(testMessages) {
			fmt.Printf("📋 全メッセージ送信完了！ (最初から開始するには r を入力)\n")
			if input == "r" || input == "R" {
				currentMessageIndex = 0
				fmt.Println("🔄 メッセージインデックスをリセットしました")
			}
			continue
		}

		msg := testMessages[currentMessageIndex]
		currentMessageIndex++

		// API状態を更新
		updateCurrentState(msg)

		// 全クライアントにメッセージを送信
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("メッセージ送信エラー: %v", err)
				client.Close()
				delete(clients, client)
			}
		}

		fmt.Printf("📤 [%d/%d] %s メッセージを送信しました (接続クライアント数: %d)\n",
			currentMessageIndex, len(testMessages), msg.Type, len(clients))

		// メッセージの内容も表示
		if data, err := json.Marshal(msg.Data); err == nil {
			fmt.Printf("   📄 Data: %s\n", string(data))
		}

		if currentMessageIndex >= len(testMessages) {
			fmt.Println("🎉 全メッセージ送信完了！")
			fmt.Println("   🔄 最初から開始するには 'r' を入力してエンターを押してください")
		}
	}
}

func generateMessageList() string {
	var html strings.Builder
	for i, msg := range testMessages {
		class := ""
		if i+1 == currentMessageIndex {
			class = "current"
		}

		// データの一部を表示用に整形
		dataPreview := ""
		if data, err := json.Marshal(msg.Data); err == nil {
			dataStr := string(data)
			if len(dataStr) > 100 {
				dataStr = dataStr[:100] + "..."
			}
			dataPreview = dataStr
		}

		html.WriteString(fmt.Sprintf(
			`<div class="command %s">%d. <strong>%s</strong><br><small>%s</small></div>`,
			class, i+1, msg.Type, dataPreview))
	}
	return html.String()
}
