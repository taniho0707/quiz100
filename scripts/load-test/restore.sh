#!/bin/bash

# バックアップからのデータ復元スクリプト

set -e

# 引数の確認
if [ $# -ne 1 ]; then
    echo "使用方法: $0 <backup_file>"
    echo "例: $0 backups/20241127_143000.tar.gz"
    echo ""
    echo "📂 利用可能なバックアップ一覧:"
    ls -la backups/*.tar.gz 2>/dev/null || echo "バックアップファイルがありません"
    exit 1
fi

BACKUP_FILE="$1"

# バックアップファイルの存在確認
if [ ! -f "$BACKUP_FILE" ]; then
    echo "❌ バックアップファイルが見つかりません: $BACKUP_FILE"
    exit 1
fi

echo "🔄 Quiz100 データを復元中..."
echo "📁 バックアップファイル: $BACKUP_FILE"

# 確認プロンプト
echo ""
echo "⚠️  注意: 現在のデータは上書きされます。"
read -p "復元を実行しますか? (y/N): " confirm

if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    echo "❌ 復元を中止しました"
    exit 0
fi

# 現在のデータのバックアップ
echo "💾 現在のデータを緊急バックアップ中..."
EMERGENCY_BACKUP="backups/emergency_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$EMERGENCY_BACKUP"

[ -f "database/quiz.db" ] && cp "database/quiz.db" "$EMERGENCY_BACKUP/"
[ -d "logs" ] && cp -r logs "$EMERGENCY_BACKUP/"
[ -f "config/quiz.toml" ] && mkdir -p "$EMERGENCY_BACKUP/config" && cp "config/quiz.toml" "$EMERGENCY_BACKUP/config/"

echo "✅ 緊急バックアップ完了: $EMERGENCY_BACKUP"

# バックアップファイルの展開
echo "📦 バックアップファイルを展開中..."
TEMP_DIR="temp_restore_$(date +%s)"
mkdir -p "$TEMP_DIR"
tar -xzf "$BACKUP_FILE" -C "$TEMP_DIR"

# 展開されたディレクトリを見つける
RESTORE_DIR=$(find "$TEMP_DIR" -type d -name "*" | head -n 1)

if [ ! -d "$RESTORE_DIR" ]; then
    echo "❌ バックアップファイルの形式が不正です"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# データの復元
echo "🔄 データを復元中..."

# データベースの復元
if [ -f "$RESTORE_DIR/quiz.db" ]; then
    echo "📊 データベースを復元中..."
    mkdir -p database
    cp "$RESTORE_DIR/quiz.db" "database/quiz.db"
    echo "✅ データベース復元完了"
fi

# ログの復元
if [ -d "$RESTORE_DIR/logs" ]; then
    echo "📋 ログファイルを復元中..."
    rm -rf logs
    cp -r "$RESTORE_DIR/logs" logs
    echo "✅ ログファイル復元完了"
fi

# 設定ファイルの復元
if [ -f "$RESTORE_DIR/config/quiz.toml" ]; then
    echo "⚙️  設定ファイルを復元中..."
    mkdir -p config
    cp "$RESTORE_DIR/config/quiz.toml" "config/quiz.toml"
    echo "✅ 設定ファイル復元完了"
fi

# 画像ファイルの復元
if [ -d "$RESTORE_DIR/static/images" ]; then
    echo "🖼️  画像ファイルを復元中..."
    mkdir -p static
    rm -rf static/images
    cp -r "$RESTORE_DIR/static/images" static/images
    echo "✅ 画像ファイル復元完了"
fi

# 一時ファイルの削除
rm -rf "$TEMP_DIR"

echo ""
echo "✅ 復元完了!"
echo "📁 緊急バックアップ: $EMERGENCY_BACKUP"
echo "💡 問題がある場合は、緊急バックアップから再度復元してください"