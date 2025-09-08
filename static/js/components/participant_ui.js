/**
 * ParticipantUI - 参加者画面のUI制御コンポーネント
 * スマートフォン向けの参加者UI・フォーム・インタラクションを統一管理
 */
class ParticipantUI {
    constructor() {
        this.elements = {};
        this.currentState = QuizConstants.EVENT_STATES.WAITING;
        this.currentScreen = 'join';
        
        // 参加者データ
        this.sessionId = localStorage.getItem('quiz_session_id');
        this.userInfo = null;
        this.currentQuestion = null;
        this.selectedAnswer = null;
        
        // UI状態
        this.isAnswerSubmitted = false;
        this.isJoined = false;
        
        console.log('[ParticipantUI] Initialized with session:', this.sessionId);
        this.initializeElements();
        this.setupEventListeners();
        
        // セッションがある場合は再参加を試行
        if (this.sessionId) {
            this.attemptRejoining();
        } else {
            this.showScreen('join');
        }
    }

    /**
     * DOM要素を初期化
     */
    initializeElements() {
        this.elements = {
            // メインコンテナ
            mainContainer: document.getElementById('main-container') || document.body,
            
            // 参加画面
            joinScreen: document.getElementById('join-screen'),
            nicknameInput: document.getElementById('nickname-input'),
            joinButton: document.getElementById('join-button'),
            joinError: document.getElementById('join-error'),
            
            // 待機画面
            waitingScreen: document.getElementById('waiting-screen'),
            userInfo: document.getElementById('user-info'),
            waitingMessage: document.getElementById('waiting-message'),
            
            // 問題画面
            questionScreen: document.getElementById('question-screen'),
            questionNumber: document.getElementById('question-number'),
            questionText: document.getElementById('question-text'),
            questionImage: document.getElementById('question-image'),
            choicesContainer: document.getElementById('choices-container'),
            submitButton: document.getElementById('submit-button'),
            
            // 結果画面
            resultScreen: document.getElementById('result-screen'),
            resultMessage: document.getElementById('result-message'),
            scoreDisplay: document.getElementById('score-display'),
            
            // 共通要素
            connectionStatus: document.getElementById('connection-status'),
            resetButton: document.getElementById('reset-button'),
            emojiButtons: document.getElementById('emoji-buttons'),
            
            // 状態表示
            statusBar: document.getElementById('status-bar'),
            currentStateDisplay: document.getElementById('current-state-display')
        };
        
        // 存在しない要素を動的に作成
        this.createMissingElements();
        
        console.log('[ParticipantUI] DOM elements initialized');
    }

    /**
     * 存在しない要素を動的に作成
     */
    createMissingElements() {
        // 参加画面が存在しない場合
        if (!this.elements.joinScreen) {
            this.createJoinScreen();
        }
        
        // 絵文字ボタンが存在しない場合
        if (!this.elements.emojiButtons) {
            this.createEmojiButtons();
        }
        
        // リセットボタンが存在しない場合
        if (!this.elements.resetButton) {
            this.createResetButton();
        }
    }

    /**
     * イベントリスナーを設定
     */
    setupEventListeners() {
        // 参加ボタン
        if (this.elements.joinButton) {
            this.elements.joinButton.addEventListener('click', () => {
                this.handleJoin();
            });
        }
        
        // ニックネーム入力のエンター
        if (this.elements.nicknameInput) {
            this.elements.nicknameInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.handleJoin();
                }
            });
        }
        
        // 回答送信ボタン
        if (this.elements.submitButton) {
            this.elements.submitButton.addEventListener('click', () => {
                this.handleAnswerSubmit();
            });
        }
        
        // リセットボタン
        if (this.elements.resetButton) {
            this.elements.resetButton.addEventListener('click', () => {
                this.handleReset();
            });
        }
        
        console.log('[ParticipantUI] Event listeners set up');
    }

    /**
     * 画面を表示
     * @param {string} screenName - 表示する画面名
     * @param {Object} data - 画面データ
     */
    showScreen(screenName, data = {}) {
        console.log(`[ParticipantUI] Showing screen: ${screenName}`, data);
        
        // 全ての画面を非表示
        this.hideAllScreens();
        
        this.currentScreen = screenName;
        
        switch (screenName) {
            case 'join':
                this.showJoinScreen(data);
                break;
            case 'waiting':
                this.showWaitingScreen(data);
                break;
            case 'question':
                this.showQuestionScreen(data);
                break;
            case 'result':
                this.showResultScreen(data);
                break;
            case 'finished':
                this.showFinishedScreen(data);
                break;
            default:
                console.warn(`[ParticipantUI] Unknown screen: ${screenName}`);
                this.showJoinScreen(data);
        }
    }

    /**
     * 全ての画面を非表示
     */
    hideAllScreens() {
        const screens = [
            'joinScreen', 'waitingScreen', 'questionScreen', 'resultScreen'
        ];
        
        screens.forEach(screenName => {
            const element = this.elements[screenName];
            if (element) {
                element.style.display = 'none';
                element.classList.remove('active', 'fade-in');
            }
        });
    }

    /**
     * 参加画面を表示
     * @param {Object} data - データ
     */
    showJoinScreen(data) {
        if (!this.elements.joinScreen) {
            this.createJoinScreen();
        }
        
        this.elements.joinScreen.style.display = 'flex';
        this.elements.joinScreen.classList.add('active', 'fade-in');
        
        // エラーメッセージをクリア
        this.clearError();
        
        // フォーカスをニックネーム入力に
        if (this.elements.nicknameInput) {
            setTimeout(() => {
                this.elements.nicknameInput.focus();
            }, 100);
        }
    }

    /**
     * 待機画面を表示
     * @param {Object} data - データ
     */
    showWaitingScreen(data) {
        if (!this.elements.waitingScreen) {
            this.createWaitingScreen();
        }
        
        this.elements.waitingScreen.style.display = 'flex';
        this.elements.waitingScreen.classList.add('active', 'fade-in');
        
        // ユーザー情報を表示
        if (this.elements.userInfo && this.userInfo) {
            this.elements.userInfo.innerHTML = `
                <div class="user-card">
                    <div class="user-nickname">${this.userInfo.nickname}</div>
                    <div class="user-score">現在のスコア: ${this.userInfo.score || 0}点</div>
                </div>
            `;
        }
        
        // 状態に応じたメッセージ
        if (this.elements.waitingMessage) {
            const message = this.getWaitingMessage(data.state || this.currentState);
            this.elements.waitingMessage.textContent = message;
        }
    }

    /**
     * 問題画面を表示
     * @param {Object} data - データ
     */
    showQuestionScreen(data) {
        if (!this.elements.questionScreen) {
            this.createQuestionScreen();
        }
        
        const question = data.question;
        const questionNumber = data.question_number || 1;
        const totalQuestions = data.total_questions || 5;
        
        if (!question) {
            console.warn('[ParticipantUI] No question data');
            return;
        }
        
        this.currentQuestion = question;
        this.selectedAnswer = null;
        this.isAnswerSubmitted = false;
        
        // 問題情報表示
        if (this.elements.questionNumber) {
            this.elements.questionNumber.textContent = `問題 ${questionNumber} / ${totalQuestions}`;
        }
        
        if (this.elements.questionText) {
            this.elements.questionText.textContent = question.text;
        }
        
        // 問題画像
        if (this.elements.questionImage) {
            if (question.image) {
                this.elements.questionImage.innerHTML = `<img src="/images/${question.image}" alt="問題画像" />`;
                this.elements.questionImage.style.display = 'block';
            } else {
                this.elements.questionImage.style.display = 'none';
            }
        }
        
        // 選択肢を生成
        this.createChoices(question.choices);
        
        // 送信ボタンを無効化
        if (this.elements.submitButton) {
            this.elements.submitButton.disabled = true;
        }
        
        this.elements.questionScreen.style.display = 'flex';
        this.elements.questionScreen.classList.add('active', 'fade-in');
    }

    /**
     * 結果画面を表示
     * @param {Object} data - データ
     */
    showResultScreen(data) {
        if (!this.elements.resultScreen) {
            this.createResultScreen();
        }
        
        const isCorrect = data.is_correct;
        const newScore = data.new_score || this.userInfo?.score || 0;
        
        // ユーザースコアを更新
        if (this.userInfo) {
            this.userInfo.score = newScore;
        }
        
        // 結果メッセージ
        if (this.elements.resultMessage) {
            const message = isCorrect ? '🎉 正解です！' : '😅 不正解でした...';
            this.elements.resultMessage.textContent = message;
            this.elements.resultMessage.className = `result-message ${isCorrect ? 'correct' : 'incorrect'}`;
        }
        
        // スコア表示
        if (this.elements.scoreDisplay) {
            this.elements.scoreDisplay.textContent = `現在のスコア: ${newScore}点`;
        }
        
        this.elements.resultScreen.style.display = 'flex';
        this.elements.resultScreen.classList.add('active', 'fade-in');
        
        // 3秒後に待機画面に戻る
        setTimeout(() => {
            this.showWaitingScreen({});
        }, 3000);
    }

    /**
     * 終了画面を表示
     * @param {Object} data - データ
     */
    showFinishedScreen(data) {
        const finishedHTML = `
            <div class="finished-screen">
                <div class="finished-content">
                    <h1>🎉 お疲れ様でした！</h1>
                    <p>クイズ大会は終了しました</p>
                    <div class="final-score">
                        最終スコア: ${this.userInfo?.score || 0}点
                    </div>
                    <div class="thank-you-message">
                        参加していただき、ありがとうございました！
                    </div>
                </div>
            </div>
        `;
        
        this.elements.mainContainer.innerHTML = finishedHTML;
    }

    /**
     * 選択肢を生成
     * @param {Array} choices - 選択肢配列
     */
    createChoices(choices) {
        if (!this.elements.choicesContainer || !choices) return;
        
        this.elements.choicesContainer.innerHTML = '';
        
        choices.forEach((choice, index) => {
            const choiceElement = document.createElement('div');
            choiceElement.className = 'choice-item';
            choiceElement.dataset.index = index;
            
            const choiceLetter = String.fromCharCode(65 + index); // A, B, C, D...
            
            if (choice.endsWith('.png') || choice.endsWith('.jpg') || choice.endsWith('.jpeg')) {
                // 画像選択肢
                choiceElement.innerHTML = `
                    <div class="choice-label">${choiceLetter}</div>
                    <div class="choice-content">
                        <img src="/images/${choice}" alt="選択肢${choiceLetter}" />
                    </div>
                `;
            } else {
                // テキスト選択肢
                choiceElement.innerHTML = `
                    <div class="choice-label">${choiceLetter}</div>
                    <div class="choice-content">
                        <div class="choice-text">${choice}</div>
                    </div>
                `;
            }
            
            // 選択イベント
            choiceElement.addEventListener('click', () => {
                if (this.isAnswerSubmitted) return; // 既に送信済みの場合は無効
                
                this.selectAnswer(index);
            });
            
            this.elements.choicesContainer.appendChild(choiceElement);
        });
    }

    /**
     * 回答を選択
     * @param {number} index - 選択肢のインデックス
     */
    selectAnswer(index) {
        // 全ての選択肢の選択状態をリセット
        const choices = this.elements.choicesContainer.querySelectorAll('.choice-item');
        choices.forEach(choice => choice.classList.remove('selected'));
        
        // 選択した選択肢をハイライト
        const selectedChoice = choices[index];
        if (selectedChoice) {
            selectedChoice.classList.add('selected');
        }
        
        this.selectedAnswer = index;
        
        // 送信ボタンを有効化
        if (this.elements.submitButton) {
            this.elements.submitButton.disabled = false;
        }
        
        console.log(`[ParticipantUI] Answer selected: ${index}`);
    }

    /**
     * 参加処理
     */
    handleJoin() {
        const nickname = this.elements.nicknameInput?.value.trim();
        
        if (!nickname) {
            this.showError('ニックネームを入力してください');
            return;
        }
        
        // ボタンを無効化
        this.setJoinButtonState(false, '参加中...');
        
        console.log(`[ParticipantUI] Attempting to join with nickname: ${nickname}`);
        
        // 参加イベントを発火
        this.emit('join', { nickname });
    }

    /**
     * 再参加の試行
     */
    attemptRejoining() {
        console.log('[ParticipantUI] Attempting to rejoin with existing session');
        
        // 再参加画面を表示
        this.showRejoiningScreen();
        
        // 再参加イベントを発火
        this.emit('rejoin', { sessionId: this.sessionId });
    }

    /**
     * 回答送信処理
     */
    handleAnswerSubmit() {
        if (this.selectedAnswer === null || this.isAnswerSubmitted) {
            return;
        }
        
        if (!this.currentQuestion) {
            this.showError('問題データがありません');
            return;
        }
        
        // ボタンを無効化
        this.setSubmitButtonState(false, '送信中...');
        this.isAnswerSubmitted = true;
        
        // 選択肢を無効化
        const choices = this.elements.choicesContainer?.querySelectorAll('.choice-item');
        if (choices) {
            choices.forEach(choice => choice.classList.add('disabled'));
        }
        
        console.log(`[ParticipantUI] Submitting answer: ${this.selectedAnswer}`);
        
        // 回答送信イベントを発火
        this.emit('submitAnswer', {
            answerIndex: this.selectedAnswer,
            questionNumber: this.currentQuestion.number
        });
    }

    /**
     * リセット処理
     */
    handleReset() {
        const confirmation = '⚠️ これまでの回答状況が消去されます。本当に破棄しますか？\n\n' +
                           'セッションを破棄すると、最初から参加し直すことになります。';
        
        if (!confirm(confirmation)) {
            return;
        }
        
        console.log('[ParticipantUI] Resetting session');
        
        // リセットイベントを発火
        this.emit('reset');
        
        // ローカルデータをクリア
        this.clearLocalData();
        
        // 参加画面に戻る
        this.showJoinScreen({});
    }

    /**
     * 絵文字を送信
     * @param {string} emoji - 絵文字
     */
    sendEmoji(emoji) {
        if (!this.isJoined) return;
        
        console.log(`[ParticipantUI] Sending emoji: ${emoji}`);
        
        // 絵文字送信イベントを発火
        this.emit('sendEmoji', { emoji });
        
        // 視覚的フィードバック
        this.showEmojiSentFeedback(emoji);
    }

    /**
     * 参加成功時の処理
     * @param {Object} userData - ユーザーデータ
     */
    onJoinSuccess(userData) {
        console.log('[ParticipantUI] Join successful:', userData);
        
        this.userInfo = userData.user;
        this.sessionId = userData.session_id;
        this.isJoined = true;
        
        // セッションIDを保存
        localStorage.setItem('quiz_session_id', this.sessionId);
        
        // 待機画面に遷移
        this.showWaitingScreen({});
        
        // 成功メッセージ
        this.showSuccess(`${this.userInfo.nickname}さん、参加しました！`);
    }

    /**
     * 参加失敗時の処理
     * @param {string} error - エラーメッセージ
     */
    onJoinError(error) {
        console.error('[ParticipantUI] Join error:', error);
        
        this.setJoinButtonState(true, '参加する');
        this.showError(error);
    }

    /**
     * 状態更新時の処理
     * @param {Object} stateData - 状態データ
     */
    onStateUpdate(stateData) {
        this.currentState = stateData.state || stateData.new_state || this.currentState;
        
        console.log(`[ParticipantUI] State updated to: ${this.currentState}`);
        
        // 状態に応じた画面制御
        switch (this.currentState) {
            case QuizConstants.EVENT_STATES.QUESTION_ACTIVE:
                if (stateData.question) {
                    this.showQuestionScreen(stateData);
                }
                break;
                
            case QuizConstants.EVENT_STATES.COUNTDOWN_ACTIVE:
                // カウントダウン中は回答を無効化
                this.disableAnswering('⏰ 回答時間終了まであと少し...');
                break;
                
            case QuizConstants.EVENT_STATES.ANSWER_STATS:
            case QuizConstants.EVENT_STATES.ANSWER_REVEAL:
            case QuizConstants.EVENT_STATES.RESULTS:
                if (this.currentScreen === 'question') {
                    this.showWaitingScreen({ state: this.currentState });
                }
                break;
                
            case QuizConstants.EVENT_STATES.FINISHED:
                this.showFinishedScreen({});
                break;
        }
        
        // 状態表示を更新
        this.updateStateDisplay();
    }

    /**
     * 接続状態を更新
     * @param {boolean} connected - 接続状態
     */
    updateConnectionStatus(connected) {
        if (this.elements.connectionStatus) {
            this.elements.connectionStatus.textContent = connected ? '🟢' : '🔴';
            this.elements.connectionStatus.title = connected ? '接続中' : '未接続';
        }
    }

    /**
     * 状態表示を更新
     */
    updateStateDisplay() {
        if (this.elements.currentStateDisplay) {
            const stateLabel = QuizUtils.getStateLabel(this.currentState);
            this.elements.currentStateDisplay.textContent = stateLabel;
        }
    }

    /**
     * 回答を無効化
     * @param {string} message - 表示メッセージ
     */
    disableAnswering(message) {
        // 選択肢を無効化
        const choices = this.elements.choicesContainer?.querySelectorAll('.choice-item');
        if (choices) {
            choices.forEach(choice => {
                choice.classList.add('disabled');
                choice.style.pointerEvents = 'none';
            });
        }
        
        // 送信ボタンを無効化
        if (this.elements.submitButton) {
            this.elements.submitButton.disabled = true;
            this.elements.submitButton.textContent = message || '回答終了';
        }
        
        this.isAnswerSubmitted = true;
    }

    /**
     * 待機メッセージを取得
     * @param {string} state - 現在の状態
     * @returns {string} メッセージ
     */
    getWaitingMessage(state) {
        const messages = {
            [QuizConstants.EVENT_STATES.WAITING]: '🔥 クイズ開始をお待ちください...',
            [QuizConstants.EVENT_STATES.STARTED]: '📺 画面をご覧ください',
            [QuizConstants.EVENT_STATES.TITLE_DISPLAY]: '📺 タイトル画面を表示中',
            [QuizConstants.EVENT_STATES.TEAM_ASSIGNMENT]: '👥 チーム分け中...',
            [QuizConstants.EVENT_STATES.ANSWER_STATS]: '📊 回答状況を確認中...',
            [QuizConstants.EVENT_STATES.ANSWER_REVEAL]: '✅ 正解を発表中！',
            [QuizConstants.EVENT_STATES.RESULTS]: '🏆 結果発表中...',
            [QuizConstants.EVENT_STATES.CELEBRATION]: '🎉 お疲れ様でした！'
        };
        
        return messages[state] || '📺 画面をご覧ください';
    }

    /**
     * ユーティリティメソッド群
     */
    setJoinButtonState(enabled, text) {
        if (this.elements.joinButton) {
            this.elements.joinButton.disabled = !enabled;
            this.elements.joinButton.textContent = text;
        }
    }

    setSubmitButtonState(enabled, text) {
        if (this.elements.submitButton) {
            this.elements.submitButton.disabled = !enabled;
            this.elements.submitButton.textContent = text;
        }
    }

    showError(message) {
        if (this.elements.joinError) {
            this.elements.joinError.textContent = message;
            this.elements.joinError.style.display = 'block';
        } else {
            alert(`エラー: ${message}`);
        }
    }

    clearError() {
        if (this.elements.joinError) {
            this.elements.joinError.textContent = '';
            this.elements.joinError.style.display = 'none';
        }
    }

    showSuccess(message) {
        console.log(`[ParticipantUI] Success: ${message}`);
        // 簡単な成功表示（実装は省略）
    }

    showEmojiSentFeedback(emoji) {
        // 絵文字送信の視覚的フィードバック（実装は省略）
        console.log(`[ParticipantUI] Emoji sent: ${emoji}`);
    }

    clearLocalData() {
        localStorage.removeItem('quiz_session_id');
        this.sessionId = null;
        this.userInfo = null;
        this.isJoined = false;
    }

    /**
     * 画面作成メソッド群
     */
    createJoinScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'join-screen';
        screenDiv.className = 'screen join-screen';
        screenDiv.innerHTML = `
            <div class="join-content">
                <h1>🎉 クイズに参加</h1>
                <div class="join-form">
                    <input type="text" id="nickname-input" placeholder="ニックネームを入力" maxlength="20" />
                    <div id="join-error" class="error-message"></div>
                    <button id="join-button" class="btn btn-primary">参加する</button>
                </div>
            </div>
        `;
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.joinScreen = screenDiv;
        this.elements.nicknameInput = screenDiv.querySelector('#nickname-input');
        this.elements.joinButton = screenDiv.querySelector('#join-button');
        this.elements.joinError = screenDiv.querySelector('#join-error');
        
        // イベントリスナーを再設定
        this.setupEventListeners();
    }

    createWaitingScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'waiting-screen';
        screenDiv.className = 'screen waiting-screen';
        screenDiv.innerHTML = `
            <div class="waiting-content">
                <div id="user-info" class="user-info"></div>
                <div id="waiting-message" class="waiting-message">画面をご覧ください</div>
                <div class="waiting-animation">
                    <div class="pulse"></div>
                </div>
            </div>
        `;
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.waitingScreen = screenDiv;
        this.elements.userInfo = screenDiv.querySelector('#user-info');
        this.elements.waitingMessage = screenDiv.querySelector('#waiting-message');
    }

    createQuestionScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'question-screen';
        screenDiv.className = 'screen question-screen';
        screenDiv.innerHTML = `
            <div class="question-content">
                <div id="question-number" class="question-number"></div>
                <div id="question-text" class="question-text"></div>
                <div id="question-image" class="question-image"></div>
                <div id="choices-container" class="choices-container"></div>
                <button id="submit-button" class="btn btn-primary" disabled>回答を送信</button>
            </div>
        `;
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.questionScreen = screenDiv;
        this.elements.questionNumber = screenDiv.querySelector('#question-number');
        this.elements.questionText = screenDiv.querySelector('#question-text');
        this.elements.questionImage = screenDiv.querySelector('#question-image');
        this.elements.choicesContainer = screenDiv.querySelector('#choices-container');
        this.elements.submitButton = screenDiv.querySelector('#submit-button');
        
        // イベントリスナーを再設定
        this.setupEventListeners();
    }

    createResultScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'result-screen';
        screenDiv.className = 'screen result-screen';
        screenDiv.innerHTML = `
            <div class="result-content">
                <div id="result-message" class="result-message"></div>
                <div id="score-display" class="score-display"></div>
                <div class="result-note">
                    <p>次の問題をお待ちください...</p>
                </div>
            </div>
        `;
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.resultScreen = screenDiv;
        this.elements.resultMessage = screenDiv.querySelector('#result-message');
        this.elements.scoreDisplay = screenDiv.querySelector('#score-display');
    }

    createEmojiButtons() {
        const emojiDiv = document.createElement('div');
        emojiDiv.id = 'emoji-buttons';
        emojiDiv.className = 'emoji-buttons';
        
        const emojis = ['❤️', '👏', '😊', '😮', '🤔', '😅'];
        emojis.forEach(emoji => {
            const button = document.createElement('button');
            button.className = 'emoji-btn';
            button.textContent = emoji;
            button.addEventListener('click', () => this.sendEmoji(emoji));
            emojiDiv.appendChild(button);
        });
        
        this.elements.mainContainer.appendChild(emojiDiv);
        this.elements.emojiButtons = emojiDiv;
    }

    createResetButton() {
        const resetDiv = document.createElement('div');
        resetDiv.className = 'reset-section';
        resetDiv.innerHTML = '<button id="reset-button" class="reset-btn">リセット</button>';
        this.elements.mainContainer.appendChild(resetDiv);
        this.elements.resetButton = resetDiv.querySelector('#reset-button');
        
        // イベントリスナーを再設定
        this.setupEventListeners();
    }

    showRejoiningScreen() {
        const rejoiningHTML = `
            <div class="rejoining-screen">
                <div class="rejoining-content">
                    <h2>🔄 再接続中...</h2>
                    <p>前回のセッションで再参加を試行しています</p>
                    <div class="loading-animation">
                        <div class="spinner"></div>
                    </div>
                </div>
            </div>
        `;
        
        this.elements.mainContainer.innerHTML = rejoiningHTML;
    }

    /**
     * イベント発火・リスナー管理
     */
    emit(eventType, data) {
        const event = new CustomEvent(`participantUI:${eventType}`, { 
            detail: data 
        });
        document.dispatchEvent(event);
    }

    on(eventType, listener) {
        document.addEventListener(`participantUI:${eventType}`, listener);
    }

    off(eventType, listener) {
        document.removeEventListener(`participantUI:${eventType}`, listener);
    }
}

// グローバルエクスポート
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ParticipantUI;
} else {
    window.ParticipantUI = ParticipantUI;
}