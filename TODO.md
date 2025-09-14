# 実装が必要な機能
- ユーザーが正答を知るのは結果発表のタイミング
- 回答受付中に show に正答を表示しない
- アニメーションをつけて show に正答を表示する
- show に表示する問題を順番に前後に移動できる補助機能を用意する
- 100人の同時負荷テスト
- 同時負荷テストで何台かを通信状況が不安定な状態を再現する
- 下記に該当する要素にアクセスするコードを削除
```                
                <div class="event-status">
                    <h3>📊 イベント状況</h3>
                    <p>状態: <span id="event-status">待機中</span></p>
                    <p><span id="current-question">問題 0</span></p>
                    <p><span id="participant-count-display">参加者: 0名</span></p>
                    <p>接続: <span id="connection-status-display">🔴 未接続</span></p>
                </div>
```

# リファクタリング実装計画

## 🔍 特定された問題点

### 1. **重複コードと保守性の課題**
- **日本語状態ラベルの重複**: `admin.js`, `screen.js`, `handlers.go` で同じ状態ラベルを定義
- **状態変換ロジックの散在**: String ↔ EventState変換が複数箇所に存在
- **WebSocketメッセージタイプの分散**: 12種類以上のメッセージタイプが各JSファイルに散在

### 2. **責任の分離不足**
- **handlers/handlers.go**: 1200行超で状態管理・WebSocket・API・DB操作が混在
- **各JSファイル**: 状態管理・UI制御・WebSocket通信が混在
- **単一責任原則違反**: 1つのファイルが多すぎる責任を持つ

### 3. **状態同期の複雑性**
- **状態とデータの整合性**: `currentQuestion`等の状態管理が複雑
- **非同期処理のエラーハンドリング**: 一貫性のないエラー処理パターン
- **状態遷移の検証不足**: 不正な状態遷移の防止機構が不十分

### 4. **テスタビリティの問題**
- **高い結合度**: コンポーネント間の依存関係が複雑
- **ハードコーディング**: 設定値やマジックナンバーが散在

## 📋 段階的リファクタリング計画

### **Phase 1: 共通定数・ユーティリティの抽出** ⚡ 即座に実施可能
```
優先度: 🔥 高 | 影響度: 低 | 工数: 1-2時間
```

#### 1.1 状態定数の統一化
- **新ファイル**: `models/constants.go`
  - EventState定数とラベル定義を一元化
  - State ↔ String変換関数を追加
  
#### 1.2 WebSocketメッセージタイプの統一化
- **新ファイル**: `websocket/message_types.go`
  - 全メッセージタイプを列挙型で定義
  - メッセージ構造体の標準化

#### 1.3 フロントエンド共通ユーティリティ
- **新ファイル**: `static/js/common/constants.js`
  - 状態ラベル・メッセージタイプの共通定義
- **新ファイル**: `static/js/common/utils.js`  
  - 状態変換・エラーハンドリングユーティリティ

### **Phase 2: バックエンド責任分離** 🏗️ 1-2日の作業
```
優先度: 🔥 高 | 影響度: 中 | 工数: 1-2日
```

#### 2.1 WebSocket管理の分離
- **新ファイル**: `websocket/hub_manager.go`
  - Hub操作を専門に扱うサービス
  - メッセージブロードキャスト処理の統一化

#### 2.2 状態管理サービスの作成
- **新ファイル**: `services/state_service.go`
  - EventStateManagerのラッパー
  - 状態遷移の検証・ログ処理を統一

#### 2.3 ハンドラーの分割
- **handlers/handlers.go**を機能別に分割:
  - `handlers/participant_handlers.go` - 参加者関連API
  - `handlers/admin_handlers.go` - 管理者関連API
  - `handlers/websocket_handlers.go` - WebSocket接続処理

### **Phase 3: フロントエンド アーキテクチャ改善** 🎨 2-3日の作業
```
優先度: 🔶 中 | 影響度: 中 | 工数: 2-3日
```

#### 3.1 状態管理の統一化
- **新ファイル**: `static/js/common/state_manager.js`
  - 各クライアントタイプ共通の状態管理機能
  - 状態遷移の検証・同期処理

#### 3.2 WebSocket通信の抽象化
- **新ファイル**: `static/js/common/websocket_client.js`
  - WebSocket接続・再接続・メッセージハンドリングの統一
  - エラーハンドリングの標準化

#### 3.3 UI コンポーネントの分離
- 各JSファイルからUI制御部分を抽出:
  - `static/js/components/admin_ui.js`
  - `static/js/components/screen_ui.js`
  - `static/js/components/participant_ui.js`

### **Phase 4: エラーハンドリング・検証強化** 🛡️ 1-2日の作業
```
優先度: 🔶 中 | 影響度: 高 | 工数: 1-2日
```

#### 4.1 エラーハンドリングの標準化
- **新ファイル**: `errors/quiz_errors.go`
  - カスタムエラータイプの定義
  - エラーレスポンスの統一化

#### 4.2 バリデーション層の追加
- **新ファイル**: `validation/request_validator.go`
  - リクエストデータの検証
  - 状態遷移の妥当性チェック

#### 4.3 ロバストネスの向上
- WebSocket切断時の自動復旧処理
- 状態不整合時の自動修正機構
- クライアント間同期エラーの検出・修復

## 🎯 期待される効果

### **即座の効果 (Phase 1完了後)**
- 重複コードの削減（約200行削減予想）
- 状態ラベル変更時の修正箇所が3箇所→1箇所に
- 新しいWebSocketメッセージタイプ追加の工数50%削減

### **中期的効果 (Phase 2-3完了後)**  
- ファイルサイズの適正化（handlers.goを1200行→400行程度に）
- 機能追加時の影響範囲の限定化
- 単体テスト作成の容易化

### **長期的効果 (全Phase完了後)**
- 新機能開発速度の向上（見積もり30%短縮）
- バグ発生率の削減（状態関連バグの大幅減少）
- 新メンバーのオンボーディング時間短縮

## ⚠️ リスクと対策

### **リスク**
- 既存機能への影響
- リファクタリング中のバグ混入
- 開発期間の延長

### **対策**
- 段階的実施による影響最小化
- 各Phase後の動作確認テスト実施  
- 既存機能の詳細テストケース作成

## 📅 実装スケジュール

### **推奨実施順序**
1. **Phase 1** (1-2時間): 共通定数・ユーティリティの抽出
2. **Phase 2** (1-2日): バックエンド責任分離
3. **Phase 3** (2-3日): フロントエンド アーキテクチャ改善
4. **Phase 4** (1-2日): エラーハンドリング・検証強化

### **実施状況**
- [x] Phase 1: 共通定数・ユーティリティの抽出 ✅ **完了 (2025/9/4)**
  - [x] 1.1 状態定数の統一化 - `models/constants.go`作成
    - EventState定数・日本語ラベル・変換関数を一元化
    - `handlers.go`の重複コード削除（約60行削減）
  - [x] 1.2 WebSocketメッセージタイプの統一化 - `websocket/message_types.go`作成
    - 16種類のメッセージタイプを統一管理
    - 型安全なメッセージ作成関数を追加
  - [x] 1.3 フロントエンド共通ユーティリティ - `static/js/common/`作成
    - `constants.js`: 状態・メッセージタイプの共通定義
    - `utils.js`: 状態変換・エラーハンドリング・バリデーション
    - 全HTMLファイルに共通スクリプトを追加
    - 3つのJSファイルから重複する状態ラベル定義を削除
- [x] Phase 2: バックエンド責任分離 ✅ **完了 (2025/9/4)**
  - [x] 2.1 WebSocket管理の分離 - `websocket/hub_manager.go`作成
    - 高レベルなWebSocket操作を提供する専用サービス
    - 16種類のメッセージタイプ別ブロードキャスト関数
    - 型安全なメッセージ処理とエラーハンドリング
  - [x] 2.2 状態管理サービスの作成 - `services/state_service.go`作成
    - EventStateManagerのラッパーサービス
    - 状態遷移の検証・ログ・WebSocket通知の統合
    - カウントダウン自動遷移・結果構造体による詳細なフィードバック
  - [x] 2.3 ハンドラーの分割 - `handlers/`分割
    - `participant_handlers.go`: 参加者関連API (Join, Answer, Emoji, ResetSession)
    - `admin_handlers.go`: 管理者関連API (AdminAction, StateJump, AvailableActions)
    - `websocket_handlers.go`: WebSocket接続・ユーティリティAPI (Status, Health, Debug)
- [x] Phase 3: フロントエンド アーキテクチャ改善 ✅ **完了 (2025/9/4)**
  - [x] 3.1 状態管理の統一化 - `static/js/common/state_manager.js`作成
    - 各クライアントタイプ共通の状態管理機能
    - 状態遷移の検証・同期処理・デバウンス機能
    - WebSocketメッセージからの自動状態更新
  - [x] 3.2 WebSocket通信の抽象化 - `static/js/common/websocket_client.js`作成
    - 接続・再接続・メッセージハンドリングの統一
    - エラーハンドリングの標準化・ハートビート機能
    - 送信待ちキュー・接続状態管理
  - [x] 3.3 UI コンポーネントの分離 - `static/js/components/`作成
    - `admin_ui.js`: 管理者画面のUI制御・ボタン管理・ログ表示
    - `screen_ui.js`: スクリーン表示画面・アニメーション・エフェクト
    - `participant_ui.js`: 参加者画面・フォーム・インタラクション
  - [x] HTMLファイル更新 - 新しい共通スクリプトとコンポーネントを統合
- [x] Phase 4: エラーハンドリング・検証強化 ✅ **完了 (2025/9/4)**
  - [x] 4.1 エラーハンドリングの標準化 - `errors/quiz_errors.go`作成
    - カスタムエラータイプ QuizError の定義 (25種類のエラーコード)
    - エラーレスポンスの統一化 (ErrorResponse, SuccessResponse)
    - エラー分類ヘルパー関数 (IsUserError, IsStateError, IsValidationError等)
    - エラーファクトリー関数とチェーン機能 (WithDetails, WithCause)
  - [x] 4.2 バリデーション層の追加 - `validation/request_validator.go`作成  
    - リクエストデータの包括的検証 (ニックネーム、回答、状態遷移)
    - フィールドレベルのエラー詳細報告 (ValidationResult構造体)
    - バッチ検証メソッド (ValidateJoinRequest, ValidateAnswerRequest等)
    - 設定整合性チェック・レート制限・システム状態検証
  - [x] 4.3 ロバストネスの向上 - `services/robustness_service.go`作成
    - WebSocket切断時の自動復旧処理とヘルスモニタリング
    - 状態不整合検出・修復機構とクライアント同期エラー対応
    - システムメトリクス収集とパフォーマンス監視
    - 自動回復メカニズムと緊急時の安全停止機能
