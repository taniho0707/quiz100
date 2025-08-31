class QuizParticipant {
    constructor() {
        this.ws = null;
        this.sessionID = localStorage.getItem('quiz_session_id') || null;
        this.user = null;
        this.currentQuestion = null;
        this.answered = false;
        this.lastTouchEnd = 0;
        this.answersBlocked = false;

        this.initializeElements();
        this.setupEventListeners();
        this.preventZoom();
        
        if (this.sessionID) {
            this.rejoinSession();
        }
    }

    initializeElements() {
        this.elements = {
            connectionStatus: document.getElementById('connection-status'),
            connectionText: document.getElementById('connection-text'),
            
            resetSessionBtn: document.getElementById('reset-session-btn'),
            resetModal: document.getElementById('reset-modal'),
            resetCancelBtn: document.getElementById('reset-cancel-btn'),
            resetConfirmBtn: document.getElementById('reset-confirm-btn'),
            
            joinSection: document.getElementById('join-section'),
            nickname: document.getElementById('nickname'),
            joinBtn: document.getElementById('join-btn'),
            
            waitingSection: document.getElementById('waiting-section'),
            userNickname: document.getElementById('user-nickname'),
            userScore: document.getElementById('user-score'),
            
            questionSection: document.getElementById('question-section'),
            currentQuestionNum: document.getElementById('current-question-num'),
            totalQuestions: document.getElementById('total-questions'),
            questionText: document.getElementById('question-text'),
            questionImage: document.getElementById('question-image'),
            choicesContainer: document.getElementById('choices-container'),
            currentScore: document.getElementById('current-score'),
            
            answerFeedback: document.getElementById('answer-feedback'),
            feedbackText: document.getElementById('feedback-text'),
            
            resultsSection: document.getElementById('results-section'),
            finalScore: document.getElementById('final-score'),
            rankings: document.getElementById('rankings'),
            
            emojiButtons: document.querySelectorAll('.emoji-btn')
        };
    }

    setupEventListeners() {
        this.elements.joinBtn.addEventListener('click', () => this.joinQuiz());
        this.elements.nickname.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.joinQuiz();
        });

        this.elements.emojiButtons.forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                const emoji = btn.dataset.emoji;
                this.sendEmoji(emoji);
                this.animateEmojiButton(btn);
            });
            
            // Touch feedback for mobile
            btn.addEventListener('touchstart', (e) => {
                e.preventDefault();
                btn.style.transform = 'scale(0.95)';
            }, { passive: false });
            
            btn.addEventListener('touchend', (e) => {
                e.preventDefault();
                btn.style.transform = 'scale(1)';
                const emoji = btn.dataset.emoji;
                this.sendEmoji(emoji);
                this.animateEmojiButton(btn);
            }, { passive: false });
        });

        // セッション破棄関連のイベントリスナー
        this.elements.resetSessionBtn.addEventListener('click', () => this.showResetModal());
        this.elements.resetCancelBtn.addEventListener('click', () => this.hideResetModal());
        this.elements.resetConfirmBtn.addEventListener('click', () => this.resetSession());
        
        // モーダルのオーバーレイクリックで閉じる
        this.elements.resetModal.addEventListener('click', (e) => {
            if (e.target === this.elements.resetModal) {
                this.hideResetModal();
            }
        });
    }

    connectWebSocket() {
        if (!this.sessionID) return;

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/participant?session_id=${this.sessionID}`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.updateConnectionStatus(true);
        };
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus(false);
            setTimeout(() => this.connectWebSocket(), 3000);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.updateConnectionStatus(false);
        };
    }

    handleWebSocketMessage(message) {
        console.log('Received message:', message);
        
        switch (message.type) {
            case 'event_started':
                this.showWaiting();
                break;
                
            case 'question_start':
                this.showQuestion(message.data);
                break;
                
            case 'question_end':
                this.disableChoices();
                this.blockAnswers();
                break;
                
            case 'time_alert': // FIXME: 消したい
                break;
                
            case 'final_results':
                this.showResults(message.data);
                break;
                
            default:
                console.log('Unknown message type:', message.type);
        }
    }

    async joinQuiz() {
        const nickname = this.elements.nickname.value.trim();
        if (!nickname) {
            this.showMessage('ニックネームを入力してください');
            return;
        }

        if (nickname.length > 20) {
            this.showMessage('ニックネームは20文字以内で入力してください');
            return;
        }

        this.elements.joinBtn.disabled = true;
        this.elements.joinBtn.textContent = '参加中...';
        
        // Hide mobile keyboard
        this.elements.nickname.blur();

        try {
            const headers = {
                'Content-Type': 'application/json'
            };
            
            if (this.sessionID) {
                headers['X-Session-ID'] = this.sessionID;
            }

            const response = await fetch('/api/join', {
                method: 'POST',
                headers: headers,
                body: JSON.stringify({ nickname: nickname })
            });

            const data = await response.json();
            
            if (response.ok) {
                this.user = data.user;
                this.sessionID = data.session_id;
                localStorage.setItem('quiz_session_id', this.sessionID);
                
                this.elements.userNickname.textContent = this.user.nickname;
                this.elements.userScore.textContent = this.user.score;
                this.elements.currentScore.textContent = this.user.score;
                
                this.showWaiting();
                this.connectWebSocket();
            } else {
                throw new Error(data.error || 'Failed to join quiz');
            }
        } catch (error) {
            console.error('Error joining quiz:', error);
            this.showMessage('参加に失敗しました: ' + error.message);
        } finally {
            this.elements.joinBtn.disabled = false;
            this.elements.joinBtn.textContent = '参加する';
        }
    }

    async rejoinSession() {
        try {
            const response = await fetch('/api/join', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': this.sessionID
                },
                body: JSON.stringify({ nickname: 'Rejoining...' })
            });

            const data = await response.json();
            
            if (response.ok) {
                this.user = data.user;
                this.elements.userNickname.textContent = this.user.nickname;
                this.elements.userScore.textContent = this.user.score;
                this.elements.currentScore.textContent = this.user.score;
                
                this.showWaiting();
                this.connectWebSocket();
            } else {
                // Session rejoin failed, clear session and show join screen
                localStorage.removeItem('quiz_session_id');
                this.sessionID = null;
                this.showJoinScreen();
            }
        } catch (error) {
            console.error('Error rejoining:', error);
            localStorage.removeItem('quiz_session_id');
            this.sessionID = null;
            this.showJoinScreen();
        }
    }

    showJoinScreen() {
        this.hideAllSections();
        this.elements.joinSection.classList.remove('hidden');
        this.elements.nickname.value = ''; // Clear nickname field
        this.elements.nickname.focus(); // Focus on nickname input
    }

    showWaiting() {
        this.hideAllSections();
        this.elements.waitingSection.classList.remove('hidden');
    }

    showQuestion(questionData) {
        this.currentQuestion = questionData;
        this.answered = false;
        this.answersBlocked = false;
        
        this.hideAllSections();
        this.elements.questionSection.classList.remove('hidden');
        
        this.elements.currentQuestionNum.textContent = questionData.question_number;
        this.elements.totalQuestions.textContent = questionData.total_questions;
        this.elements.questionText.textContent = questionData.question.text;
        
        if (questionData.question.Image) {
            this.elements.questionImage.src = `/images/${questionData.question.Image}`;
            this.elements.questionImage.classList.remove('hidden');
        } else {
            this.elements.questionImage.classList.add('hidden');
        }
        
        this.renderChoices(questionData.question.Choices);
    }

    renderChoices(choices) {
        this.elements.choicesContainer.innerHTML = '';
        
        choices.forEach((choice, index) => {
            const button = document.createElement('button');
            button.className = 'choice-btn';
            button.textContent = `${String.fromCharCode(65 + index)}. ${choice}`;
            
            // Standard click event
            button.addEventListener('click', (e) => {
                e.preventDefault();
                this.selectAnswer(index);
            });
            
            // Touch events for better mobile experience
            button.addEventListener('touchstart', (e) => {
                if (this.answered) return;
                button.style.backgroundColor = '#f0f8ff';
            }, { passive: true });
            
            button.addEventListener('touchend', (e) => {
                e.preventDefault();
                if (this.answered) return;
                button.style.backgroundColor = '';
                this.selectAnswer(index);
            }, { passive: false });
            
            button.addEventListener('touchcancel', (e) => {
                button.style.backgroundColor = '';
            });
            
            this.elements.choicesContainer.appendChild(button);
        });
    }

    async selectAnswer(answerIndex) {
        if (this.answered || this.answersBlocked) return;
        
        this.answered = true;
        this.disableChoices();
        
        try {
            const response = await fetch('/api/answer', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': this.sessionID
                },
                body: JSON.stringify({
                    question_number: this.currentQuestion.question_number,
                    answer_index: answerIndex + 1  // Convert 0-based to 1-based
                })
            });

            const data = await response.json();
            
            if (response.ok) {
                this.showFeedback(data.is_correct);
                this.user.score = data.new_score;
                this.elements.currentScore.textContent = data.new_score;
                this.elements.userScore.textContent = data.new_score;
            } else {
                if (!this.answersBlocked) {
                    console.error('Error submitting answer:', data.error);
                    this.answered = false;
                    this.enableChoices();
                }
            }
        } catch (error) {
            if (!this.answersBlocked) {
                console.error('Error submitting answer:', error);
                this.answered = false;
                this.enableChoices();
            }
        }
    }

    disableChoices() {
        const choices = this.elements.choicesContainer.querySelectorAll('.choice-btn');
        choices.forEach(btn => btn.disabled = true);
    }

    enableChoices() {
        const choices = this.elements.choicesContainer.querySelectorAll('.choice-btn');
        choices.forEach(btn => btn.disabled = false);
    }
    
    blockAnswers() {
        this.answersBlocked = true;
        this.disableChoices();
    }

    showFeedback(isCorrect) {
        this.elements.feedbackText.textContent = isCorrect ? '正解！ 🎉' : '不正解 😔';
        this.elements.answerFeedback.className = `feedback ${isCorrect ? 'correct' : 'incorrect'}`;
        this.elements.answerFeedback.classList.remove('hidden');
        
        setTimeout(() => {
            this.elements.answerFeedback.classList.add('hidden');
        }, 2000);
    }

    showResults(resultsData) {
        this.hideAllSections();
        this.elements.resultsSection.classList.remove('hidden');
        
        if (resultsData.team_mode && resultsData.teams) {
            // チーム戦の場合はチーム結果のみ表示
            this.elements.finalScore.textContent = `チーム戦結果`;
            this.renderTeamRankings(resultsData.teams);
        } else {
            // 個人戦の場合は従来通り
            this.elements.finalScore.textContent = `あなたのスコア: ${this.user.score}点`;
            this.renderRankings(resultsData.results);
        }
    }

    renderRankings(results) {
        results.sort((a, b) => b.score - a.score);
        
        this.elements.rankings.innerHTML = '';
        
        results.forEach((user, index) => {
            const item = document.createElement('div');
            item.className = 'ranking-item';
            
            const isCurrentUser = user.id === this.user.id;
            if (isCurrentUser) {
                item.style.backgroundColor = '#f0f8ff';
                item.style.fontWeight = 'bold';
            }
            
            item.innerHTML = `
                <span class="rank">${index + 1}位</span>
                <span class="name">${user.nickname}</span>
                <span class="score">${user.score}点</span>
            `;
            
            this.elements.rankings.appendChild(item);
        });
    }

    renderTeamRankings(teams) {
        // チームを得点順にソート
        teams.sort((a, b) => b.score - a.score);
        
        this.elements.rankings.innerHTML = '';
        
        // 現在のユーザーが所属するチームを特定
        let userTeamId = null;
        if (this.user) {
            for (const team of teams) {
                if (team.members && team.members.some(member => member.id === this.user.id)) {
                    userTeamId = team.id;
                    break;
                }
            }
        }
        
        teams.forEach((team, index) => {
            const teamItem = document.createElement('div');
            teamItem.className = 'team-ranking-item';
            
            // 自分のチームをハイライト
            const isUserTeam = team.id === userTeamId;
            if (isUserTeam) {
                teamItem.style.backgroundColor = '#f0f8ff';
                teamItem.style.fontWeight = 'bold';
                teamItem.style.border = '2px solid #007bff';
            }
            
            // チーム情報のヘッダー
            const teamHeader = document.createElement('div');
            teamHeader.className = 'team-header';
            teamHeader.innerHTML = `
                <span class="rank">${index + 1}位</span>
                <span class="team-name">${team.name}</span>
                <span class="team-score">${team.score}点</span>
            `;
            teamItem.appendChild(teamHeader);
            
            // チームメンバーの詳細
            if (team.members && team.members.length > 0) {
                const membersDiv = document.createElement('div');
                membersDiv.className = 'team-members';
                
                team.members.forEach(member => {
                    const memberDiv = document.createElement('div');
                    memberDiv.className = 'team-member';
                    
                    // 自分自身をハイライト
                    const isCurrentUser = member.id === this.user.id;
                    if (isCurrentUser) {
                        memberDiv.style.backgroundColor = '#e6f3ff';
                        memberDiv.style.fontWeight = 'bold';
                    }
                    
                    memberDiv.innerHTML = `
                        <span class="member-name">${member.nickname}</span>
                        <span class="member-score">${member.score}点</span>
                    `;
                    
                    membersDiv.appendChild(memberDiv);
                });
                
                teamItem.appendChild(membersDiv);
            }
            
            this.elements.rankings.appendChild(teamItem);
        });
    }

    async sendEmoji(emoji) {
        if (!this.sessionID) return;
        
        try {
            await fetch('/api/emoji', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': this.sessionID
                },
                body: JSON.stringify({ emoji: emoji })
            });
        } catch (error) {
            console.error('Error sending emoji:', error);
        }
    }

    animateEmojiButton(button) {
        button.style.transform = 'scale(1.2)';
        setTimeout(() => {
            button.style.transform = 'scale(1)';
        }, 200);
    }

    hideAllSections() {
        this.elements.joinSection.classList.add('hidden');
        this.elements.waitingSection.classList.add('hidden');
        this.elements.questionSection.classList.add('hidden');
        this.elements.resultsSection.classList.add('hidden');
    }

    showResetModal() {
        this.elements.resetModal.classList.remove('hidden');
    }

    hideResetModal() {
        this.elements.resetModal.classList.add('hidden');
    }

    async resetSession() {
        if (!this.sessionID) {
            this.showMessage('セッションが見つかりません');
            return;
        }

        try {
            const response = await fetch('/api/reset-session', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': this.sessionID
                }
            });

            if (response.ok) {
                // セッション情報をクリア
                localStorage.removeItem('quiz_session_id');
                this.sessionID = null;
                this.user = null;
                
                // WebSocket接続を切断
                if (this.ws) {
                    this.ws.close();
                    this.ws = null;
                }
                
                // モーダルを閉じる
                this.hideResetModal();
                
                // 参加登録画面に戻る
                this.hideAllSections();
                this.elements.joinSection.classList.remove('hidden');
                this.elements.nickname.value = '';
                
                // 接続状態をリセット
                this.updateConnectionStatus(false);
                
                this.showMessage('セッションが破棄されました。再度参加登録を行ってください。');
            } else {
                const error = await response.json();
                this.showMessage(`セッション破棄に失敗しました: ${error.error}`);
            }
        } catch (error) {
            console.error('Reset session error:', error);
            this.showMessage('セッション破棄中にエラーが発生しました');
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
    
    showMessage(message) {
        // Replace alert() with a mobile-friendly message display
        if (window.confirm) {
            // Use confirm for important messages that need user acknowledgment
            if (message.includes('破棄') || message.includes('失敗')) {
                confirm(message);
            } else {
                alert(message);
            }
        } else {
            alert(message);
        }
    }
    
    preventZoom() {
        // Prevent double-tap zoom on iOS
        document.addEventListener('touchend', (e) => {
            const now = new Date().getTime();
            if (now - this.lastTouchEnd <= 300) {
                e.preventDefault();
            }
            this.lastTouchEnd = now;
        }, false);
        
        this.lastTouchEnd = 0;
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new QuizParticipant();
});