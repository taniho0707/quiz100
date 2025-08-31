class QuizAdmin {
    constructor() {
        this.ws = null;
        this.currentEvent = null;
        this.participants = new Map();
        this.teams = new Map();
        this.currentQuestion = null;
        this.answers = new Map();
        this.teamMode = false;

        this.initializeElements();
        this.setupEventListeners();
        this.connectWebSocket();
        this.loadStatus();
    }

    initializeElements() {
        this.elements = {
            connectionStatus: document.getElementById('connection-status'),
            connectionText: document.getElementById('connection-text'),
            
            startBtn: document.getElementById('start-btn'),
            nextBtn: document.getElementById('next-btn'),
            alertBtn: document.getElementById('alert-btn'),
            stopBtn: document.getElementById('stop-btn'),
            createTeamsBtn: document.getElementById('create-teams-btn'),
            
            eventStatus: document.getElementById('event-status'),
            currentQuestion: document.getElementById('current-question'),
            totalQuestions: document.getElementById('total-questions'),
            
            participantCount: document.getElementById('participant-count'),
            participantsList: document.getElementById('participants-list'),
            teamsContainer: document.getElementById('teams-container'),
            teamsList: document.getElementById('teams-list'),
            
            currentQuestionDisplay: document.getElementById('current-question-display'),
            answersDisplay: document.getElementById('answers-display'),
            logDisplay: document.getElementById('log-display')
        };
    }

    setupEventListeners() {
        this.elements.startBtn.addEventListener('click', () => this.startEvent());
        this.elements.nextBtn.addEventListener('click', () => this.nextQuestion());
        this.elements.alertBtn.addEventListener('click', () => this.sendAlert());
        this.elements.stopBtn.addEventListener('click', () => this.stopEvent());
        this.elements.createTeamsBtn.addEventListener('click', () => this.createTeams());
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
            case 'user_joined':
                this.handleUserJoined(message.data);
                break;
                
            case 'answer_received':
                this.handleAnswerReceived(message.data);
                break;
                
            case 'event_started':
                this.handleEventStarted(message.data);
                break;
                
            case 'question_start':
                this.handleQuestionStart(message.data);
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
                
            default:
                console.log('Unknown message type:', message.type);
        }
    }

    handleUserJoined(data) {
        if (data.assigned_team) {
            this.addLog(`${data.nickname} が参加しました (${data.assigned_team.name}に配置)`, 'info');
        } else {
            this.addLog(`${data.nickname} が参加しました`, 'info');
        }
        this.loadStatus();
    }

    handleAnswerReceived(data) {
        this.answers.set(data.user_id, {
            nickname: data.nickname,
            answer_index: data.answer_index,
            is_correct: data.is_correct,
            new_score: data.new_score
        });
        
        this.addLog(`${data.nickname} が回答しました (${data.is_correct ? '正解' : '不正解'})`, 
                   data.is_correct ? 'success' : 'info');
        
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

    async startEvent() {
        this.elements.startBtn.disabled = true;
        
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
            this.elements.startBtn.disabled = false;
        }
    }

    async nextQuestion() {
        this.elements.nextBtn.disabled = true;
        
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
                    this.currentEvent.current_question = data.question_number;
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
            this.elements.nextBtn.disabled = false;
        }
    }

    async sendAlert() {
        this.elements.alertBtn.disabled = true;
        
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
                this.elements.alertBtn.disabled = false;
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
                    this.elements.totalQuestions.textContent = 
                        data.config.questions?.length || '-';
                }
            }
        } catch (error) {
            console.error('Error loading status:', error);
        }
    }

    updateEventStatus() {
        if (!this.currentEvent) {
            this.elements.eventStatus.textContent = '待機中';
            this.elements.currentQuestion.textContent = '-';
            this.elements.startBtn.disabled = false;
            this.elements.nextBtn.disabled = true;
            this.elements.alertBtn.disabled = true;
            this.elements.stopBtn.disabled = true;
            return;
        }

        this.elements.eventStatus.textContent = 
            this.currentEvent.status === 'started' ? '進行中' : 
            this.currentEvent.status === 'finished' ? '終了' : '待機中';
        
        this.elements.currentQuestion.textContent = this.currentEvent.current_question || 0;
        
        this.elements.startBtn.disabled = this.currentEvent.status === 'started';
        this.elements.nextBtn.disabled = this.currentEvent.status !== 'started';
        this.elements.alertBtn.disabled = this.currentEvent.status !== 'started';
        this.elements.stopBtn.disabled = this.currentEvent.status !== 'started';
    }

    updateParticipants(users) {
        this.participants.clear();
        users.forEach(user => this.participants.set(user.id, user));
        
        this.elements.participantCount.textContent = users.length;
        this.elements.participantsList.innerHTML = '';
        
        users.forEach(user => {
            const item = document.createElement('div');
            item.className = 'participant-item';
            
            item.innerHTML = `
                <div class="participant-info">
                    <div class="connection-status ${user.connected ? '' : 'disconnected'}"></div>
                    <span class="participant-name">${user.nickname}</span>
                </div>
                <span class="participant-score">${user.score}点</span>
            `;
            
            this.elements.participantsList.appendChild(item);
        });
    }

    updateQuestionDisplay() {
        if (!this.currentQuestion) {
            this.elements.currentQuestionDisplay.innerHTML = '<p>問題が開始されていません</p>';
            return;
        }

        const question = this.currentQuestion.question;
        let html = `
            <h4>問題 ${this.currentQuestion.question_number}</h4>
            <p><strong>${question.Text}</strong></p>
        `;
        
        if (question.Image) {
            html += `<img src="/images/${question.Image}" alt="問題画像" class="question-image">`;
        }
        
        html += '<div class="choices-list">';
        question.Choices.forEach((choice, index) => {
            // Convert 0-based index to 1-based for comparison with 1-based correct answer
            const isCorrect = (index + 1) === question.Correct;
            html += `
                <div class="choice-item ${isCorrect ? 'correct' : ''}">
                    ${String.fromCharCode(65 + index)}. ${choice}
                    ${isCorrect ? ' ✓' : ''}
                </div>
            `;
        });
        html += '</div>';
        
        this.elements.currentQuestionDisplay.innerHTML = html;
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
        // Update the team in our local storage
        if (data.team) {
            this.teams.set(data.team.id, data.team);
            this.updateTeamsDisplay();
            this.addLog(`${data.user.nickname} が ${data.team.name} に自動配置されました`, 'success');
        }
    }

    addLog(message, type = 'info') {
        const timestamp = new Date().toLocaleTimeString();
        const logEntry = document.createElement('div');
        logEntry.className = 'log-entry';
        
        logEntry.innerHTML = `
            <span class="log-timestamp">${timestamp}</span>
            <span class="log-type-${type}">${message}</span>
        `;
        
        this.elements.logDisplay.appendChild(logEntry);
        this.elements.logDisplay.scrollTop = this.elements.logDisplay.scrollHeight;
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
}

document.addEventListener('DOMContentLoaded', () => {
    new QuizAdmin();
});