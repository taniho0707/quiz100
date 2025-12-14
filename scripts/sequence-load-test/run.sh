#!/bin/bash

# シーケンス負荷テスト実行スクリプト

set -e

cd "$(dirname "$0")"

echo "🚀 シーケンス負荷テスト準備中..."

# 依存関係のチェック
if [ ! -d "vendor" ] && [ ! -f "go.sum" ]; then
    echo "📦 依存関係をダウンロード中..."
    go mod download
fi

# サーバーが起動しているか確認
echo "🔍 サーバーの状態を確認中..."
if ! curl -s http://localhost:8080/api/status > /dev/null 2>&1; then
    echo ""
    echo "❌ エラー: サーバーが起動していません"
    echo ""
    echo "別のターミナルで以下のコマンドを実行してサーバーを起動してください："
    echo "  cd /mnt/s/git/quiz100"
    echo "  go run main.go"
    echo ""
    exit 1
fi

echo "✅ サーバーが起動しています"
echo ""
echo "📝 注意事項:"
echo "  - 管理者画面 (http://localhost:8080/admin) を開いておいてください"
echo "  - テストの指示に従ってエンターキーを押してください"
echo "  - 各段階で管理者画面で適切な操作を行ってください"
echo ""
read -p "準備ができたらエンターキーを押してください... " -r
echo ""

# テスト実行
go run main.go

echo ""
echo "✅ テスト完了"
