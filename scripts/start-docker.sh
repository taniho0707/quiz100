#!/bin/bash

# Docker環境でのクイズシステム起動スクリプト

set -e

echo "🐳 Docker環境でQuiz100システムを起動中..."

# .envファイルの確認
if [ ! -f ".env" ]; then
    echo "⚠️  .envファイルが見つかりません。作成します..."
    cat > .env << 'EOF'
# 管理者認証設定 (いずれか一つ以上を設定)
# Docker環境では IP認証 または TOKEN認証を推奨
QUIZ_ADMIN_IP_ADDR=192.168.1.100
# QUIZ_ADMIN_MAC_ADDR=aa:bb:cc:dd:ee:ff,11:22:33:44:55:66
# QUIZ_ADMIN_TOKEN=your-secret-admin-token-here

# ポート設定
PORT=8080

# データベース設定
DATABASE_PATH=./database/quiz.db

# ログ設定
LOG_LEVEL=info
EOF
    echo "✅ .envファイルを作成しました。必要に応じて編集してください。"
fi

# docker-compose.ymlの確認
if [ ! -f "docker-compose.yml" ]; then
    echo "❌ docker-compose.yml が見つかりません"
    exit 1
fi

# コンテナの停止・削除
echo "🧹 既存のコンテナを停止・削除中..."
docker-compose down --remove-orphans

# イメージのビルド
echo "🔨 Dockerイメージをビルド中..."
docker-compose build --no-cache

# コンテナの起動
echo "🚀 コンテナを起動中..."
docker-compose up -d

# ステータスの確認
echo "📊 コンテナステータス:"
docker-compose ps

# ログの表示
echo ""
echo "📋 ログを表示中... (Ctrl+C で停止)"
echo "   管理者ページ: http://localhost:8080/admin"
echo "   参加者ページ: http://localhost:8080/"
echo "   スクリーン表示: http://localhost:8080/show"
echo "   ヘルスチェック: http://localhost:8080/api/health"
echo ""

docker-compose logs -f quiz100