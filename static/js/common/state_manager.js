/**
 * StateManager - 統一的なクライアント状態管理
 * 各クライアントタイプ（admin, screen, participant）で共通使用する状態管理機能
 */
class StateManager {
    constructor() {
        this.currentState = QuizConstants.EVENT_STATES.WAITING;
        this.currentQuestion = 0;
        this.eventData = null;
        this.participantData = {};
        this.teamData = [];
        this.questionData = null;
        this.listeners = [];
        
        // デバウンス用のタイマー
        this.debounceTimers = new Map();
        
        // 状態変更履歴（デバッグ用）
        this.stateHistory = [];
        
        console.log('[StateManager] Initialized with state:', this.currentState);
    }

    /**
     * 状態変更リスナーを登録
     * @param {Function} listener - state, previousState, data を受け取るコールバック関数
     */
    addStateChangeListener(listener) {
        if (typeof listener === 'function') {
            this.listeners.push(listener);
            console.log('[StateManager] State change listener added');
        }
    }

    /**
     * 状態変更リスナーを削除
     * @param {Function} listener - 削除するリスナー
     */
    removeStateChangeListener(listener) {
        const index = this.listeners.indexOf(listener);
        if (index !== -1) {
            this.listeners.splice(index, 1);
            console.log('[StateManager] State change listener removed');
        }
    }

    /**
     * 現在の状態を取得
     * @returns {string} 現在の状態
     */
    getCurrentState() {
        return this.currentState;
    }

    /**
     * 現在の問題番号を取得
     * @returns {number} 現在の問題番号
     */
    getCurrentQuestion() {
        return this.currentQuestion;
    }

    /**
     * 状態を設定（内部使用）
     * @param {string} newState - 新しい状態
     * @param {Object} data - 状態に関連するデータ
     * @param {boolean} skipValidation - 検証をスキップするかどうか
     */
    setState(newState, data = {}, skipValidation = false) {
        const previousState = this.currentState;
        
        // 状態検証（skipValidationがtrueでない場合）
        if (!skipValidation && !this.isValidTransition(previousState, newState)) {
            console.warn(`[StateManager] Invalid state transition: ${previousState} -> ${newState}`);
            return false;
        }
        
        // 同じ状態への変更は無視（ただし、データ更新は行う）
        if (previousState === newState && Object.keys(data).length === 0) {
            return false;
        }
        
        // 状態を更新
        this.currentState = newState;
        
        // 問題番号の更新
        if (data.question_number !== undefined) {
            this.currentQuestion = data.question_number;
        }
        
        // 関連データの更新
        this.updateStateData(newState, data);
        
        // 状態変更履歴を記録
        this.stateHistory.push({
            timestamp: Date.now(),
            from: previousState,
            to: newState,
            data: { ...data }
        });
        
        // 履歴が100件を超えたら古いものを削除
        if (this.stateHistory.length > 100) {
            this.stateHistory.shift();
        }
        
        console.log(`[StateManager] State changed: ${previousState} -> ${newState}`, data);
        
        // リスナーに通知（デバウンスで500ms以内の重複通知を防ぐ）
        this.debounceNotifyListeners(newState, previousState, data);
        
        return true;
    }

    /**
     * 状態遷移の妥当性をチェック
     * @param {string} from - 遷移元の状態
     * @param {string} to - 遷移先の状態
     * @returns {boolean} 遷移が妥当かどうか
     */
    isValidTransition(from, to) {
        // 基本的な遷移ルール（簡単な実装）
        const validTransitions = {
            [QuizConstants.EVENT_STATES.WAITING]: [
                QuizConstants.EVENT_STATES.STARTED,
                QuizConstants.EVENT_STATES.QUESTION_ACTIVE
            ],
            [QuizConstants.EVENT_STATES.STARTED]: [
                QuizConstants.EVENT_STATES.TITLE_DISPLAY,
                QuizConstants.EVENT_STATES.TEAM_ASSIGNMENT,
                QuizConstants.EVENT_STATES.QUESTION_ACTIVE
            ],
            [QuizConstants.EVENT_STATES.TITLE_DISPLAY]: [
                QuizConstants.EVENT_STATES.TEAM_ASSIGNMENT,
                QuizConstants.EVENT_STATES.QUESTION_ACTIVE
            ],
            [QuizConstants.EVENT_STATES.TEAM_ASSIGNMENT]: [
                QuizConstants.EVENT_STATES.QUESTION_ACTIVE
            ],
            [QuizConstants.EVENT_STATES.QUESTION_ACTIVE]: [
                QuizConstants.EVENT_STATES.COUNTDOWN_ACTIVE,
                QuizConstants.EVENT_STATES.ANSWER_STATS
            ],
            [QuizConstants.EVENT_STATES.COUNTDOWN_ACTIVE]: [
                QuizConstants.EVENT_STATES.ANSWER_STATS
            ],
            [QuizConstants.EVENT_STATES.ANSWER_STATS]: [
                QuizConstants.EVENT_STATES.ANSWER_REVEAL
            ],
            [QuizConstants.EVENT_STATES.ANSWER_REVEAL]: [
                QuizConstants.EVENT_STATES.QUESTION_ACTIVE,
                QuizConstants.EVENT_STATES.RESULTS
            ],
            [QuizConstants.EVENT_STATES.RESULTS]: [
                QuizConstants.EVENT_STATES.CELEBRATION
            ],
            [QuizConstants.EVENT_STATES.CELEBRATION]: [
                QuizConstants.EVENT_STATES.FINISHED
            ]
        };
        
        // 同じ状態への遷移は常に許可
        if (from === to) {
            return true;
        }
        
        // 定義されていない状態からの遷移は許可しない
        if (!validTransitions[from]) {
            return false;
        }
        
        return validTransitions[from].includes(to);
    }

    /**
     * 状態に関連するデータを更新
     * @param {string} state - 状態
     * @param {Object} data - データ
     */
    updateStateData(state, data) {
        // イベントデータの更新
        if (data.event) {
            this.eventData = { ...this.eventData, ...data.event };
        }
        
        // 問題データの更新
        if (data.question) {
            this.questionData = data.question;
        }
        
        // チームデータの更新
        if (data.teams) {
            this.teamData = data.teams;
        }
        
        // 参加者データの更新
        if (data.users || data.participants) {
            const users = data.users || data.participants;
            if (Array.isArray(users)) {
                users.forEach(user => {
                    this.participantData[user.id || user.user_id] = user;
                });
            }
        }
        
        // 個別参加者データの更新
        if (data.user) {
            this.participantData[data.user.id || data.user.user_id] = data.user;
        }
    }

    /**
     * デバウンス付きリスナー通知
     * @param {string} newState - 新しい状態
     * @param {string} previousState - 前の状態
     * @param {Object} data - データ
     */
    debounceNotifyListeners(newState, previousState, data) {
        const key = `${previousState}->${newState}`;
        
        // 既存のタイマーをクリア
        if (this.debounceTimers.has(key)) {
            clearTimeout(this.debounceTimers.get(key));
        }
        
        // 新しいタイマーを設定
        const timerId = setTimeout(() => {
            this.notifyListeners(newState, previousState, data);
            this.debounceTimers.delete(key);
        }, 100); // 100msのデバウンス
        
        this.debounceTimers.set(key, timerId);
    }

    /**
     * 状態変更をリスナーに通知
     * @param {string} newState - 新しい状態
     * @param {string} previousState - 前の状態
     * @param {Object} data - データ
     */
    notifyListeners(newState, previousState, data) {
        const stateChangeEvent = {
            state: newState,
            previousState: previousState,
            data: data,
            timestamp: Date.now(),
            currentQuestion: this.currentQuestion,
            eventData: this.eventData,
            questionData: this.questionData,
            teamData: this.teamData,
            participantData: this.participantData
        };
        
        this.listeners.forEach(listener => {
            try {
                listener(stateChangeEvent);
            } catch (error) {
                console.error('[StateManager] Error in state change listener:', error);
            }
        });
    }

    /**
     * WebSocketメッセージから状態を更新
     * @param {string} messageType - メッセージタイプ
     * @param {Object} messageData - メッセージデータ
     */
    handleWebSocketMessage(messageType, messageData) {
        console.log(`[StateManager] Handling WebSocket message: ${messageType}`, messageData);
        
        switch (messageType) {
            case QuizConstants.MESSAGE_TYPES.EVENT_STARTED:
                this.setState(QuizConstants.EVENT_STATES.STARTED, {
                    event: messageData.event,
                    title: messageData.title
                });
                break;
                
            case QuizConstants.MESSAGE_TYPES.TITLE_DISPLAY:
                this.setState(QuizConstants.EVENT_STATES.TITLE_DISPLAY, {
                    title: messageData.title
                });
                break;
                
            case QuizConstants.MESSAGE_TYPES.TEAM_ASSIGNMENT:
                this.setState(QuizConstants.EVENT_STATES.TEAM_ASSIGNMENT, {
                    teams: messageData.teams
                });
                break;
                
            case QuizConstants.MESSAGE_TYPES.QUESTION_START:
                this.setState(QuizConstants.EVENT_STATES.QUESTION_ACTIVE, {
                    question: messageData.question,
                    question_number: messageData.question_number,
                    total_questions: messageData.total_questions
                });
                break;
                
            case QuizConstants.MESSAGE_TYPES.COUNTDOWN:
                this.setState(QuizConstants.EVENT_STATES.COUNTDOWN_ACTIVE, {
                    seconds_left: messageData.seconds_left
                });
                break;
                
            case QuizConstants.MESSAGE_TYPES.QUESTION_END:
                // カウントダウン終了後は回答状況表示へ
                this.setState(QuizConstants.EVENT_STATES.ANSWER_STATS, messageData);
                break;
                
            case QuizConstants.MESSAGE_TYPES.ANSWER_STATS:
                this.setState(QuizConstants.EVENT_STATES.ANSWER_STATS, messageData);
                break;
                
            case QuizConstants.MESSAGE_TYPES.ANSWER_REVEAL:
                this.setState(QuizConstants.EVENT_STATES.ANSWER_REVEAL, {
                    question: messageData.question,
                    correct_index: messageData.correct_index
                });
                break;
                
            case QuizConstants.MESSAGE_TYPES.FINAL_RESULTS:
                this.setState(QuizConstants.EVENT_STATES.RESULTS, {
                    results: messageData.results,
                    teams: messageData.teams,
                    team_mode: messageData.team_mode
                });
                break;
                
            case QuizConstants.MESSAGE_TYPES.CELEBRATION:
                this.setState(QuizConstants.EVENT_STATES.CELEBRATION, messageData);
                break;
                
            case QuizConstants.MESSAGE_TYPES.STATE_CHANGED:
                // 直接的な状態変更メッセージ
                if (messageData.new_state) {
                    this.setState(messageData.new_state, messageData, true); // 検証をスキップ
                }
                break;
                
            case QuizConstants.MESSAGE_TYPES.USER_JOINED:
                this.updateStateData(this.currentState, {
                    user: messageData.user
                });
                break;
                
            default:
                console.log(`[StateManager] Unhandled message type: ${messageType}`);
        }
    }

    /**
     * 状態データをリセット
     */
    reset() {
        this.currentState = QuizConstants.EVENT_STATES.WAITING;
        this.currentQuestion = 0;
        this.eventData = null;
        this.participantData = {};
        this.teamData = [];
        this.questionData = null;
        this.stateHistory = [];
        
        // 全てのデバウンスタイマーをクリア
        this.debounceTimers.forEach(timerId => clearTimeout(timerId));
        this.debounceTimers.clear();
        
        console.log('[StateManager] State reset to initial values');
    }

    /**
     * 現在の状態情報を取得（デバッグ用）
     * @returns {Object} 状態情報
     */
    getStateInfo() {
        return {
            currentState: this.currentState,
            currentQuestion: this.currentQuestion,
            eventData: this.eventData,
            participantCount: Object.keys(this.participantData).length,
            teamCount: this.teamData.length,
            hasQuestionData: !!this.questionData,
            historyCount: this.stateHistory.length,
            listenerCount: this.listeners.length,
            pendingDebounces: this.debounceTimers.size
        };
    }

    /**
     * 状態変更履歴を取得（デバッグ用）
     * @param {number} limit - 取得する履歴の数（デフォルト: 10）
     * @returns {Array} 状態変更履歴
     */
    getStateHistory(limit = 10) {
        return this.stateHistory.slice(-limit);
    }
}

// グローバルエクスポート
if (typeof module !== 'undefined' && module.exports) {
    // Node.js環境
    module.exports = StateManager;
} else {
    // ブラウザ環境
    window.StateManager = StateManager;
}