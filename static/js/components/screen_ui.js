/**
 * ScreenUI - スクリーン表示画面のUI制御コンポーネント
 * 大画面表示用の画面遷移・アニメーション・エフェクトを統一管理
 */
class ScreenUI {
    constructor() {
        this.elements = {};
        this.currentState = QuizConstants.EVENT_STATES.WAITING;
        this.currentScreen = 'waiting';
        
        // アニメーション管理
        this.animations = new Map();
        this.timers = new Map();
        
        // 状態データ
        this.eventData = null;
        this.questionData = null;
        this.teamData = [];
        this.participantData = {};
        
        console.log('[ScreenUI] Initialized');
        this.initializeElements();
        this.showScreen('waiting');
    }

    /**
     * DOM要素を初期化
     */
    initializeElements() {
        this.elements = {
            // メインコンテナ
            mainContainer: document.getElementById('main-container'),
            
            // 各画面要素
            waitingScreen: document.getElementById('waiting-screen'),
            titleScreen: document.getElementById('title-screen'),
            teamScreen: document.getElementById('team-screen'),
            questionScreen: document.getElementById('question-screen'),
            answerStatsScreen: document.getElementById('answer-stats-screen'),
            answerRevealScreen: document.getElementById('answer-reveal-screen'),
            resultsScreen: document.getElementById('results-screen'),
            celebrationScreen: document.getElementById('celebration-screen'),
            
            // 共通要素
            eventTitle: document.getElementById('event-title'),
            participantCount: document.getElementById('participant-count'),
            connectionStatus: document.getElementById('connection-status'),
            
            // 問題表示関連
            questionContainer: document.getElementById('question-container'),
            questionNumber: document.getElementById('question-number'),
            questionText: document.getElementById('question-text'),
            questionImage: document.getElementById('question-image'),
            choicesContainer: document.getElementById('choices-container'),
            
            // カウントダウン表示
            countdownDisplay: document.getElementById('countdown-number-display'),
            countdownBorder: document.getElementById('countdown-border'),
            timeUpDisplay: document.getElementById('time-up-display'),
            
            // 絵文字表示
            emojiContainer: document.getElementById('emoji-container'),
            
            // 結果表示
            resultsContainer: document.getElementById('results-container'),
            
            // クラッカー演出
            crackerContainer: document.getElementById('cracker-container')
        };
        
        // 動的要素が存在しない場合は作成
        this.createMissingElements();
        
        console.log('[ScreenUI] DOM elements initialized');
    }

    /**
     * 存在しない要素を動的に作成
     */
    createMissingElements() {
        // メインコンテナが存在しない場合
        if (!this.elements.mainContainer) {
            this.elements.mainContainer = document.body;
        }
        
        // カウントダウン表示要素を作成
        if (!this.elements.countdownDisplay) {
            const countdownDiv = document.createElement('div');
            countdownDiv.id = 'countdown-number-display';
            countdownDiv.className = 'countdown-number-display';
            countdownDiv.style.display = 'none';
            document.body.appendChild(countdownDiv);
            this.elements.countdownDisplay = countdownDiv;
        }
        
        // 時間切れ表示要素を作成
        if (!this.elements.timeUpDisplay) {
            const timeUpDiv = document.createElement('div');
            timeUpDiv.id = 'time-up-display';
            timeUpDiv.className = 'time-up-display';
            timeUpDiv.style.display = 'none';
            timeUpDiv.innerHTML = '<div class="time-up-text">終了！</div>';
            document.body.appendChild(timeUpDiv);
            this.elements.timeUpDisplay = timeUpDiv;
        }
        
        // カウントダウン枠要素を作成
        if (!this.elements.countdownBorder) {
            const borderDiv = document.createElement('div');
            borderDiv.id = 'countdown-border';
            borderDiv.className = 'countdown-border';
            borderDiv.style.display = 'none';
            document.body.appendChild(borderDiv);
            this.elements.countdownBorder = borderDiv;
        }
        
        // 絵文字コンテナを作成
        if (!this.elements.emojiContainer) {
            const emojiDiv = document.createElement('div');
            emojiDiv.id = 'emoji-container';
            emojiDiv.className = 'emoji-container';
            document.body.appendChild(emojiDiv);
            this.elements.emojiContainer = emojiDiv;
        }
    }

    /**
     * 画面を表示
     * @param {string} screenName - 表示する画面名
     * @param {Object} data - 画面データ
     */
    showScreen(screenName, data = {}) {
        console.log(`[ScreenUI] Showing screen: ${screenName}`, data);
        
        // 現在の画面を非表示
        this.hideAllScreens();
        
        // 新しい画面を表示
        this.currentScreen = screenName;
        
        switch (screenName) {
            case 'waiting':
                this.showWaitingScreen(data);
                break;
            case 'title':
                this.showTitleScreen(data);
                break;
            case 'teams':
                this.showTeamScreen(data);
                break;
            case 'question':
                this.showQuestionScreen(data);
                break;
            case 'answer_stats':
                this.showAnswerStatsScreen(data);
                break;
            case 'answer_reveal':
                this.showAnswerRevealScreen(data);
                break;
            case 'results':
                this.showResultsScreen(data);
                break;
            case 'celebration':
                this.showCelebrationScreen(data);
                break;
            default:
                console.warn(`[ScreenUI] Unknown screen: ${screenName}`);
                this.showWaitingScreen(data);
        }
    }

    /**
     * 全ての画面を非表示
     */
    hideAllScreens() {
        const screens = [
            'waitingScreen', 'titleScreen', 'teamScreen', 'questionScreen',
            'answerStatsScreen', 'answerRevealScreen', 'resultsScreen', 'celebrationScreen'
        ];
        
        screens.forEach(screenName => {
            const element = this.elements[screenName];
            if (element) {
                element.style.display = 'none';
                element.classList.remove('active', 'fade-in', 'slide-in');
            }
        });
        
        // 特殊要素も非表示
        this.hideCountdown();
        this.hideCracker();
    }

    /**
     * 待機画面を表示
     * @param {Object} data - データ
     */
    showWaitingScreen(data) {
        if (!this.elements.waitingScreen) {
            // 動的に待機画面を作成
            this.createWaitingScreen();
        }
        
        this.elements.waitingScreen.style.display = 'flex';
        this.elements.waitingScreen.classList.add('active', 'fade-in');
        
        // 参加者数更新
        if (data.participantCount !== undefined) {
            this.updateParticipantCount(data.participantCount);
        }
    }

    /**
     * タイトル画面を表示
     * @param {Object} data - データ
     */
    showTitleScreen(data) {
        if (!this.elements.titleScreen) {
            this.createTitleScreen();
        }
        
        const title = data.title || data.event_title || 'クイズ大会';
        
        this.elements.titleScreen.innerHTML = `
            <div class="title-content">
                <h1 class="main-title glow-effect">${title}</h1>
                <p class="welcome-message">🎉 クイズ大会へようこそ！</p>
            </div>
        `;
        
        this.elements.titleScreen.style.display = 'flex';
        this.elements.titleScreen.classList.add('active', 'fade-in');
    }

    /**
     * チーム発表画面を表示
     * @param {Object} data - データ
     */
    showTeamScreen(data) {
        if (!this.elements.teamScreen) {
            this.createTeamScreen();
        }
        
        const teams = data.teams || [];
        
        let teamHTML = `
            <div class="team-announcement">
                <h2 class="team-title">👥 チーム発表</h2>
                <div class="teams-grid">
        `;
        
        teams.forEach((team, index) => {
            const members = team.members || [];
            const membersList = members.map(member => 
                `<div class="member-card">${member.nickname || member.name}</div>`
            ).join('');
            
            teamHTML += `
                <div class="team-card slide-in" style="animation-delay: ${index * 0.2}s">
                    <div class="team-header">
                        <h3>チーム ${team.name || (index + 1)}</h3>
                        <div class="team-score">スコア: ${team.score || 0}</div>
                    </div>
                    <div class="team-members">
                        ${membersList}
                    </div>
                </div>
            `;
        });
        
        teamHTML += `
                </div>
            </div>
        `;
        
        this.elements.teamScreen.innerHTML = teamHTML;
        this.elements.teamScreen.style.display = 'flex';
        this.elements.teamScreen.classList.add('active');
    }

    /**
     * 問題画面を表示
     * @param {Object} data - データ
     */
    showQuestionScreen(data) {
        if (!this.elements.questionScreen) {
            this.createQuestionScreen();
        }
        
        const question = data.question || this.questionData;
        const questionNumber = data.question_number || data.current_question || 1;
        const totalQuestions = data.total_questions || 5;
        
        if (!question) {
            console.warn('[ScreenUI] No question data available');
            return;
        }
        
        // 問題データを保存
        this.questionData = question;
        
        let questionHTML = `
            <div class="question-content">
                <div class="question-header">
                    <div class="question-number">問題 ${questionNumber} / ${totalQuestions}</div>
                </div>
                <div class="question-text">${question.text}</div>
        `;
        
        // 画像がある場合
        if (question.image) {
            questionHTML += `<div class="question-image">
                <img src="/images/${question.image}" alt="問題画像" />
            </div>`;
        }
        
        // 選択肢表示
        if (question.choices && question.choices.length > 0) {
            questionHTML += '<div class="choices-container">';
            question.choices.forEach((choice, index) => {
                const choiceLetter = String.fromCharCode(65 + index); // A, B, C, D...
                
                if (choice.endsWith('.png') || choice.endsWith('.jpg') || choice.endsWith('.jpeg')) {
                    // 画像選択肢
                    questionHTML += `
                        <div class="choice-item image-choice" data-index="${index}">
                            <div class="choice-label">${choiceLetter}</div>
                            <img src="/images/${choice}" alt="選択肢${choiceLetter}" />
                        </div>
                    `;
                } else {
                    // テキスト選択肢
                    questionHTML += `
                        <div class="choice-item text-choice" data-index="${index}">
                            <div class="choice-label">${choiceLetter}</div>
                            <div class="choice-text">${choice}</div>
                        </div>
                    `;
                }
            });
            questionHTML += '</div>';
        }
        
        questionHTML += `
                <div class="answer-prompt">
                    <p>🤔 答えを選んでスマートフォンで回答してください</p>
                </div>
            </div>
        `;
        
        this.elements.questionScreen.innerHTML = questionHTML;
        this.elements.questionScreen.style.display = 'flex';
        this.elements.questionScreen.classList.add('active', 'fade-in');
    }

    /**
     * 回答状況画面を表示
     * @param {Object} data - データ
     */
    showAnswerStatsScreen(data) {
        if (!this.elements.answerStatsScreen) {
            this.createAnswerStatsScreen();
        }
        
        const totalParticipants = data.total_participants || 0;
        const answeredCount = data.answered_count || 0;
        const correctCount = data.correct_count || 0;
        const correctRate = data.correct_rate || 0;
        
        const statsHTML = `
            <div class="stats-content">
                <h2 class="stats-title">📊 回答状況</h2>
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-number">${totalParticipants}</div>
                        <div class="stat-label">参加者数</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-number">${answeredCount}</div>
                        <div class="stat-label">回答者数</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-number">${correctCount}</div>
                        <div class="stat-label">正解者数</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-number">${Math.round(correctRate)}%</div>
                        <div class="stat-label">正解率</div>
                    </div>
                </div>
                <div class="progress-container">
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${(answeredCount / Math.max(totalParticipants, 1)) * 100}%"></div>
                    </div>
                    <div class="progress-text">回答進捗: ${answeredCount} / ${totalParticipants}</div>
                </div>
            </div>
        `;
        
        this.elements.answerStatsScreen.innerHTML = statsHTML;
        this.elements.answerStatsScreen.style.display = 'flex';
        this.elements.answerStatsScreen.classList.add('active', 'fade-in');
    }

    /**
     * 回答発表画面を表示
     * @param {Object} data - データ
     */
    showAnswerRevealScreen(data) {
        if (!this.elements.answerRevealScreen) {
            this.createAnswerRevealScreen();
        }
        
        const question = data.question || this.questionData;
        const correctIndex = data.correct_index;
        
        if (!question || correctIndex === undefined) {
            console.warn('[ScreenUI] Insufficient data for answer reveal');
            return;
        }
        
        let revealHTML = `
            <div class="reveal-content">
                <h2 class="reveal-title">✅ 正解発表</h2>
                <div class="question-recap">
                    <div class="question-text">${question.text}</div>
                </div>
                <div class="choices-container">
        `;
        
        question.choices.forEach((choice, index) => {
            const choiceLetter = String.fromCharCode(65 + index);
            const isCorrect = index === correctIndex;
            
            revealHTML += `
                <div class="choice-item ${isCorrect ? 'correct' : 'incorrect'}" data-index="${index}">
                    <div class="choice-label">${choiceLetter}</div>
                    <div class="choice-text">${choice}</div>
                    ${isCorrect ? '<div class="correct-indicator">✓ 正解</div>' : ''}
                </div>
            `;
        });
        
        revealHTML += `
                </div>
                <div class="correct-answer">
                    <div class="correct-text pulse-animation">
                        正解は ${String.fromCharCode(65 + correctIndex)} です！
                    </div>
                </div>
            </div>
        `;
        
        this.elements.answerRevealScreen.innerHTML = revealHTML;
        this.elements.answerRevealScreen.style.display = 'flex';
        this.elements.answerRevealScreen.classList.add('active', 'fade-in');
    }

    /**
     * 結果発表画面を表示
     * @param {Object} data - データ
     */
    showResultsScreen(data) {
        if (!this.elements.resultsScreen) {
            this.createResultsScreen();
        }
        
        const results = data.results || [];
        const teams = data.teams || [];
        const teamMode = data.team_mode || false;
        
        let resultsHTML = `
            <div class="results-content">
                <h2 class="results-title">🏆 最終結果発表</h2>
        `;
        
        if (teamMode && teams.length > 0) {
            // チーム戦の結果表示
            const sortedTeams = teams.sort((a, b) => b.score - a.score);
            
            resultsHTML += '<div class="team-results">';
            sortedTeams.forEach((team, index) => {
                const rank = index + 1;
                const medal = rank === 1 ? '🥇' : rank === 2 ? '🥈' : rank === 3 ? '🥉' : '🏅';
                
                resultsHTML += `
                    <div class="result-team slide-in" style="animation-delay: ${index * 0.2}s">
                        <div class="team-rank">${rank}位 ${medal}</div>
                        <div class="team-info">
                            <div class="team-name">チーム ${team.name}</div>
                            <div class="team-score">${team.score}点</div>
                        </div>
                    </div>
                `;
            });
            resultsHTML += '</div>';
        } else {
            // 個人戦の結果表示
            const sortedResults = results.sort((a, b) => b.score - a.score);
            
            resultsHTML += '<div class="individual-results">';
            sortedResults.slice(0, 10).forEach((user, index) => { // 上位10名のみ表示
                const rank = index + 1;
                const medal = rank === 1 ? '🥇' : rank === 2 ? '🥈' : rank === 3 ? '🥉' : '🏅';
                
                resultsHTML += `
                    <div class="result-user slide-in" style="animation-delay: ${index * 0.1}s">
                        <div class="user-rank">${rank}位 ${medal}</div>
                        <div class="user-info">
                            <div class="user-name">${user.nickname}</div>
                            <div class="user-score">${user.score}点</div>
                        </div>
                    </div>
                `;
            });
            resultsHTML += '</div>';
        }
        
        resultsHTML += '</div>';
        
        this.elements.resultsScreen.innerHTML = resultsHTML;
        this.elements.resultsScreen.style.display = 'flex';
        this.elements.resultsScreen.classList.add('active');
    }

    /**
     * お疲れ様画面を表示
     * @param {Object} data - データ
     */
    showCelebrationScreen(data) {
        if (!this.elements.celebrationScreen) {
            this.createCelebrationScreen();
        }
        
        const celebrationHTML = `
            <div class="celebration-content">
                <h1 class="celebration-title glow-effect">🎉 お疲れ様でした！</h1>
                <p class="celebration-message">クイズ大会にご参加いただき、ありがとうございました</p>
                <div class="celebration-emojis">
                    <span class="emoji bounce">🎊</span>
                    <span class="emoji bounce" style="animation-delay: 0.1s">🎉</span>
                    <span class="emoji bounce" style="animation-delay: 0.2s">👏</span>
                    <span class="emoji bounce" style="animation-delay: 0.3s">🏆</span>
                    <span class="emoji bounce" style="animation-delay: 0.4s">✨</span>
                </div>
            </div>
        `;
        
        this.elements.celebrationScreen.innerHTML = celebrationHTML;
        this.elements.celebrationScreen.style.display = 'flex';
        this.elements.celebrationScreen.classList.add('active', 'fade-in');
        
        // 5秒後にクラッカー演出を開始
        setTimeout(() => {
            this.showCracker();
        }, 1000);
    }

    /**
     * カウントダウンを表示
     * @param {number} seconds - 残り秒数
     */
    showCountdown(seconds) {
        if (this.elements.countdownDisplay) {
            this.elements.countdownDisplay.textContent = seconds;
            this.elements.countdownDisplay.style.display = 'flex';
            this.elements.countdownDisplay.classList.add('pulse-animation');
        }
        
        if (this.elements.countdownBorder) {
            this.elements.countdownBorder.style.display = 'block';
        }
    }

    /**
     * 時間切れ表示を表示
     */
    showTimeUp() {
        if (this.elements.timeUpDisplay) {
            this.elements.timeUpDisplay.style.display = 'flex';
            this.elements.timeUpDisplay.classList.add('fade-in');
            
            // 3秒後に非表示
            setTimeout(() => {
                this.hideTimeUp();
            }, 3000);
        }
    }

    /**
     * カウントダウン表示を非表示
     */
    hideCountdown() {
        [this.elements.countdownDisplay, this.elements.countdownBorder, this.elements.timeUpDisplay].forEach(element => {
            if (element) {
                element.style.display = 'none';
                element.classList.remove('pulse-animation', 'fade-in');
            }
        });
    }

    /**
     * 時間切れ表示を非表示
     */
    hideTimeUp() {
        if (this.elements.timeUpDisplay) {
            this.elements.timeUpDisplay.style.display = 'none';
            this.elements.timeUpDisplay.classList.remove('fade-in');
        }
    }

    /**
     * 絵文字リアクションを表示
     * @param {Object} emojiData - 絵文字データ
     */
    showEmoji(emojiData) {
        if (!this.elements.emojiContainer) return;
        
        const emoji = emojiData.emoji || '❤️';
        const nickname = emojiData.nickname || 'Someone';
        
        const emojiElement = document.createElement('div');
        emojiElement.className = 'emoji-reaction';
        emojiElement.innerHTML = `
            <div class="emoji-icon">${emoji}</div>
            <div class="emoji-user">${nickname}</div>
        `;
        
        // ランダムな位置に配置
        const x = Math.random() * (window.innerWidth - 100);
        const y = Math.random() * (window.innerHeight - 100);
        
        emojiElement.style.left = `${x}px`;
        emojiElement.style.top = `${y}px`;
        
        this.elements.emojiContainer.appendChild(emojiElement);
        
        // アニメーション後に削除
        setTimeout(() => {
            if (emojiElement.parentNode) {
                emojiElement.parentNode.removeChild(emojiElement);
            }
        }, 3000);
    }

    /**
     * クラッカー演出を表示
     */
    showCracker() {
        console.log('[ScreenUI] Starting cracker animation');
        
        // 既存のクラッカーをクリア
        this.clearCracker();
        
        // クラッカーコンテナが存在しない場合は作成
        if (!this.elements.crackerContainer) {
            const crackerDiv = document.createElement('div');
            crackerDiv.id = 'cracker-container';
            crackerDiv.className = 'cracker-container';
            document.body.appendChild(crackerDiv);
            this.elements.crackerContainer = crackerDiv;
        }
        
        // 左右のクラッカーアイコンを作成
        const leftCracker = document.createElement('div');
        leftCracker.className = 'cracker-icon left';
        leftCracker.textContent = '🎊';
        
        const rightCracker = document.createElement('div');
        rightCracker.className = 'cracker-icon right';
        rightCracker.textContent = '🎊';
        
        this.elements.crackerContainer.appendChild(leftCracker);
        this.elements.crackerContainer.appendChild(rightCracker);
        
        // 紙吹雪を作成（左側から50個）
        for (let i = 0; i < 50; i++) {
            setTimeout(() => {
                this.createConfetti('left');
            }, i * 50);
        }
        
        // 紙吹雪を作成（右側から50個）
        for (let i = 0; i < 50; i++) {
            setTimeout(() => {
                this.createConfetti('right');
            }, i * 50);
        }
        
        // 5秒後にクラッカーをクリア
        setTimeout(() => {
            this.clearCracker();
        }, 5000);
    }

    /**
     * 紙吹雪を作成
     * @param {string} side - 左右どちら側か ('left' | 'right')
     */
    createConfetti(side) {
        if (!this.elements.crackerContainer) return;
        
        const confetti = document.createElement('div');
        confetti.className = 'confetti';
        
        // ランダムな色
        const colors = ['#ff6b6b', '#4ecdc4', '#45b7d1', '#f9d71c', '#dda0dd', '#98fb98'];
        const color = colors[Math.floor(Math.random() * colors.length)];
        confetti.style.backgroundColor = color;
        
        // 位置を設定
        const startX = side === 'left' ? 50 : window.innerWidth - 50;
        const endX = startX + (Math.random() - 0.5) * 400; // ±200pxの範囲
        
        confetti.style.left = `${startX}px`;
        confetti.style.top = '50px';
        
        // アニメーション設定
        const duration = 2000 + Math.random() * 1000; // 2-3秒
        const rotation = Math.random() * 720; // 0-720度回転
        
        confetti.style.animation = `
            confettiFall ${duration}ms linear forwards,
            confettiRotate ${duration}ms linear infinite
        `;
        
        // CSS変数でアニメーション値を設定
        confetti.style.setProperty('--end-x', `${endX}px`);
        confetti.style.setProperty('--rotation', `${rotation}deg`);
        
        this.elements.crackerContainer.appendChild(confetti);
        
        // アニメーション終了後に削除
        setTimeout(() => {
            if (confetti.parentNode) {
                confetti.parentNode.removeChild(confetti);
            }
        }, duration);
    }

    /**
     * クラッカー演出をクリア
     */
    clearCracker() {
        if (this.elements.crackerContainer) {
            this.elements.crackerContainer.innerHTML = '';
        }
    }

    /**
     * クラッカー演出を非表示
     */
    hideCracker() {
        this.clearCracker();
        if (this.elements.crackerContainer) {
            this.elements.crackerContainer.style.display = 'none';
        }
    }

    /**
     * 参加者数を更新
     * @param {number} count - 参加者数
     */
    updateParticipantCount(count) {
        if (this.elements.participantCount) {
            this.elements.participantCount.textContent = `参加者: ${count}名`;
        }
    }

    /**
     * 接続状態を更新
     * @param {boolean} connected - 接続状態
     */
    updateConnectionStatus(connected) {
        if (this.elements.connectionStatus) {
            this.elements.connectionStatus.textContent = connected ? '🟢 接続中' : '🔴 未接続';
            this.elements.connectionStatus.className = connected ? 'connected' : 'disconnected';
        }
    }

    /**
     * 画面を動的に作成するメソッド群
     */
    createWaitingScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'waiting-screen';
        screenDiv.className = 'screen waiting-screen';
        screenDiv.innerHTML = `
            <div class="waiting-content">
                <h1>📱 参加者を待機中...</h1>
                <p>スマートフォンでQRコードを読み取り、参加してください</p>
                <div class="waiting-animation">
                    <div class="dot"></div>
                    <div class="dot"></div>
                    <div class="dot"></div>
                </div>
            </div>
        `;
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.waitingScreen = screenDiv;
    }

    createTitleScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'title-screen';
        screenDiv.className = 'screen title-screen';
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.titleScreen = screenDiv;
    }

    createTeamScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'team-screen';
        screenDiv.className = 'screen team-screen';
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.teamScreen = screenDiv;
    }

    createQuestionScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'question-screen';
        screenDiv.className = 'screen question-screen';
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.questionScreen = screenDiv;
    }

    createAnswerStatsScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'answer-stats-screen';
        screenDiv.className = 'screen answer-stats-screen';
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.answerStatsScreen = screenDiv;
    }

    createAnswerRevealScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'answer-reveal-screen';
        screenDiv.className = 'screen answer-reveal-screen';
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.answerRevealScreen = screenDiv;
    }

    createResultsScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'results-screen';
        screenDiv.className = 'screen results-screen';
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.resultsScreen = screenDiv;
    }

    createCelebrationScreen() {
        const screenDiv = document.createElement('div');
        screenDiv.id = 'celebration-screen';
        screenDiv.className = 'screen celebration-screen';
        this.elements.mainContainer.appendChild(screenDiv);
        this.elements.celebrationScreen = screenDiv;
    }
}

// グローバルエクスポート
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ScreenUI;
} else {
    window.ScreenUI = ScreenUI;
}