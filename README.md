# Quiz100 - 宴会クイズシステム

大人数での宴会で使用するリアルタイムクイズシステムです。WebSocketを使用してリアルタイムでクイズイベントを進行できます。

## ✨ 特徴

- 🎯 **リアルタイム参加**: WebSocketによる即座の状態同期
- 👥 **チーム戦対応**: 自動チーム分け機能付き
- 📱 **レスポンシブデザイン**: スマートフォン・タブレット・デスクトップ対応
- 🖼️ **画像問題対応**: テキストと画像両方の問題形式
- 😀 **絵文字リアクション**: 参加者の反応をリアルタイム表示
- 🔒 **セキュアな管理**: MAC/IP/TOKEN認証による管理者アクセス制限
- 📊 **詳細監視**: ヘルスチェック・デバッグ情報・統計情報
- ⏰ **制限時間アラート**: 残り5秒アラート機能

## 🚀 クイックスタート

### 必要要件

- Go 1.19以上
- Docker (オプション)

### 1. ネイティブ実行

```bash
# 1. 管理者認証の設定
export QUIZ_ADMIN_IP_ADDR="192.168.1.100"  # 管理者のIPアドレス

# 2. システム起動
./scripts/start.sh
```

### 2. Docker実行

```bash
# 1. Docker環境での起動
./scripts/start-docker.sh

# 2. ログ確認
docker-compose logs -f
```

## 🔧 設定

### 管理者認証設定

管理者ページとAPIへのアクセスを制限するため、以下のいずれかの環境変数を設定：

```bash
# IP認証 (推奨 - Docker環境)
export QUIZ_ADMIN_IP_ADDR="192.168.1.100,10.0.0.5"

# MAC認証 (ネイティブ実行)
export QUIZ_ADMIN_MAC_ADDR="aa:bb:cc:dd:ee:ff,11:22:33:44:55:66"

# TOKEN認証 (API経由アクセス)
export QUIZ_ADMIN_TOKEN="your-secret-token-here"
```

### クイズ設定ファイル

`config/quiz.toml` でクイズ内容を設定：

```toml
[event]
title = "新年会クイズ大会"
team_mode = true
team_size = 5

[team_separation]
avoid_groups = ["田中", "山田", "佐藤"]

[[questions]]
type = "text"
text = "Goの作者は誰？"
choices = ["Rob Pike", "Linus Torvalds", "Dennis Ritchie", "Ken Thompson"]
correct = 1

[[questions]]
type = "image"
text = "この画像は何？"
image = "sample.png"
choices = ["選択肢1", "選択肢2", "選択肢3"]
correct = 2
```

## 🎮 使用方法

### 管理者

1. **http://localhost:8080/admin** にアクセス
2. 「イベント開始」でクイズ開始
3. 「次の問題」で問題を順次出題
4. 「⏰ 残り5秒アラート」で時間切れ警告
5. 「イベント終了」で結果発表

### 参加者

1. **http://localhost:8080/** にアクセス
2. ニックネームを入力して参加
3. 問題が表示されたら回答選択
4. 絵文字ボタンでリアクション

### スクリーン表示

1. **http://localhost:8080/show** にアクセス
2. プロジェクター等で参加者全員に表示

## 🔍 監視・運用

### ヘルスチェック

```bash
# システム状態確認
./scripts/health-check.sh

# API経由での確認
curl http://localhost:8080/api/health
```

### バックアップ・復元

```bash
# データバックアップ
./scripts/backup.sh

# データ復元
./scripts/restore.sh backups/20241127_143000.tar.gz
```

### デバッグ情報

```bash
# 詳細システム情報 (管理者認証が必要)
curl http://localhost:8080/api/admin/debug
```

## 🧪 テスト実行

```bash
# 全テスト実行
go test ./...

# 詳細出力
go test ./... -v

# 特定パッケージのテスト
go test ./models -v
go test ./handlers -v
```

## 📊 API エンドポイント

### 公開API

- `GET /` - 参加者ページ
- `POST /api/join` - 参加者登録
- `POST /api/answer` - 回答送信
- `POST /api/emoji` - 絵文字送信
- `GET /api/status` - システム状態
- `GET /api/health` - ヘルスチェック

### 管理者API (認証必要)

- `GET /admin` - 管理者ページ
- `POST /api/admin/start` - イベント開始
- `POST /api/admin/next` - 次の問題
- `POST /api/admin/alert` - 5秒アラート
- `POST /api/admin/stop` - イベント終了
- `POST /api/admin/teams` - チーム作成
- `GET /api/admin/debug` - デバッグ情報

### WebSocket

- `ws://localhost:8080/ws/participant` - 参加者用
- `ws://localhost:8080/ws/admin` - 管理者用 (認証必要)
- `ws://localhost:8080/ws/screen` - スクリーン用 (認証必要)

## 📁 ディレクトリ構造

```
quiz100/
├── main.go                 # メインアプリケーション
├── config/
│   └── quiz.toml          # クイズ設定ファイル
├── handlers/              # HTTPハンドラー
├── models/                # データモデル
├── middleware/            # 認証ミドルウェア
├── websocket/             # WebSocket処理
├── database/              # データベース処理
├── static/                # 静的ファイル
│   ├── html/             # HTMLテンプレート
│   ├── css/              # スタイルシート
│   ├── js/               # JavaScript
│   └── images/           # 問題画像
├── scripts/               # 運用スクリプト
├── logs/                 # ログファイル
└── backups/              # バックアップファイル
```

## 🛠️ 開発

### 新機能の追加

1. 適切なパッケージに実装を追加
2. テストを作成・実行
3. ドキュメントを更新

### ログ確認

```bash
# リアルタイムログ
tail -f logs/quiz_*.log

# エラーログのみ
grep ERROR logs/quiz_*.log
```

## 🔧 トラブルシューティング

### よくある問題

**Q: 管理者ページにアクセスできない**
A: 環境変数 `QUIZ_ADMIN_IP_ADDR` 等が正しく設定されているか確認

**Q: WebSocket接続が失敗する**
A: ファイアウォール設定とポート8080の開放を確認

**Q: 画像が表示されない**
A: `static/images/` ディレクトリに画像ファイルが存在するか確認

**Q: データベースエラーが発生する**
A: `database/` ディレクトリの権限とディスク容量を確認

### システム要件

- **メモリ**: 最小512MB、推奨1GB以上
- **ディスク**: 最小100MB、推奨1GB以上
- **ネットワーク**: ポート8080のアクセス許可
- **同時接続数**: 最大100人 (設計上限)

## 📝 ライセンス

このプロジェクトはMITライセンスで公開されています。

## 🤝 貢献

Issues・Pull Requestをお待ちしています！

---

## 🎉 楽しいクイズイベントを！

Quiz100は宴会や社内イベントでの利用を想定して開発されました。参加者全員でワイワイ楽しめるクイズイベントをお楽しみください！