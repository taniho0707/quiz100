#!/bin/bash

# システムヘルスチェックスクリプト

set -e

echo "🔍 Quiz100 システムヘルスチェックを実行中..."

# 設定
HOST="localhost"
PORT="8080"
HEALTH_ENDPOINT="/api/health"
DEBUG_ENDPOINT="/api/admin/debug"

# 色付きメッセージ用の関数
print_status() {
    local status=$1
    local message=$2
    
    case $status in
        "OK")
            echo "✅ $message"
            ;;
        "WARN")
            echo "⚠️  $message"
            ;;
        "ERROR")
            echo "❌ $message"
            ;;
        "INFO")
            echo "ℹ️  $message"
            ;;
    esac
}

# ヘルスチェック API の確認
check_health_api() {
    print_status "INFO" "ヘルスチェックAPI を確認中..."
    
    if response=$(curl -s "http://$HOST:$PORT$HEALTH_ENDPOINT" 2>/dev/null); then
        status=$(echo "$response" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
        
        if [ "$status" = "ok" ]; then
            print_status "OK" "アプリケーションは正常に動作しています"
            
            # メモリ使用量の確認
            memory_alloc=$(echo "$response" | grep -o '"alloc":[0-9]*' | cut -d':' -f2)
            if [ -n "$memory_alloc" ]; then
                print_status "INFO" "メモリ使用量: ${memory_alloc}MB"
            fi
            
            # WebSocket接続数の確認
            participant_count=$(echo "$response" | grep -o '"participant":[0-9]*' | cut -d':' -f2)
            admin_count=$(echo "$response" | grep -o '"admin":[0-9]*' | cut -d':' -f2)
            screen_count=$(echo "$response" | grep -o '"screen":[0-9]*' | cut -d':' -f2)
            
            if [ -n "$participant_count" ]; then
                print_status "INFO" "参加者接続数: $participant_count"
            fi
            if [ -n "$admin_count" ]; then
                print_status "INFO" "管理者接続数: $admin_count"
            fi
            if [ -n "$screen_count" ]; then
                print_status "INFO" "スクリーン接続数: $screen_count"
            fi
            
            return 0
        else
            print_status "ERROR" "アプリケーションステータス異常: $status"
            return 1
        fi
    else
        print_status "ERROR" "ヘルスチェックAPIにアクセスできません (http://$HOST:$PORT$HEALTH_ENDPOINT)"
        return 1
    fi
}

# ポートの確認
check_port() {
    print_status "INFO" "ポート $PORT の確認中..."
    
    if nc -z "$HOST" "$PORT" 2>/dev/null; then
        print_status "OK" "ポート $PORT は開いています"
        return 0
    else
        print_status "ERROR" "ポート $PORT に接続できません"
        return 1
    fi
}

# プロセスの確認
check_process() {
    print_status "INFO" "Quiz100プロセスの確認中..."
    
    if pgrep -f "quiz100" > /dev/null 2>&1; then
        pid=$(pgrep -f "quiz100")
        print_status "OK" "Quiz100プロセスが実行中 (PID: $pid)"
        
        # CPU使用率とメモリ使用量の確認
        if command -v ps >/dev/null 2>&1; then
            cpu_mem=$(ps -p "$pid" -o pid,pcpu,pmem --no-headers 2>/dev/null)
            if [ -n "$cpu_mem" ]; then
                print_status "INFO" "リソース使用量: $cpu_mem (PID CPU% MEM%)"
            fi
        fi
        return 0
    else
        print_status "ERROR" "Quiz100プロセスが見つかりません"
        return 1
    fi
}

# ファイルシステムの確認
check_filesystem() {
    print_status "INFO" "ファイルシステムの確認中..."
    
    # データベースファイル
    if [ -f "database/quiz.db" ]; then
        db_size=$(stat -f%z "database/quiz.db" 2>/dev/null || stat -c%s "database/quiz.db" 2>/dev/null)
        print_status "OK" "データベースファイル存在 (${db_size} bytes)"
    else
        print_status "WARN" "データベースファイルが見つかりません"
    fi
    
    # 設定ファイル
    if [ -f "config/quiz.toml" ]; then
        print_status "OK" "設定ファイル存在"
    else
        print_status "ERROR" "設定ファイルが見つかりません"
    fi
    
    # ログディレクトリ
    if [ -d "logs" ]; then
        log_count=$(find logs -name "*.log" | wc -l)
        print_status "OK" "ログディレクトリ存在 (${log_count} ログファイル)"
    else
        print_status "WARN" "ログディレクトリが見つかりません"
    fi
    
    # ディスク使用量
    if command -v df >/dev/null 2>&1; then
        disk_usage=$(df -h . | tail -1 | awk '{print $5 " used"}')
        print_status "INFO" "ディスク使用量: $disk_usage"
    fi
}

# メイン実行
main() {
    echo "📊 システム情報:"
    echo "   ホスト: $HOST"
    echo "   ポート: $PORT"
    echo "   日時: $(date)"
    echo ""
    
    # 各チェックの実行
    check_port
    echo ""
    
    check_process
    echo ""
    
    check_health_api
    echo ""
    
    check_filesystem
    echo ""
    
    # 総合結果
    if [ $? -eq 0 ]; then
        print_status "OK" "システムヘルスチェック完了"
        echo ""
        echo "🌐 アクセス URL:"
        echo "   管理者: http://$HOST:$PORT/admin"
        echo "   参加者: http://$HOST:$PORT/"
        echo "   スクリーン: http://$HOST:$PORT/show"
        exit 0
    else
        print_status "ERROR" "システムに問題があります"
        exit 1
    fi
}

main "$@"