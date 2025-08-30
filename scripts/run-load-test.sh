#!/bin/bash

# Quiz100 負荷テスト実行スクリプト

set -e

# 設定
DEFAULT_USERS=100
DEFAULT_DURATION="5m"
SERVER_URL="http://localhost:8080"
LOG_DIR="./load-test-results"

# 色付きメッセージ用の関数
print_message() {
    local type=$1
    local message=$2
    case $type in
        "INFO")  echo -e "\033[0;36mℹ️  $message\033[0m" ;;
        "WARN")  echo -e "\033[0;33m⚠️  $message\033[0m" ;;
        "ERROR") echo -e "\033[0;31m❌ $message\033[0m" ;;
        "SUCCESS") echo -e "\033[0;32m✅ $message\033[0m" ;;
    esac
}

# 使用方法の表示
show_usage() {
    echo "Quiz100 負荷テストスクリプト"
    echo ""
    echo "使用方法:"
    echo "  $0 [オプション]"
    echo ""
    echo "オプション:"
    echo "  -u, --users NUMBER      同時接続ユーザー数 (デフォルト: $DEFAULT_USERS)"
    echo "  -d, --duration TIME     テスト実行時間 (デフォルト: $DEFAULT_DURATION)"
    echo "  -s, --server URL        サーバーURL (デフォルト: $SERVER_URL)"
    echo "  -h, --help             このヘルプを表示"
    echo ""
    echo "例:"
    echo "  $0 -u 50 -d 3m          50人で3分間テスト"
    echo "  $0 --users 200          200人でデフォルト時間テスト"
    echo "  $0 -s http://192.168.1.100:8080  別サーバーでテスト"
}

# パラメータ解析
USERS=$DEFAULT_USERS
DURATION=$DEFAULT_DURATION
SERVER_URL_PARAM=$SERVER_URL

while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--users)
            USERS="$2"
            shift 2
            ;;
        -d|--duration)
            DURATION="$2"
            shift 2
            ;;
        -s|--server)
            SERVER_URL_PARAM="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_message "ERROR" "不明なオプション: $1"
            show_usage
            exit 1
            ;;
    esac
done

# パラメータ検証
if ! [[ "$USERS" =~ ^[0-9]+$ ]] || [ "$USERS" -le 0 ]; then
    print_message "ERROR" "ユーザー数は正の整数で指定してください: $USERS"
    exit 1
fi

if [ "$USERS" -gt 1000 ]; then
    print_message "WARN" "ユーザー数が多すぎます ($USERS人). システムリソースに注意してください."
    read -p "続行しますか? (y/N): " confirm
    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        print_message "INFO" "テストを中止しました"
        exit 0
    fi
fi

# 結果ディレクトリの準備
mkdir -p "$LOG_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_FILE="$LOG_DIR/load_test_${USERS}users_${TIMESTAMP}.log"

print_message "INFO" "Quiz100 負荷テスト開始"
print_message "INFO" "設定:"
echo "  • 同時接続ユーザー数: $USERS人"
echo "  • テスト実行時間: $DURATION"
echo "  • サーバーURL: $SERVER_URL_PARAM"
echo "  • 結果ファイル: $RESULT_FILE"
echo ""

# サーバーの生存確認
print_message "INFO" "サーバーの生存確認中..."
if ! curl -s "$SERVER_URL_PARAM/api/health" >/dev/null 2>&1; then
    print_message "ERROR" "サーバーが応答しません: $SERVER_URL_PARAM"
    print_message "INFO" "Quiz100サーバーが起動していることを確認してください"
    exit 1
fi

print_message "SUCCESS" "サーバー接続確認完了"

# システムリソースの確認
print_message "INFO" "システムリソースを確認中..."

# 利用可能なファイルディスクリプタ数の確認
if command -v ulimit >/dev/null 2>&1; then
    current_limit=$(ulimit -n)
    recommended_limit=$((USERS * 3))
    
    if [ "$current_limit" -lt "$recommended_limit" ]; then
        print_message "WARN" "ファイルディスクリプタ制限が低い可能性があります"
        print_message "WARN" "現在: $current_limit, 推奨: $recommended_limit"
        print_message "INFO" "必要に応じて 'ulimit -n $recommended_limit' を実行してください"
    fi
fi

# メモリ使用量の確認
if command -v free >/dev/null 2>&1; then
    available_mem=$(free -m | awk 'NR==2{print $7}')
    if [ "$available_mem" -lt 512 ]; then
        print_message "WARN" "利用可能メモリが少ない可能性があります: ${available_mem}MB"
    fi
fi

# 負荷テスト用Goプログラムのビルド
print_message "INFO" "負荷テストプログラムをビルド中..."
cd "$(dirname "$0")"

if ! go build -o load-tester load-test.go; then
    print_message "ERROR" "負荷テストプログラムのビルドに失敗しました"
    exit 1
fi

print_message "SUCCESS" "ビルド完了"

# 事前確認
print_message "WARN" "重要: 負荷テストはサーバーに大きな負荷をかけます"
print_message "WARN" "本番環境では実行しないでください"
echo ""
read -p "テストを開始しますか? (y/N): " confirm

if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    print_message "INFO" "テストを中止しました"
    exit 0
fi

print_message "INFO" "負荷テストを開始します..."
echo "ログ出力先: $RESULT_FILE"
echo ""

# テスト開始時刻を記録
echo "=== Quiz100 負荷テスト ===", date >> "$RESULT_FILE"
echo "ユーザー数: $USERS" >> "$RESULT_FILE"
echo "実行時間: $DURATION" >> "$RESULT_FILE"
echo "サーバーURL: $SERVER_URL_PARAM" >> "$RESULT_FILE"
echo "開始時刻: $(date)" >> "$RESULT_FILE"
echo "" >> "$RESULT_FILE"

# 負荷テスト実行
export SERVER_URL="$SERVER_URL_PARAM"
export WEBSOCKET_URL=$(echo "$SERVER_URL_PARAM" | sed 's/http/ws/')

if timeout "${DURATION}" ./load-tester "$USERS" 2>&1 | tee -a "$RESULT_FILE"; then
    print_message "SUCCESS" "負荷テスト完了"
else
    print_message "WARN" "負荷テストが異常終了しました"
fi

# 終了時刻を記録
echo "" >> "$RESULT_FILE"
echo "終了時刻: $(date)" >> "$RESULT_FILE"

# 結果の簡易サマリー
print_message "INFO" "結果サマリー:"
echo "  • 結果ファイル: $RESULT_FILE"

# ログからエラー率を抽出
if grep -q "エラー率:" "$RESULT_FILE"; then
    error_rate=$(grep "エラー率:" "$RESULT_FILE" | tail -1 | grep -o '[0-9.]*%')
    echo "  • エラー率: $error_rate"
fi

# 平均レスポンス時間を抽出
if grep -q "平均:" "$RESULT_FILE"; then
    avg_response=$(grep "平均:" "$RESULT_FILE" | tail -1 | grep -o '[0-9]*ms')
    echo "  • 平均レスポンス時間: $avg_response"
fi

# システムリソースの最終確認
print_message "INFO" "テスト終了後のシステム状態:"
if command -v ps >/dev/null 2>&1; then
    quiz_processes=$(pgrep -c "quiz100" || echo "0")
    echo "  • Quiz100プロセス数: $quiz_processes"
fi

if command -v netstat >/dev/null 2>&1; then
    connections=$(netstat -an | grep ":8080" | wc -l)
    echo "  • ポート8080への接続数: $connections"
fi

# クリーンアップ
rm -f load-tester

print_message "SUCCESS" "負荷テスト完了!"
print_message "INFO" "詳細な結果は $RESULT_FILE をご確認ください"

# 結果ファイルを開くかどうか確認
if command -v less >/dev/null 2>&1; then
    read -p "結果ファイルを表示しますか? (y/N): " show_results
    if [ "$show_results" = "y" ] || [ "$show_results" = "Y" ]; then
        less "$RESULT_FILE"
    fi
fi