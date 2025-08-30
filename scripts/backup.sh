#!/bin/bash

# データベースとログのバックアップスクリプト

set -e

echo "💾 Quiz100 データのバックアップを開始します..."

# バックアップディレクトリの作成
BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "📁 バックアップディレクトリ: $BACKUP_DIR"

# データベースのバックアップ
if [ -f "database/quiz.db" ]; then
    echo "📊 データベースをバックアップ中..."
    cp "database/quiz.db" "$BACKUP_DIR/quiz.db"
    echo "✅ データベースバックアップ完了"
else
    echo "⚠️  データベースファイルが見つかりません: database/quiz.db"
fi

# ログファイルのバックアップ
if [ -d "logs" ] && [ "$(ls -A logs)" ]; then
    echo "📋 ログファイルをバックアップ中..."
    cp -r logs "$BACKUP_DIR/logs"
    echo "✅ ログファイルバックアップ完了"
else
    echo "⚠️  ログファイルが見つかりません"
fi

# 設定ファイルのバックアップ
if [ -f "config/quiz.toml" ]; then
    echo "⚙️  設定ファイルをバックアップ中..."
    mkdir -p "$BACKUP_DIR/config"
    cp "config/quiz.toml" "$BACKUP_DIR/config/quiz.toml"
    echo "✅ 設定ファイルバックアップ完了"
else
    echo "⚠️  設定ファイルが見つかりません: config/quiz.toml"
fi

# 画像ファイルのバックアップ
if [ -d "static/images" ] && [ "$(ls -A static/images)" ]; then
    echo "🖼️  画像ファイルをバックアップ中..."
    mkdir -p "$BACKUP_DIR/static"
    cp -r static/images "$BACKUP_DIR/static/images"
    echo "✅ 画像ファイルバックアップ完了"
else
    echo "ℹ️  画像ファイルはありません"
fi

# バックアップの圧縮
echo "📦 バックアップを圧縮中..."
cd backups
tar -czf "$(basename $BACKUP_DIR).tar.gz" "$(basename $BACKUP_DIR)"
rm -rf "$(basename $BACKUP_DIR)"
cd ..

echo "✅ バックアップ完了: backups/$(basename $BACKUP_DIR).tar.gz"

# 古いバックアップの削除（30日以上前）
echo "🧹 古いバックアップを削除中..."
find backups -name "*.tar.gz" -mtime +30 -delete
echo "✅ 古いバックアップの削除完了"

# バックアップリストの表示
echo ""
echo "📂 現在のバックアップ一覧:"
ls -la backups/*.tar.gz 2>/dev/null || echo "バックアップファイルがありません"