class QuizParticipant {
    constructor() {
        this.ws = null;
        this.sessionID = localStorage.getItem('quiz_session_id') || null;
        this.user = null;
        this.currentQuestion = null;
        this.answered = false;

        this.initializeElements();
        this.setupEventListeners();
        
        if (this.sessionID) {
            this.rejoinSession();
        }
    }

    initializeElements() {
        this.elements = {
            connectionStatus: document.getElementById('connection-status'),
            connectionText: document.getElementById('connection-text'),
            
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
            btn.addEventListener('click', () => {
                const emoji = btn.dataset.emoji;
                this.sendEmoji(emoji);
                this.animateEmojiButton(btn);
            });
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
                break;
                
            case 'time_alert':
                this.showTimeAlert(message.data);
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
            alert('ニックネームを入力してください');
            return;
        }

        this.elements.joinBtn.disabled = true;
        this.elements.joinBtn.textContent = '参加中...';

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
            alert('参加に失敗しました: ' + error.message);
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
                localStorage.removeItem('quiz_session_id');
                this.sessionID = null;
            }
        } catch (error) {
            console.error('Error rejoining:', error);
            localStorage.removeItem('quiz_session_id');
            this.sessionID = null;
        }
    }

    showWaiting() {
        this.hideAllSections();
        this.elements.waitingSection.classList.remove('hidden');
    }

    showQuestion(questionData) {
        this.currentQuestion = questionData;
        this.answered = false;
        
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
            button.addEventListener('click', () => this.selectAnswer(index));
            
            this.elements.choicesContainer.appendChild(button);
        });
    }

    async selectAnswer(answerIndex) {
        if (this.answered) return;
        
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
                console.error('Error submitting answer:', data.error);
                this.answered = false;
                this.enableChoices();
            }
        } catch (error) {
            console.error('Error submitting answer:', error);
            this.answered = false;
            this.enableChoices();
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
        
        this.elements.finalScore.textContent = `あなたのスコア: ${this.user.score}点`;
        
        this.renderRankings(resultsData.results);
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

    showTimeAlert(data) {
        // Create alert overlay
        const alertOverlay = document.createElement('div');
        alertOverlay.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(255, 87, 34, 0.95);
            color: white;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            font-size: 2rem;
            font-weight: bold;
            z-index: 9999;
            animation: pulse 0.5s ease-in-out infinite alternate;
        `;

        alertOverlay.innerHTML = `
            <div style="text-align: center;">
                <div style="font-size: 4rem; margin-bottom: 1rem;">⏰</div>
                <div>${data.message}</div>
            </div>
        `;

        // Add CSS animation
        const style = document.createElement('style');
        style.textContent = `
            @keyframes pulse {
                0% { opacity: 0.8; transform: scale(1); }
                100% { opacity: 1; transform: scale(1.05); }
            }
        `;
        document.head.appendChild(style);

        document.body.appendChild(alertOverlay);

        // Remove after 3 seconds
        setTimeout(() => {
            if (alertOverlay.parentNode) {
                alertOverlay.parentNode.removeChild(alertOverlay);
            }
            if (style.parentNode) {
                style.parentNode.removeChild(style);
            }
        }, 3000);
    }

    hideAllSections() {
        this.elements.joinSection.classList.add('hidden');
        this.elements.waitingSection.classList.add('hidden');
        this.elements.questionSection.classList.add('hidden');
        this.elements.resultsSection.classList.add('hidden');
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
    new QuizParticipant();
});