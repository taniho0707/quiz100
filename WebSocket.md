# WebSocket メッセージ仕様

## メッセージ構造
```json
{
  "type": "メッセージタイプ",
  "data": {任意のデータ},
  "target": "対象クライアント（オプション）"
}
```

## クライアントタイプ
- `participant`: 参加者
- `admin`: 管理者
- `screen`: スクリーン表示

## メッセージタイプ一覧

### イベント進行メッセージ
- `event_started`: イベント開始
- `title_display`: タイトル表示（スクリーン専用）
- `team_assignment`: チーム分け結果
- `question_start`: 問題開始
- `question_end`: 問題終了
- `countdown`: カウントダウン（5秒→1秒）
- `answer_stats`: 回答状況表示（スクリーン専用）
- `answer_reveal`: 回答発表
- `final_results`: 最終結果
- `celebration`: お疲れ様画面（スクリーン専用）

### ユーザー操作メッセージ
- `user_joined`: ユーザー参加通知（管理者・スクリーン）
- `user_left`: ユーザー離脱通知（管理者・スクリーン）
- `answer_received`: 回答受信通知（管理者専用）
- `emoji`: 絵文字リアクション（スクリーン専用）
- `team_member_added`: チームメンバー追加

### 状態管理メッセージ
- `state_changed`: 状態変更通知
- `initial_sync`: 初期同期
- `state_sync`: 状態同期
- `sync_request`: 同期要求
- `sync_complete`: 同期完了

### 通信監視メッセージ
- `ping`: ping送信（管理者→参加者）
- `pong`: pong応答（参加者→管理者）
- `ping_result`: ping結果（管理者専用）

## 主要メッセージのデータ例

### question_start
```json
{
  "type": "question_start",
  "data": {
    "question_number": 1,
    "question": {
      "text": "問題文",
      "image": "画像ファイル名（オプション）",
      "choices": ["選択肢1", "選択肢2", "選択肢3", "選択肢4"]
    }
  }
}
```

### answer_stats
```json
{
  "type": "answer_stats",
  "data": {
    "total_participants": 10,
    "answered_count": 8,
    "correct_count": 5,
    "correct_rate": 62.5,
    "answer_breakdown": [2, 3, 2, 1]
  }
}
```

### team_assignment
```json
{
  "type": "team_assignment",
  "data": {
    "teams": [
      {"name": "チーム1", "members": ["太郎", "花子"]},
      {"name": "チーム2", "members": ["次郎", "三郎"]}
    ]
  }
}
```

### countdown
```json
{
  "type": "countdown",
  "data": {
    "seconds_left": 3
  }
}
```

### emoji
```json
{
  "type": "emoji",
  "data": {
    "emoji": "😊",
    "user_nickname": "太郎"
  }
}
```

### final_results
```json
{
  "type": "final_results",
  "data": {
    "results": [
      {"nickname": "太郎", "score": 85, "rank": 1},
      {"nickname": "花子", "score": 80, "rank": 2}
    ],
    "teams": [
      {"name": "チーム1", "score": 165, "rank": 1}
    ],
    "team_mode": true
  }
}
```