# WebSocket メッセージ仕様

## メッセージ構造
```json
{
  "type": "メッセージタイプ",
  "data": {任意のデータ},
}
```

## クライアントタイプ
- `participant`: 参加者
- `admin`: 管理者
- `screen`: スクリーン表示

## メッセージタイプ一覧

### イベント進行メッセージ
- `title_display`: タイトル表示 (screen)
- `team_assignment`: チーム分け結果 (admin/screen)
- `question_start`: 問題開始 (admin/screen/participant)
- `countdown`: カウントダウン (5秒のみ)
- `question_end`: 問題終了 (admin/screen/participant)
- `answer_stats`: 回答状況表示 (admin/screen/participant)
- `answer_reveal`: 回答発表 (admin/screen)
- `final_results`: 最終結果 (admin/screen/participant)

### ユーザー操作メッセージ
- `user_joined`: ユーザー参加通知 (admin/screen)
- `user_left`: ユーザー離脱通知 (admin/screen)
- `answer_received`: 回答受信通知 (admin)
- `emoji`: 絵文字リアクション (admin/screen)
- `team_member_added`: チームメンバー追加 (admin/participant)

### 状態管理メッセージ
- `state_changed`: 状態変更通知 = デバッグ専用
- `initial_sync`: 初期同期 (admin/screen/participant)
- `state_sync`: 状態同期 (admin/screen/participant)

### 通信監視メッセージ
- `ping`: ping送信 (participant)
- `pong`: pong応答 (participant)
- `ping_result`: ping結果 (admin)

## 主要メッセージのデータ例

### title_display: screen
```json
{
  "type": "title_display",
  "data": {
    "title": "新年会クイズ大会"
  },
}
```

### team_assignment: admin/screen
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

### question_start: admin/screen/participant
```json
{
  "type": "question_start",
  "data": {
    "question_number": 1,
    "question": {
      "type": "text",
      "text": "問題文",
      "image": "画像ファイル名（オプション）",
      "choices": ["選択肢1", "選択肢2", "選択肢3", "選択肢4"]
    },
    "correct": 0, // only for admin
    "total_questions": 5 // only for admin
  }
}
```

### countdown: screen
カウントダウン開始のみ通知、0と同時にquestion_endと同様の表示に自動遷移
```json
{
  "type": "countdown",
  "data": {
    "seconds_left": 5
  }
}
```

### question_end: admin/screen/participant
```json
{
  "type": "question_end",
  "data": {}
}
```

### answer_stats: admin/screen
```json
{
  "type": "answer_stats",
  "data": {
    "total_participants": 10,
    "choices_counts": [2, 3, 2, 1]
  }
}
```

### answer_reveal: admin/screen/participant
```json
{
  "type": "answer_reveal",
  "data": {
    "correct": 0,
  }
}
```

### final_results: admin/screen/participant
時間経過後に celebration 同等の表示を行う
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

### user_joined: admin/screen
```json
{
  "type": "user_joined",
  "data": {
    "teamname": "チーム1",
    "nickname": "太郎",
    "score": 0,
  }
}
```

### user_left: admin/screen
```json
{
  "type": "user_left",
  "data": {
    "nickname": "太郎",
    "team_id": 1 | null
  }
}
```

### answer_received: admin
```json
{
  "type": "answer_received",
  "data": {
    "nickname": "太郎",
    "answer": 0
  }
}
```

### emoji: admin/screen
```json
{
  "type": "emoji",
  "data": {
    "emoji": "😊",
    "nickname": "太郎"
  }
}
```

### team_member_added: admin/participant
/// TODO
```json
{
  "type": "team_member_added",
  "data": {
    "team_id": 1,
    "nickname": "太郎"
  }
}
```

### state_changed (for debug)
/// TODO
```json

```

### initial_sync: admin/screen/participant
/// TODO
```json

```

### state_sync: admin/screen/participant
/// TODO
```json

```

### ping: participant
```json
{
  "type": "ping",
  "data": {}
}
```

### pong: participant
```json
{
  "type": "pong",
  "data": {}
}
```

### ping_result: admin
```json
{
  "type": "ping_result",
  "data": {
    "nickname": "太郎",
    "result": 10 | null // null means unreachable
  }
}
```
