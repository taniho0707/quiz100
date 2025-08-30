#!/bin/bash

# クイズシステム起動スクリプト

set -e

echo "🎮 Quiz100 システムを起動中..."

# 環境変数の確認
if [ -z "$QUIZ_ADMIN_IP_ADDR" ] && [ -z "$QUIZ_ADMIN_MAC_ADDR" ] && [ -z "$QUIZ_ADMIN_TOKEN" ]; then
    echo "⚠️  警告: 管理者認証の環境変数が設定されていません"
    echo "   QUIZ_ADMIN_IP_ADDR, QUIZ_ADMIN_MAC_ADDR, QUIZ_ADMIN_TOKEN のいずれかを設定してください"
    echo "   例: export QUIZ_ADMIN_IP_ADDR=\"192.168.1.100\""
    exit 1
fi

# 必要なディレクトリの作成
echo "📁 ディレクトリを作成中..."
mkdir -p logs database static/images

# 設定ファイルの確認
if [ ! -f "config/quiz.toml" ]; then
    echo "❌ 設定ファイル config/quiz.toml が見つかりません"
    exit 1
fi

# バイナリのビルド
echo "🔨 アプリケーションをビルド中..."
go build -o quiz100 .

# 起動
echo "🚀 サーバーを起動中..."
echo "   管理者ページ: http://localhost:8080/admin"
echo "   参加者ページ: http://localhost:8080/"
echo "   スクリーン表示: http://localhost:8080/show"
echo "   ヘルスチェック: http://localhost:8080/api/health"
echo ""
echo "💡 停止するには Ctrl+C を押してください"

# 実行
./quiz100