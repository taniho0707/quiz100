/**
 * WebSocketClient - 統一的なWebSocket通信クライアント
 * 接続・再接続・メッセージハンドリング・エラーハンドリングの標準化
 */
class WebSocketClient {
    constructor(options = {}) {
        this.url = options.url || this.buildWebSocketUrl(options.endpoint || '/ws/participant');
        this.reconnectInterval = options.reconnectInterval || 3000; // 3秒
        this.maxReconnectAttempts = options.maxReconnectAttempts || 10;
        this.heartbeatInterval = options.heartbeatInterval || 30000; // 30秒
        
        // 状態管理
        this.ws = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.reconnectTimer = null;
        this.heartbeatTimer = null;
        this.connectionId = null;
        
        // イベントリスナー
        this.messageHandlers = new Map();
        this.connectionListeners = [];
        this.errorListeners = [];
        
        // 送信待ちキュー（接続前のメッセージ用）
        this.sendQueue = [];
        
        console.log('[WebSocketClient] Initialized with URL:', this.url);
        
        // 自動接続（オプション）
        if (options.autoConnect !== false) {
            this.connect();
        }
    }

    /**
     * WebSocket URLを構築
     * @param {string} endpoint - エンドポイント
     * @returns {string} WebSocket URL
     */
    buildWebSocketUrl(endpoint) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        return `${protocol}//${host}${endpoint}`;
    }

    /**
     * WebSocket接続を開始
     */
    connect() {
        if (this.isConnected || (this.ws && this.ws.readyState === WebSocket.CONNECTING)) {
            console.log('[WebSocketClient] Already connected or connecting');
            return;
        }

        console.log('[WebSocketClient] Connecting to:', this.url);

        try {
            this.ws = new WebSocket(this.url);
            this.setupWebSocketEvents();
        } catch (error) {
            console.error('[WebSocketClient] Connection error:', error);
            this.handleError('connection_failed', error);
            this.scheduleReconnect();
        }
    }

    /**
     * WebSocketイベントハンドラーを設定
     */
    setupWebSocketEvents() {
        if (!this.ws) return;

        this.ws.onopen = (event) => {
            console.log('[WebSocketClient] Connection opened');
            this.isConnected = true;
            this.reconnectAttempts = 0;
            this.clearReconnectTimer();
            
            // 接続IDを生成
            this.connectionId = `conn_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
            
            // ハートビート開始
            this.startHeartbeat();
            
            // 送信待ちキューを処理
            this.processSendQueue();
            
            // 接続リスナーに通知
            this.notifyConnectionListeners('connected', event);
        };

        this.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.handleMessage(data);
            } catch (error) {
                console.error('[WebSocketClient] Message parse error:', error);
                this.handleError('message_parse_error', error, event.data);
            }
        };

        this.ws.onclose = (event) => {
            console.log('[WebSocketClient] Connection closed:', event.code, event.reason);
            this.isConnected = false;
            this.stopHeartbeat();
            
            // 接続リスナーに通知
            this.notifyConnectionListeners('disconnected', event);
            
            // 意図的な切断でなければ再接続を試行
            if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
                this.scheduleReconnect();
            }
        };

        this.ws.onerror = (event) => {
            console.error('[WebSocketClient] WebSocket error:', event);
            this.handleError('websocket_error', event);
        };
    }

    /**
     * メッセージを処理
     * @param {Object} data - 受信データ
     */
    handleMessage(data) {
        const messageType = data.type;
        const messageData = data.data || data;
        
        console.log(`[WebSocketClient] Received message: ${messageType}`, messageData);
        
        // ハートビート応答の処理
        if (messageType === 'heartbeat_response' || messageType === 'pong') {
            return;
        }
        
        // 登録されたハンドラーを実行
        if (this.messageHandlers.has(messageType)) {
            const handlers = this.messageHandlers.get(messageType);
            handlers.forEach(handler => {
                try {
                    handler(messageData, data);
                } catch (error) {
                    console.error(`[WebSocketClient] Error in message handler for ${messageType}:`, error);
                    this.handleError('handler_error', error, messageType, messageData);
                }
            });
        } else {
            console.log(`[WebSocketClient] No handler registered for message type: ${messageType}`);
        }
    }

    /**
     * メッセージハンドラーを登録
     * @param {string} messageType - メッセージタイプ
     * @param {Function} handler - ハンドラー関数
     */
    on(messageType, handler) {
        if (typeof handler !== 'function') {
            throw new Error('Handler must be a function');
        }
        
        if (!this.messageHandlers.has(messageType)) {
            this.messageHandlers.set(messageType, []);
        }
        
        this.messageHandlers.get(messageType).push(handler);
        console.log(`[WebSocketClient] Registered handler for: ${messageType}`);
    }

    /**
     * メッセージハンドラーを削除
     * @param {string} messageType - メッセージタイプ
     * @param {Function} handler - ハンドラー関数
     */
    off(messageType, handler) {
        if (!this.messageHandlers.has(messageType)) return;
        
        const handlers = this.messageHandlers.get(messageType);
        const index = handlers.indexOf(handler);
        
        if (index !== -1) {
            handlers.splice(index, 1);
            console.log(`[WebSocketClient] Removed handler for: ${messageType}`);
            
            // ハンドラーがなくなったらMapからも削除
            if (handlers.length === 0) {
                this.messageHandlers.delete(messageType);
            }
        }
    }

    /**
     * 接続状態変更リスナーを登録
     * @param {Function} listener - リスナー関数
     */
    onConnectionChange(listener) {
        if (typeof listener === 'function') {
            this.connectionListeners.push(listener);
        }
    }

    /**
     * エラーリスナーを登録
     * @param {Function} listener - リスナー関数
     */
    onError(listener) {
        if (typeof listener === 'function') {
            this.errorListeners.push(listener);
        }
    }

    /**
     * メッセージを送信
     * @param {string|Object} message - 送信メッセージ
     */
    send(message) {
        const messageData = typeof message === 'string' ? message : JSON.stringify(message);
        
        if (this.isConnected && this.ws.readyState === WebSocket.OPEN) {
            try {
                this.ws.send(messageData);
                console.log('[WebSocketClient] Message sent:', message);
            } catch (error) {
                console.error('[WebSocketClient] Send error:', error);
                this.handleError('send_error', error, message);
            }
        } else {
            // 未接続の場合はキューに追加
            console.log('[WebSocketClient] Queuing message (not connected):', message);
            this.sendQueue.push(messageData);
            
            // キューが満杯になったら古いものから削除
            if (this.sendQueue.length > 100) {
                this.sendQueue.shift();
            }
        }
    }

    /**
     * 送信待ちキューを処理
     */
    processSendQueue() {
        if (this.sendQueue.length === 0) return;
        
        console.log(`[WebSocketClient] Processing send queue (${this.sendQueue.length} messages)`);
        
        while (this.sendQueue.length > 0 && this.isConnected) {
            const message = this.sendQueue.shift();
            try {
                this.ws.send(message);
            } catch (error) {
                console.error('[WebSocketClient] Queue processing error:', error);
                this.handleError('queue_processing_error', error, message);
                break;
            }
        }
    }

    /**
     * 接続を切断
     */
    disconnect() {
        console.log('[WebSocketClient] Manually disconnecting');
        
        this.clearReconnectTimer();
        this.stopHeartbeat();
        
        if (this.ws) {
            // 意図的な切断をマーク
            this.ws.wasClean = true;
            this.ws.close(1000, 'Manual disconnect');
        }
        
        this.isConnected = false;
        this.ws = null;
    }

    /**
     * 再接続をスケジュール
     */
    scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('[WebSocketClient] Max reconnect attempts reached');
            this.handleError('max_reconnect_attempts', new Error('Maximum reconnect attempts exceeded'));
            return;
        }

        this.clearReconnectTimer();
        
        this.reconnectAttempts++;
        const delay = Math.min(this.reconnectInterval * this.reconnectAttempts, 30000); // 最大30秒
        
        console.log(`[WebSocketClient] Scheduling reconnect attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`);
        
        this.reconnectTimer = setTimeout(() => {
            this.connect();
        }, delay);
    }

    /**
     * 再接続タイマーをクリア
     */
    clearReconnectTimer() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
    }

    /**
     * ハートビートを開始
     */
    startHeartbeat() {
        this.stopHeartbeat();
        
        this.heartbeatTimer = setInterval(() => {
            if (this.isConnected) {
                this.send({
                    type: 'heartbeat',
                    timestamp: Date.now(),
                    connectionId: this.connectionId
                });
            }
        }, this.heartbeatInterval);
    }

    /**
     * ハートビートを停止
     */
    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }

    /**
     * 接続状態変更を通知
     * @param {string} type - 変更タイプ
     * @param {Event} event - イベント
     */
    notifyConnectionListeners(type, event) {
        const connectionEvent = {
            type: type,
            connected: this.isConnected,
            reconnectAttempts: this.reconnectAttempts,
            connectionId: this.connectionId,
            timestamp: Date.now(),
            originalEvent: event
        };
        
        this.connectionListeners.forEach(listener => {
            try {
                listener(connectionEvent);
            } catch (error) {
                console.error('[WebSocketClient] Error in connection listener:', error);
            }
        });
    }

    /**
     * エラーを処理
     * @param {string} type - エラータイプ
     * @param {Error|Event} error - エラーオブジェクト
     * @param {...any} details - 詳細情報
     */
    handleError(type, error, ...details) {
        const errorEvent = {
            type: type,
            error: error,
            details: details,
            connected: this.isConnected,
            reconnectAttempts: this.reconnectAttempts,
            timestamp: Date.now()
        };
        
        console.error(`[WebSocketClient] Error (${type}):`, error, ...details);
        
        this.errorListeners.forEach(listener => {
            try {
                listener(errorEvent);
            } catch (listenerError) {
                console.error('[WebSocketClient] Error in error listener:', listenerError);
            }
        });
    }

    /**
     * 接続状態を取得
     * @returns {boolean} 接続状態
     */
    isConnectedState() {
        return this.isConnected;
    }

    /**
     * 接続統計を取得
     * @returns {Object} 接続統計
     */
    getStats() {
        return {
            connected: this.isConnected,
            url: this.url,
            reconnectAttempts: this.reconnectAttempts,
            maxReconnectAttempts: this.maxReconnectAttempts,
            connectionId: this.connectionId,
            sendQueueLength: this.sendQueue.length,
            registeredHandlers: Array.from(this.messageHandlers.keys()),
            connectionListeners: this.connectionListeners.length,
            errorListeners: this.errorListeners.length,
            hasHeartbeat: !!this.heartbeatTimer,
            readyState: this.ws ? this.ws.readyState : null
        };
    }

    /**
     * デバッグ情報を出力
     */
    debug() {
        console.log('[WebSocketClient] Debug Info:', this.getStats());
    }
}

// グローバルエクスポート
if (typeof module !== 'undefined' && module.exports) {
    // Node.js環境
    module.exports = WebSocketClient;
} else {
    // ブラウザ環境
    window.WebSocketClient = WebSocketClient;
}