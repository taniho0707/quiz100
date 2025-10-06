class QuizAdmin {
    constructor() {
        this.ws = null;
        this.currentEvent = null;
        this.participants = new Map();
        this.teams = new Map();
        this.currentQuestion = null;
        this.answers = new Map();
        this.teamMode = false;
        this.pingResults = new Map(); // Store ping results per user
        this.syncStatus = new Map(); // Store sync status for users
        this.sortMode = 'name'; // 'name', 'score', 'ping'

        this.initializeElements();
        this.setupEventListeners();

        this.connectWebSocket();
        this.loadStatus();
        this.loadAvailableStates();

        // Start periodic sync status monitoring
        this.startSyncStatusMonitoring();
    }

    initializeElements() {
        this.elements = {
            // 接続状況
            connectionStatus: document.getElementById('connection-status'),
            connectionText: document.getElementById('connection-text'),
            connectionStatusDisplay: document.getElementById('connection-status-display'),
            
            // 制御ボタン
            startEventBtn: document.getElementById('btn-start-event'),
            showTitleBtn: document.getElementById('btn-show-title'),
            assignTeamsBtn: document.getElementById('btn-assign-teams'),
            nextQuestionBtn: document.getElementById('btn-next-question'),
            countdownAlertBtn: document.getElementById('btn-countdown-alert'),
            showAnswerStatsBtn: document.getElementById('btn-show-answer-stats'),
            revealAnswerBtn: document.getElementById('btn-reveal-answer'),
            showResultsBtn: document.getElementById('btn-show-results'),
            celebrationBtn: document.getElementById('btn-celebration'),
            
            // イベント状況
            // eventStatus: document.getElementById('event-status'),
            currentQuestion: document.getElementById('current-question'),
            participantCount: document.getElementById('participant-count'),
            participantCountDisplay: document.getElementById('participant-count-display'),
            
            // 参加者・チーム表示
            participantsList: document.getElementById('participants-list'),
            teamsContainer: document.getElementById('teams-container'),
            teamsList: document.getElementById('teams-list'),
            
            // 問題・回答表示
            questionDisplay: document.getElementById('question-display'),
            answersDisplay: document.getElementById('answers-display'),
            
            // デバッグ
            jumpStateSelect: document.getElementById('jump-state-select'),
            jumpQuestionInput: document.getElementById('jump-question-input'),
            jumpStateBtn: document.getElementById('jump-state-btn'),
            
            // ログ表示
            logContainer: document.getElementById('log-container'),
            logList: document.getElementById('log-list'),

            // 並び替えボタン
            sortByNameBtn: document.getElementById('sort-by-name'),
            sortByScoreBtn: document.getElementById('sort-by-score'),
            sortByPingBtn: document.getElementById('sort-by-ping')
        };
    }

    setupEventListeners() {
        // アクションボタンのイベントリスナー
        this.elements.startEventBtn?.addEventListener('click', () => this.executeAction('start_event'));
        this.elements.showTitleBtn?.addEventListener('click', () => this.executeAction('show_title'));
        this.elements.assignTeamsBtn?.addEventListener('click', () => this.executeAction('assign_teams'));
        this.elements.nextQuestionBtn?.addEventListener('click', () => this.executeAction('next_question'));
        this.elements.countdownAlertBtn?.addEventListener('click', () => this.executeAction('countdown_alert'));
        this.elements.showAnswerStatsBtn?.addEventListener('click', () => this.executeAction('show_answer_stats'));
        this.elements.revealAnswerBtn?.addEventListener('click', () => this.executeAction('reveal_answer'));
        this.elements.showResultsBtn?.addEventListener('click', () => this.executeAction('show_results'));
        this.elements.celebrationBtn?.addEventListener('click', () => this.executeAction('celebration'));
        
        // デバッグ ステートジャンプ
        this.elements.jumpStateBtn?.addEventListener('click', () => this.handleStateJump());

        // 並び替えボタン
        this.elements.sortByNameBtn?.addEventListener('click', () => this.setSortMode('name'));
        this.elements.sortByScoreBtn?.addEventListener('click', () => this.setSortMode('score'));
        this.elements.sortByPingBtn?.addEventListener('click', () => this.setSortMode('ping'));

        // 参加者リストの動的ボタンイベント（イベントデリゲーション）
        this.elements.participantsList?.addEventListener('click', (e) => {
            if (e.target.classList.contains('participant-button-sync')) {
                const userID = parseInt(e.target.getAttribute('data-user-id'));
                if (userID) {
                    this.requestClientSync(userID);
                }
            }
        });
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/admin`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('Admin WebSocket connected');
            this.updateConnectionStatus(true);
            this.addLog('管理者WebSocket接続しました', 'success');
        };
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        };
        
        this.ws.onclose = () => {
            console.log('Admin WebSocket disconnected');
            this.updateConnectionStatus(false);
            this.addLog('WebSocket接続が切断されました', 'warning');
            setTimeout(() => this.connectWebSocket(), 3000);
        };
        
        this.ws.onerror = (error) => {
            console.error('Admin WebSocket error:', error);
            this.updateConnectionStatus(false);
            this.addLog('WebSocket接続エラーが発生しました', 'error');
        };
    }

    handleWebSocketMessage(message) {
        console.log('Received message:', message);

        switch (message.type) {
            case 'initial_sync':
                this.handleInitialSync(message.data);
                break;

            case 'user_joined':
                this.handleUserJoined(message.data);
                break;

            case 'user_left':
                this.handleUserLeft(message.data);

            case 'answer_received':
                this.handleAnswerReceived(message.data);
                break;

            case 'event_started':
                this.handleEventStarted(message.data);
                break;

            case 'question_start':
                this.handleQuestionStart(message.data);
                break;

            case 'question_end':
                break;

            case 'answer_stats':
                break;

            case 'answer_reveal':
                break;

            case 'team_assignment':
                this.handleTeamAssignment(message.data);
                break;

            case 'final_results':
                this.handleFinalResults(message.data);
                break;

            case 'team_member_added':
                this.handleTeamMemberAdded(message.data);
                break;

            case 'emoji':
                this.handleEmojiReceived(message.data);
                break;

            case 'ping_result':
                this.handlePingResult(message.data);
                break;

            default:
                console.log('Unknown message type:', message.type);
        }
    }

    handleInitialSync(data) {
        console.log('Initial sync received:', data);

        if (!data) {
            console.warn('No sync data received');
            return;
        }

        // Update current question if available
        if (data.question_number !== undefined) {
            this.elements.currentQuestion.textContent = data.question_number;
        }

        // Update question display if question data is available
        if (data.question) {
            this.currentQuestion = data.question;
            this.updateQuestionDisplay();
        }

        // Reload full status (participants, teams, etc.)
        this.loadStatus();

        // Reload available actions to update button states
        this.loadAvailableActions();

        this.addLog('サーバー状態と同期しました', 'success');
    }

    handleUserJoined(data) {
        if (data.teamname) {
            this.addLog(`${data.nickname} が参加しました (${data.teamname}に配置)`, 'info');
        } else {
            this.addLog(`${data.nickname} が参加しました`, 'info');
        }
        this.loadStatus();
    }

    handleUserLeft(data) {
        this.addLog(`${data.nickname} が退出しました`, 'info');
        this.loadStatus();
    }

    handleAnswerReceived(data) {
        this.answers.set(data.user_id, {
            nickname: data.nickname,
            answer: data.answer
        });
        
        this.addLog(`${data.nickname} が回答しました`, 'info');
        this.updateAnswersDisplay();
        this.loadStatus();
    }

    handleEventStarted(data) {
        this.currentEvent = data.event;
        this.updateEventStatus();
        this.addLog(`イベント「${data.title}」が開始されました`, 'success');
    }

    handleQuestionStart(data) {
        this.currentQuestion = data;
        this.answers.clear();
        this.updateQuestionDisplay();
        this.updateAnswersDisplay();
        this.addLog(`問題 ${data.question_number} を開始しました`, 'info');
    }

    handleTeamAssignment(data) {
        this.teams.clear();
        data.teams.forEach(team => {
            this.teams.set(team.id, team);
        });
        this.updateTeamsDisplay();
        this.addLog(`チーム分けが完了しました (${data.teams.length}チーム)`, 'success');
    }

    handleFinalResults(data) {
        if (data.team_mode && data.teams) {
            this.displayFinalTeamResults(data.teams);
        }
        this.displayFinalResults(data.results);
    }

    handleEmojiReceived(data) {
        return;
    }

    async startEvent() {
        this.elements.startEventBtn.disabled = true;
        
        try {
            const response = await fetch('/api/admin/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            const data = await response.json();
            
            if (response.ok) {
                this.currentEvent = data.event;
                this.updateEventStatus();
                this.addLog('イベントを開始しました', 'success');
            } else {
                throw new Error(data.error || 'Failed to start event');
            }
        } catch (error) {
            console.error('Error starting event:', error);
            alert('イベント開始に失敗しました: ' + error.message);
            this.addLog(`イベント開始エラー: ${error.message}`, 'error');
        } finally {
            this.elements.startEventBtn.disabled = false;
        }
    }

    async nextQuestion() {
        this.elements.nextQuestionBtn.disabled = true;
        
        try {
            const response = await fetch('/api/admin/next', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            const data = await response.json();
            
            if (response.ok) {
                this.currentQuestion = data;
                this.answers.clear();
                this.updateQuestionDisplay();
                this.updateAnswersDisplay();
                this.addLog(`問題 ${data.question_number} を開始しました`, 'info');
                
                if (this.currentEvent) {
                    this.currentEvent.question_number = data.question_number;
                    this.updateEventStatus();
                }
            } else {
                throw new Error(data.error || 'Failed to start next question');
            }
        } catch (error) {
            console.error('Error starting next question:', error);
            alert('次の問題の開始に失敗しました: ' + error.message);
            this.addLog(`次の問題開始エラー: ${error.message}`, 'error');
        } finally {
            this.elements.nextQuestionBtn.disabled = false;
        }
    }

    async sendAlert() {
        this.elements.countdownAlertBtn.disabled = true;
        
        try {
            const response = await fetch('/api/admin/alert', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            const data = await response.json();
            
            if (response.ok) {
                this.addLog('5秒アラートを送信しました', 'warning');
            } else {
                throw new Error(data.error || 'Failed to send alert');
            }
        } catch (error) {
            console.error('Error sending alert:', error);
            alert('アラート送信に失敗しました: ' + error.message);
            this.addLog(`アラート送信エラー: ${error.message}`, 'error');
        } finally {
            setTimeout(() => {
                this.elements.countdownAlertBtn.disabled = false;
            }, 2000); // 2秒間無効にして連打を防ぐ
        }
    }

    async stopEvent() {
        this.elements.stopBtn.disabled = true;
        
        try {
            const response = await fetch('/api/admin/stop', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            const data = await response.json();
            
            if (response.ok) {
                this.currentEvent = data.event;
                this.updateEventStatus();
                this.addLog('イベントを終了しました', 'success');
                this.displayFinalResults(data.results);
            } else {
                throw new Error(data.error || 'Failed to stop event');
            }
        } catch (error) {
            console.error('Error stopping event:', error);
            alert('イベント終了に失敗しました: ' + error.message);
            this.addLog(`イベント終了エラー: ${error.message}`, 'error');
        } finally {
            this.elements.stopBtn.disabled = false;
        }
    }

    async createTeams() {
        this.elements.createTeamsBtn.disabled = true;
        
        try {
            const response = await fetch('/api/admin/teams', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            const data = await response.json();
            
            if (response.ok) {
                this.teams.clear();
                data.teams.forEach(team => {
                    this.teams.set(team.id, team);
                });
                this.updateTeamsDisplay();
                this.addLog(`チーム分けが完了しました (${data.teams.length}チーム)`, 'success');
            } else {
                throw new Error(data.error || 'Failed to create teams');
            }
        } catch (error) {
            console.error('Error creating teams:', error);
            alert('チーム分けに失敗しました: ' + error.message);
            this.addLog(`チーム分けエラー: ${error.message}`, 'error');
        } finally {
            this.elements.createTeamsBtn.disabled = false;
        }
    }

    async loadStatus() {
        try {
            const response = await fetch('/api/status');
            const data = await response.json();
            
            if (response.ok) {
                this.updateParticipants(data.users || []);
                
                if (data.teams) {
                    this.teams.clear();
                    data.teams.forEach(team => {
                        this.teams.set(team.id, team);
                    });
                    this.updateTeamsDisplay();
                }
                
                if (data.event) {
                    this.currentEvent = data.event;
                    this.updateEventStatus();
                }
                
                if (data.config) {
                    this.teamMode = data.config.team_mode || false;
                    this.updateTeamModeDisplay();
                    // Total questions count - can be displayed in console or elsewhere if needed
                    console.log(`Total questions: ${data.config.questions?.length || 0}`);
                }
                
                // Load available actions and update button states
                this.loadAvailableActions();
            }
        } catch (error) {
            console.error('Error loading status:', error);
        }
    }

    async loadAvailableActions() {
        try {
            const response = await fetch('/api/admin/actions');
            const data = await response.json();
            
            if (response.ok) {
                this.updateButtonStates(data.available_actions || []);
            }
        } catch (error) {
            console.error('Error loading available actions:', error);
        }
    }

    updateButtonStates(availableActions) {
        const buttonMap = {
            'show_title': this.elements.showTitleBtn,
            'assign_teams': this.elements.assignTeamsBtn,
            'next_question': this.elements.nextQuestionBtn,
            'countdown_alert': this.elements.countdownAlertBtn,
            'show_answer_stats': this.elements.showAnswerStatsBtn,
            'reveal_answer': this.elements.revealAnswerBtn,
            'show_results': this.elements.showResultsBtn,
            'celebration': this.elements.celebrationBtn
        };

        // すべてのボタンを無効にし、利用可能なもののみ有効にする
        Object.values(buttonMap).forEach(button => {
            if (button) button.disabled = true;
        });

        availableActions.forEach(action => {
            const button = buttonMap[action];
            if (button) {
                button.disabled = false;
            }
        });

        // チーム分けボタンの表示制御
        if (this.elements.assignTeamsBtn) {
            this.elements.assignTeamsBtn.style.display = 
                availableActions.includes('assign_teams') ? 'inline-block' : 'none';
        }
    }

    async executeAction(action) {
        try {
            const response = await fetch('/api/admin/action', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ action: action })
            });

            const data = await response.json();
            
            if (response.ok) {
                this.addLog(data.message || `${action} を実行しました`, 'success');
                
                // 状態更新とボタン制御を再読み込み
                this.loadAvailableActions();
                
                // 必要に応じて特定の更新処理
                switch(action) {
                    case 'next_question':
                        this.currentQuestion = data.question;
                        this.answers.clear();
                        this.updateQuestionDisplay();
                        this.updateAnswersDisplay();
                        break;
                    case 'assign_teams':
                        if (data.teams) {
                            data.teams.forEach(team => {
                                this.teams.set(team.id, team);
                            });
                            this.updateTeamsDisplay();
                        }
                        break;
                    case 'show_results':
                        if (data.results) {
                            this.displayFinalResults(data.results);
                        }
                        break;
                }
                
                if (data.event) {
                    this.currentEvent = data.event;
                    this.updateEventStatus();
                }
            } else {
                throw new Error(data.error || `Failed to execute ${action}`);
            }
        } catch (error) {
            console.error(`Error executing ${action}:`, error);
            alert(`${action}の実行に失敗しました: ${error.message}`);
            this.addLog(`${action}実行エラー: ${error.message}`, 'error');
        }
    }

    updateEventStatus() {
        if (!this.currentEvent) {
            // this.elements.eventStatus.textContent = '待機中';
            this.elements.currentQuestion.textContent = '-';
            this.elements.startEventBtn.disabled = false;
            this.elements.nextQuestionBtn.disabled = true;
            this.elements.countdownAlertBtn.disabled = true;
            return;
        }

        // this.elements.eventStatus.textContent = 
        //     this.currentEvent.status === 'started' ? '進行中' : 
        //     this.currentEvent.status === 'finished' ? '終了' : '待機中';
        
        this.elements.currentQuestion.textContent = this.currentEvent.question_number || 0;
        
        this.elements.startEventBtn.disabled = this.currentEvent.status === 'started';
        this.elements.nextQuestionBtn.disabled = this.currentEvent.status !== 'started';
        this.elements.countdownAlertBtn.disabled = this.currentEvent.status !== 'started';
    }

    updateParticipants(users) {
        this.participants.clear();
        users.forEach(user => this.participants.set(user.id, user));

        this.elements.participantCount.textContent = users.length;
        this.updateParticipantsDisplay();
    }

    setSortMode(mode) {
        this.sortMode = mode;
        this.updateParticipantsDisplay();
    }

    updateParticipantsDisplay() {
        this.elements.participantsList.innerHTML = '';

        // 参加者を並び替え
        const sortedParticipants = this.getSortedParticipants();

        sortedParticipants.forEach(user => {
            const item = document.createElement('div');
            item.className = 'participant-item';

            // Get ping data for this user
            const pingData = this.pingResults.get(user.nickname);
            let pingStatusHtml = '';

            if (pingData) {
                const isStale = pingData.result > 5000; // Data older than 5 seconds

                if (isStale) {
                    pingStatusHtml = '<span class="ping-status stale">--ms</span>';
                } else {
                    const latencyText = pingData.result < 0 ? 'timeout' : `${pingData.result}ms`;
                    let pingDataStatus = '';
                    if (pingData.result < 0) {
                        pingDataStatus = 'unknown';
                    } else if (pingData.result < 300) {
                        pingDataStatus = 'good';
                    } else if (pingData.result < 1000) {
                        pingDataStatus = 'slow';
                    } else {
                        pingDataStatus = 'bad';
                    }
                    pingStatusHtml = `<span class="ping-status ${pingDataStatus}">${latencyText}</span>`;
                }
            } else {
                pingStatusHtml = '<span class="ping-status unknown">--ms</span>';
            }

            // // Get sync status for this user
            // const syncData = this.syncStatus.get(user.nickname);
            // let syncStatusHtml = '<span class="sync-status unknown">🔄 未知</span>';

            // if (syncData) {
            //     const syncAge = Date.now() - new Date(syncData.last_sync_time).getTime();
            //     const isOutdated = syncAge > 60000; // More than 1 minute old

            //     switch (syncData.status) {
            //         case 'synchronized':
            //             syncStatusHtml = `<span class="sync-status synchronized">✓ 同期済</span>`;
            //             break;
            //         case 'outdated':
            //             syncStatusHtml = `<span class="sync-status outdated">⚠ 古い</span>`;
            //             break;
            //         case 'uninitialized':
            //             syncStatusHtml = `<span class="sync-status uninitialized">🔄 未初期化</span>`;
            //             break;
            //         default:
            //             syncStatusHtml = `<span class="sync-status unknown">❓ 不明</span>`;
            //     }

            //     if (isOutdated && syncData.status === 'synchronized') {
            //         syncStatusHtml = `<span class="sync-status outdated">⏰ 期限切れ</span>`;
            //     }
            // }

            item.innerHTML = `
                <div class="participant-info">
                    <div class="connection-status ${user.connected ? '' : 'disconnected'}"></div>
                    <span class="participant-name">${user.nickname}</span>
                    ${pingStatusHtml}
                </div>
                <span class="participant-score">${user.score}点</span>
                <div class="participant-control">
                    <button class="smallbtn participant-button-delete">削除</div>
                    <button class="smallbtn participant-button-resetscore">成績消去</div>
                    <button class="smallbtn participant-button-changename">名前変更</div>
                </div>
            `;
            //     <!-- ${syncStatusHtml} -->
            // <!-- <button class="smallbtn participant-button-sync" data-user-id="${user.id}">🔄 同期</div> -->

            this.elements.participantsList.appendChild(item);
        });
    }

    getSortedParticipants() {
        const participants = Array.from(this.participants.values());

        switch (this.sortMode) {
            case 'name':
                return participants.sort((a, b) => a.nickname.localeCompare(b.nickname));

            case 'score':
                return participants.sort((a, b) => b.score - a.score);

            case 'ping':
                return participants.sort((a, b) => {
                    const pingA = this.pingResults.get(a.id);
                    const pingB = this.pingResults.get(b.id);

                    // データがない場合は最下位
                    if (!pingA && !pingB) return 0;
                    if (!pingA) return 1;
                    if (!pingB) return -1;

                    // 古いデータは最下位
                    const ageA = Date.now() - pingA.timestamp;
                    const ageB = Date.now() - pingB.timestamp;
                    const isStaleA = ageA > 15000;
                    const isStaleB = ageB > 15000;

                    if (isStaleA && !isStaleB) return 1;
                    if (!isStaleA && isStaleB) return -1;
                    if (isStaleA && isStaleB) return 0;

                    // 良い通信 → 普通 → 悪い通信 の順序
                    const statusOrder = { 'good': 0, 'slow': 1, 'bad': 2 };
                    const orderA = statusOrder[pingA.status] || 3;
                    const orderB = statusOrder[pingB.status] || 3;

                    if (orderA !== orderB) {
                        return orderA - orderB;
                    }

                    // 同じステータスの場合はレイテンシが低い順
                    return pingA.latency - pingB.latency;
                });

            default:
                return participants;
        }
    }

    updateQuestionDisplay() {
        if (!this.currentQuestion) {
            this.elements.questionDisplay.innerHTML = '<p>問題が開始されていません</p>';
            return;
        }

        const question = this.currentQuestion.question;
        let html = `
            <h4>問題 ${this.currentQuestion.question_number}</h4>
            <p><strong>${question.text}</strong></p>
        `;
        
        if (question.image) {
            html += `<img src="/images/${question.image}" alt="問題画像" class="question-image">`;
        }
        
        html += '<div class="choices-list">';
        question.choices.forEach((choice, index) => {
            // Convert 0-based index to 1-based for comparison with 1-based correct answer
            const isCorrect = (index + 1) === question.correct;
            html += `
                <div class="choice-item ${isCorrect ? 'correct' : ''}">
                    ${String.fromCharCode(65 + index)}. ${choice}
                    ${isCorrect ? ' ✓' : ''}
                </div>
            `;
        });
        html += '</div>';
        
        this.elements.questionDisplay.innerHTML = html;
    }

    updateAnswersDisplay() {
        if (!this.currentQuestion) {
            this.elements.answersDisplay.innerHTML = '<p>問題が開始されていません</p>';
            return;
        }

        const totalParticipants = this.participants.size;
        const answeredCount = this.answers.size;
        const correctCount = Array.from(this.answers.values())
            .filter(answer => answer.is_correct).length;
        
        let html = `
            <div class="answer-stats">
                <div class="answer-stat">
                    <div class="answer-stat-number">${answeredCount}</div>
                    <div class="answer-stat-label">回答済み</div>
                </div>
                <div class="answer-stat">
                    <div class="answer-stat-number">${totalParticipants - answeredCount}</div>
                    <div class="answer-stat-label">未回答</div>
                </div>
                <div class="answer-stat">
                    <div class="answer-stat-number">${correctCount}</div>
                    <div class="answer-stat-label">正解者</div>
                </div>
                <div class="answer-stat">
                    <div class="answer-stat-number">${Math.round(correctCount / Math.max(answeredCount, 1) * 100)}%</div>
                    <div class="answer-stat-label">正解率</div>
                </div>
            </div>
        `;
        
        this.elements.answersDisplay.innerHTML = html;
    }

    displayFinalResults(results) {
        results.sort((a, b) => b.score - a.score);
        
        this.elements.answersDisplay.innerHTML = `
            <h4>🏆 最終結果</h4>
            ${results.map((user, index) => `
                <div class="participant-item">
                    <div class="participant-info">
                        <span class="rank">${index + 1}位</span>
                        <span class="participant-name">${user.nickname}</span>
                    </div>
                    <span class="participant-score">${user.score}点</span>
                </div>
            `).join('')}
        `;
    }

    updateTeamsDisplay() {
        if (!this.elements.teamsList) return;
        
        this.elements.teamsList.innerHTML = '';
        
        if (this.teams.size === 0) {
            this.elements.teamsList.innerHTML = '<p>チームが作成されていません</p>';
            return;
        }

        const teamsArray = Array.from(this.teams.values())
            .sort((a, b) => b.score - a.score);

        teamsArray.forEach((team, index) => {
            const teamElement = document.createElement('div');
            teamElement.className = 'team-item';
            
            const membersHtml = team.members ? team.members.map(member => `
                <div class="team-member">
                    <div class="connection-status ${member.connected ? '' : 'disconnected'}"></div>
                    <span>${member.nickname}</span>
                    <span class="member-score">${member.score}点</span>
                </div>
            `).join('') : '';

            teamElement.innerHTML = `
                <div class="team-header">
                    <div class="team-info">
                        <span class="team-rank">${index + 1}位</span>
                        <span class="team-name">${team.name}</span>
                        <span class="team-member-count">(${team.members ? team.members.length : 0}人)</span>
                    </div>
                    <span class="team-score">${team.score}点</span>
                </div>
                <div class="team-members">
                    ${membersHtml}
                </div>
            `;
            
            this.elements.teamsList.appendChild(teamElement);
        });
    }

    updateTeamModeDisplay() {
        if (this.elements.teamsContainer) {
            this.elements.teamsContainer.style.display = this.teamMode ? 'block' : 'none';
        }
        if (this.elements.createTeamsBtn) {
            this.elements.createTeamsBtn.style.display = this.teamMode ? 'inline-block' : 'none';
        }
    }

    displayFinalTeamResults(teams) {
        if (!this.elements.teamsList) return;
        
        teams.sort((a, b) => b.score - a.score);
        
        this.elements.teamsList.innerHTML = `
            <h4>🏆 チーム最終結果</h4>
            ${teams.map((team, index) => `
                <div class="team-item final-result">
                    <div class="team-header">
                        <div class="team-info">
                            <span class="team-rank">${index + 1}位</span>
                            <span class="team-name">${team.name}</span>
                        </div>
                        <span class="team-score">${team.score}点</span>
                    </div>
                    <div class="team-members">
                        ${team.members ? team.members.map(member => `
                            <div class="team-member">
                                <span>${member.nickname}</span>
                                <span class="member-score">${member.score}点</span>
                            </div>
                        `).join('') : ''}
                    </div>
                </div>
            `).join('')}
        `;
    }
    
    handleTeamMemberAdded(data) {
        this.teams.set(data.team_id, data.team); // FIXME: データ構造直す
        this.updateTeamsDisplay();
        this.addLog(`${data.nickname} が ${data.team_id} に自動配置されました`, 'success');
    }

    handlePingResult(data) {
        // Store ping result for the user
        this.pingResults.set(data.nickname, {
            result: data.result
        });

        // Update the participants display to show ping status
        this.updateParticipantsDisplay();

        // Optional: Add to log for debugging
        if (data.status === 'bad' || data.result > 5000) {
            this.addLog(`${data.nickname} の通信が不安定です (${data.latency === 0 ? 'タイムアウト' : data.latency + 'ms'})`, 'warning');
        }
    }

    addLog(message, type = 'info') {
        const timestamp = new Date().toLocaleTimeString();
        const logEntry = document.createElement('div');
        logEntry.className = 'log-entry';
        // logEntry.style.cssText = 'padding: 5px; margin-bottom: 3px; border-left: 3px solid #ccc; background: white; border-radius: 3px;';
        
        // タイプ別の色分け
        const typeColors = {
            'info': '#2196F3',
            'success': '#4CAF50', 
            'warning': '#FF9800',
            'error': '#F44336'
        };
        const color = typeColors[type] || '#2196F3';
        logEntry.style.borderLeftColor = color;
        
        logEntry.innerHTML = `
            <span class="log-timestamp" style="color: #666;">${timestamp}</span>
            <span class="log-type-${type}">${message}</span>
        `;
        
        // 新しいログを先頭に追加（column-reverseで実際は下に追加されるが、表示上は上に見える）
        this.elements.logList.appendChild(logEntry);
        
        // 一番上（最新ログ）まで強制スクロール
        this.elements.logContainer.scrollTop = 0;
        
        // ログが多すぎる場合は古いものを削除（最大1000件）
        const maxLogs = 1000;
        const logEntries = this.elements.logList.children;
        while (logEntries.length > maxLogs) {
            this.elements.logList.removeChild(logEntries[0]);
        }
    }

    updateConnectionStatus(connected) {
        if (connected) {
            this.elements.connectionStatus.className = 'status-indicator connected';
            this.elements.connectionText.textContent = '接続済み';
        } else {
            this.elements.connectionStatus.className = 'status-indicator disconnected';
            this.elements.connectionText.textContent = '接続中...';
        }
    }

    async loadAvailableStates() {
        try {
            const response = await fetch('/api/admin/available-states');
            if (response.ok) {
                const data = await response.json();
                this.populateStateOptions(data.available_states);
                this.updateCurrentStateDisplay(data.current_state);
            }
        } catch (error) {
            console.error('Failed to load available states:', error);
        }
    }

    populateStateOptions(states) {
        const select = this.elements.jumpStateSelect;
        select.innerHTML = '<option value="">状態を選択...</option>';
        
        states.forEach(state => {
            const option = document.createElement('option');
            option.value = state.value;
            option.textContent = state.label;
            select.appendChild(option);
        });
    }

    updateCurrentStateDisplay(currentState) {
        // Update the event status display using shared constants
        // this.elements.eventStatus.textContent = QuizUtils.StateUtils.getStateLabel(currentState);
    }

    // Sync Status Management

    startSyncStatusMonitoring() {
        // // Load sync status every 10 seconds
        // this.loadSyncStatus();
        // this.syncStatusInterval = setInterval(() => {
        //     this.loadSyncStatus();
        // }, 10000);
    }

    async loadSyncStatus() {
        try {
            const response = await fetch('/api/ws/sync-status');
            if (response.ok) {
                const data = await response.json();
                this.updateSyncStatusDisplay(data);
            }
        } catch (error) {
            console.error('Failed to load sync status:', error);
        }
    }

    updateSyncStatusDisplay(syncReport) {
        // Update sync status for each client
        if (syncReport.client_details) {
            this.syncStatus.clear();
            syncReport.client_details.forEach(client => {
                this.syncStatus.set(client.user_id, client);
            });

            // Update participants display to show sync status
            this.updateParticipantsDisplay();
        }

        // Log sync rate if needed
        if (syncReport.sync_rate !== undefined && syncReport.sync_rate < 80) {
            this.addLog(`同期率が低下しています: ${syncReport.sync_rate.toFixed(1)}% (${syncReport.synchronized}/${syncReport.total_clients})`, 'warning');
        }
    }

    async requestClientSync(userID) {
        try {
            const response = await fetch('/api/ws/sync-client', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    user_id: userID,
                    sync_type: 'manual'
                })
            });

            if (response.ok) {
                const result = await response.json();
                this.addLog(`ユーザー${userID}の手動同期を要求しました`, 'info');

                // Reload sync status after a short delay
                setTimeout(() => this.loadSyncStatus(), 1000);
            } else {
                const error = await response.json();
                this.addLog(`同期要求エラー: ${error.error}`, 'error');
            }
        } catch (error) {
            console.error('Client sync request failed:', error);
            this.addLog('同期要求に失敗しました', 'error');
        }
    }

    async syncAllClients() {
        try {
            const response = await fetch('/api/ws/sync-all', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (response.ok) {
                const result = await response.json();
                this.addLog('全参加者の手動同期を要求しました', 'info');

                // Reload sync status after a short delay
                setTimeout(() => this.loadSyncStatus(), 1000);
            } else {
                const error = await response.json();
                this.addLog(`全体同期要求エラー: ${error.error}`, 'error');
            }
        } catch (error) {
            console.error('Sync all clients failed:', error);
            this.addLog('全体同期要求に失敗しました', 'error');
        }
    }

    async handleStateJump() {
        const selectedState = this.elements.jumpStateSelect.value;
        const questionNumber = this.elements.jumpQuestionInput.value;

        if (!selectedState) {
            alert('ステートを選択してください');
            return;
        }

        // Show confirmation dialog
        const confirmMessage = questionNumber ? 
            `ステート '${selectedState}' (問題${questionNumber}) にジャンプしますか？\n\n⚠️ これはデバッグ機能です。予期しない動作が発生する可能性があります。` :
            `ステート '${selectedState}' にジャンプしますか？\n\n⚠️ これはデバッグ機能です。予期しない動作が発生する可能性があります。`;
        
        if (!confirm(confirmMessage)) {
            return;
        }

        try {
            const requestBody = { state: selectedState };
            if (questionNumber && questionNumber.trim() !== '') {
                requestBody.question_number = parseInt(questionNumber);
            }

            const response = await fetch('/api/admin/jump-state', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody)
            });

            if (response.ok) {
                const result = await response.json();
                this.addLog(result.message, 'warning');
                this.updateCurrentStateDisplay(result.new_state);
                this.loadAvailableActions(); // Refresh button states
                
                // Clear form
                this.elements.jumpStateSelect.value = '';
                this.elements.jumpQuestionInput.value = '';
            } else {
                const error = await response.json();
                this.addLog(`ステートジャンプエラー: ${error.error}`, 'error');
            }
        } catch (error) {
            console.error('State jump failed:', error);
            this.addLog('ステートジャンプに失敗しました', 'error');
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new QuizAdmin();
});