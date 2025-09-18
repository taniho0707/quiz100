package main

// テストケース設定の参考例
// この配列を編集することで、様々なシナリオをテストできます

// 🎯 テストケース別メッセージ定義の参考例
//
// 下記のパターンを参考に、main.goのtestMessages配列を編集してください：

// 1. 【少人数シナリオ】 - 4人程度の小規模クイズ
var smallGroupScenario = []Message{
	// 参加者登録（少人数）
	{Type: "user_joined", Data: map[string]interface{}{"user_id": 1, "nickname": "太郎"}},
	{Type: "user_joined", Data: map[string]interface{}{"user_id": 2, "nickname": "花子"}},
	{Type: "user_joined", Data: map[string]interface{}{"user_id": 3, "nickname": "次郎"}},
	{Type: "user_joined", Data: map[string]interface{}{"user_id": 4, "nickname": "三郎"}},

	// 回答状況（少人数、低正解率）
	{Type: "answer_stats", Data: map[string]interface{}{
		"total_participants": 4,
		"answered_count":     3,
		"correct_count":      1,
		"correct_rate":       33.3,
		"answer_breakdown":   []int{1, 1, 1, 0},
	}},
}

// 2. 【大人数シナリオ】 - 100人規模の大型クイズ
var largeGroupScenario = []Message{
	// 回答状況（大人数、高正解率）
	{Type: "answer_stats", Data: map[string]interface{}{
		"total_participants": 100,
		"answered_count":     95,
		"correct_count":      80,
		"correct_rate":       84.2,
		"answer_breakdown":   []int{5, 80, 8, 2},
	}},

	// チーム分け（大人数）
	{Type: "team_assignment", Data: map[string]interface{}{
		"teams": []map[string]interface{}{
			{"name": "チーム赤組", "members": []string{"太郎", "花子", "次郎", "三郎", "五郎", "六郎", "七郎", "八郎"}},
			{"name": "チーム青組", "members": []string{"一郎", "二郎", "三郎", "四郎", "五郎", "六郎", "七郎", "八郎"}},
			{"name": "チーム黄組", "members": []string{"A子", "B子", "C子", "D子", "E子", "F子", "G子", "H子"}},
			{"name": "チーム緑組", "members": []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta"}},
		},
	}},
}

// 3. 【エッジケースシナリオ】 - 極端な状況のテスト
var edgeCaseScenario = []Message{
	// 回答者0人
	{Type: "answer_stats", Data: map[string]interface{}{
		"total_participants": 10,
		"answered_count":     0,
		"correct_count":      0,
		"correct_rate":       0.0,
		"answer_breakdown":   []int{0, 0, 0, 0},
	}},

	// 全員正解
	{Type: "answer_stats", Data: map[string]interface{}{
		"total_participants": 10,
		"answered_count":     10,
		"correct_count":      10,
		"correct_rate":       100.0,
		"answer_breakdown":   []int{10, 0, 0, 0},
	}},

	// 1人チーム
	{Type: "team_assignment", Data: map[string]interface{}{
		"teams": []map[string]interface{}{
			{"name": "チーム孤独", "members": []string{"太郎"}},
		},
	}},
}

// 4. 【画面遷移シナリオ】 - 全画面の確認
var fullTransitionScenario = []Message{
	// 1. 待機画面
	{Type: "event_started", Data: map[string]interface{}{"event_title": "フル遷移テスト"}},

	// 2. タイトル表示
	{Type: "title_display", Data: map[string]interface{}{"title": "🎉 フル遷移テストへようこそ！"}},

	// 3. チーム分け
	{Type: "team_assignment", Data: map[string]interface{}{
		"teams": []map[string]interface{}{
			{"name": "テストチーム", "members": []string{"テスター1", "テスター2"}},
		},
	}},

	// 4. 問題表示
	{Type: "question_start", Data: map[string]interface{}{
		"question_number": 1,
		"question": map[string]interface{}{
			"text":    "これはテスト問題です。",
			"choices": []string{"選択肢A", "選択肢B", "選択肢C", "選択肢D"},
		},
	}},

	// 5. カウントダウン
	{Type: "countdown", Data: map[string]interface{}{"seconds_left": 3}},
	{Type: "countdown", Data: map[string]interface{}{"seconds_left": 2}},
	{Type: "countdown", Data: map[string]interface{}{"seconds_left": 1}},

	// 6. 問題終了
	{Type: "question_end", Data: map[string]interface{}{}},

	// 7. 回答状況
	{Type: "answer_stats", Data: map[string]interface{}{
		"total_participants": 5,
		"answered_count":     4,
		"correct_count":      2,
		"correct_rate":       50.0,
		"answer_breakdown":   []int{2, 1, 1, 0},
	}},

	// 8. 回答発表
	{Type: "answer_reveal", Data: map[string]interface{}{
		"correct_answer": 0,
		"explanation":    "選択肢Aが正解でした。",
	}},

	// 9. 最終結果
	{Type: "final_results", Data: map[string]interface{}{
		"results": []map[string]interface{}{
			{"nickname": "テスター1", "score": 100, "rank": 1},
			{"nickname": "テスター2", "score": 80, "rank": 2},
		},
		"teams":     []map[string]interface{}{},
		"team_mode": false,
	}},

	// 10. お疲れ様画面
	{Type: "celebration", Data: map[string]interface{}{}},
}

// 5. 【絵文字テストシナリオ】 - リアクションの確認
var emojiTestScenario = []Message{
	{Type: "emoji", Data: map[string]interface{}{"emoji": "😊", "user_nickname": "太郎"}},
	{Type: "emoji", Data: map[string]interface{}{"emoji": "🎉", "user_nickname": "花子"}},
	{Type: "emoji", Data: map[string]interface{}{"emoji": "👏", "user_nickname": "次郎"}},
	{Type: "emoji", Data: map[string]interface{}{"emoji": "❤️", "user_nickname": "三郎"}},
	{Type: "emoji", Data: map[string]interface{}{"emoji": "🔥", "user_nickname": "五郎"}},
}

// 6. 【画像問題シナリオ】 - 画像つき問題の確認
var imageQuestionScenario = []Message{
	{Type: "question_start", Data: map[string]interface{}{
		"question_number": 1,
		"question": map[string]interface{}{
			"text":    "この画像は何を表していますか？",
			"image":   "test_image.png", // 実際の画像ファイルは不要（表示確認用）
			"choices": []string{"選択肢A", "選択肢B", "選択肢C", "選択肢D"},
		},
	}},
}

// 🛠️ 使用方法：
//
// 1. main.goのtestMessages変数を任意のシナリオで置き換える
//    例: var testMessages = smallGroupScenario
//
// 2. 複数シナリオを組み合わせる場合：
//    var testMessages = append(append(smallGroupScenario, fullTransitionScenario...), emojiTestScenario...)
//
// 3. カスタムシナリオを作成する場合：
//    上記のパターンを参考に新しいMessage配列を定義
//
// 💡 テストのコツ：
// - 人数を変えて表示レイアウトを確認
// - 正解率を極端に設定して状況表示を確認
// - チーム数・メンバー数を変えてグリッド表示を確認
// - カウントダウンの表示タイミングを確認
// - 絵文字の重複表示動作を確認
