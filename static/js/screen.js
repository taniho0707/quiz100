class QuizScreen {
    constructor() {
        this.ws = null;
        this.currentEvent = null;
        this.participants = new Map();
        this.currentQuestion = null;
        this.emojiAnimations = [];

        this.initializeElements();
        this.connectWebSocket();
        this.loadStatus();
        this.setupEmojiCleanup();
    }

    initializeElements() {
        this.elements = {
            connectionStatus: document.getElementById('connection-status'),
            connectionText: document.getElementById('connection-text'),
            
            eventTitle: document.getElementById('event-title'),
            questionStatus: document.getElementById('question-status'),
            participantCount: document.getElementById('participant-count'),
            
            waitingScreen: document.getElementById('waiting-screen'),
            questionScreen: document.getElementById('question-screen'),
            resultsScreen: document.getElementById('results-screen'),
            
            joinUrl: document.getElementById('join-url'),
            participantsGrid: document.getElementById('participants-grid'),
            
            currentQuestionNum: document.getElementById('current-question-num'),
            totalQuestions: document.getElementById('total-questions'),
            questionText: document.getElementById('question-text'),
            questionImage: document.getElementById('question-image'),
            choicesDisplay: document.getElementById('choices-display'),
            
            answerStats: document.getElementById('answer-stats'),
            progressFill: document.getElementById('progress-fill'),
            answerCount: document.getElementById('answer-count'),
            
            rankingsDisplay: document.getElementById('rankings-display'),
            emojiReactions: document.getElementById('emoji-reactions')
        };
        
        this.elements.joinUrl.textContent = window.location.origin;
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/screen`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('Screen WebSocket connected');
            this.updateConnectionStatus(true);
        };
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        };
        
        this.ws.onclose = () => {
            console.log('Screen WebSocket disconnected');
            this.updateConnectionStatus(false);
            setTimeout(() => this.connectWebSocket(), 3000);
        };
        
        this.ws.onerror = (error) => {
            console.error('Screen WebSocket error:', error);
            this.updateConnectionStatus(false);
        };
    }

    handleWebSocketMessage(message) {
        console.log('Received message:', message);
        
        switch (message.type) {
            case 'user_joined':
                this.handleUserJoined(message.data);
                break;
                
            case 'event_started':
                this.handleEventStarted(message.data);
                break;
                
            case 'question_start':
                this.handleQuestionStart(message.data);
                break;
                
            case 'answer_received':
                this.handleAnswerReceived(message.data);
                break;
                
            case 'time_alert':
                this.showTimeAlert(message.data);
                break;
                
            case 'final_results':
                this.handleFinalResults(message.data);
                break;
                
            case 'emoji':
                this.handleEmojiReaction(message.data);
                break;
                
            default:
                console.log('Unknown message type:', message.type);
        }
    }

    handleUserJoined(data) {
        this.loadStatus();
    }

    handleEventStarted(data) {
        this.currentEvent = data.event;
        this.elements.eventTitle.textContent = data.title;
        this.elements.questionStatus.textContent = 'イベント開始！';
        this.showWaitingScreen();
    }

    handleQuestionStart(data) {
        this.currentQuestion = data;
        this.showQuestionScreen();
        this.displayQuestion(data);
    }

    handleAnswerReceived(data) {
        this.updateAnswerProgress();
    }

    handleFinalResults(data) {
        this.showResultsScreen();
        this.displayFinalResults(data.results);
    }

    handleEmojiReaction(data) {
        this.showEmojiReaction(data.emoji);
    }

    async loadStatus() {
        try {
            const response = await fetch('/api/status');
            const data = await response.json();
            
            if (response.ok) {
                this.updateParticipants(data.users || []);
                
                if (data.event) {
                    this.currentEvent = data.event;
                    this.elements.eventTitle.textContent = data.config?.title || 'クイズ大会';
                }
            }
        } catch (error) {
            console.error('Error loading status:', error);
        }
    }

    updateParticipants(users) {
        this.participants.clear();
        users.forEach(user => this.participants.set(user.id, user));
        
        this.elements.participantCount.textContent = `参加者: ${users.length}人`;
        
        this.elements.participantsGrid.innerHTML = '';
        users.forEach(user => {
            const card = document.createElement('div');
            card.className = 'participant-card';
            card.innerHTML = `
                <div class="participant-name">${user.nickname}</div>
                <div class="participant-score">${user.score}点</div>
            `;
            this.elements.participantsGrid.appendChild(card);
        });
        
        this.updateAnswerProgress();
    }

    showWaitingScreen() {
        this.hideAllScreens();
        this.elements.waitingScreen.classList.remove('hidden');
        this.elements.questionStatus.textContent = '参加者をお待ちしています';
    }

    showQuestionScreen() {
        this.hideAllScreens();
        this.elements.questionScreen.classList.remove('hidden');
        this.elements.questionStatus.textContent = '問題進行中';
    }

    showResultsScreen() {
        this.hideAllScreens();
        this.elements.resultsScreen.classList.remove('hidden');
        this.elements.questionStatus.textContent = '結果発表';
    }

    hideAllScreens() {
        this.elements.waitingScreen.classList.add('hidden');
        this.elements.questionScreen.classList.add('hidden');
        this.elements.resultsScreen.classList.add('hidden');
    }

    displayQuestion(questionData) {
        const question = questionData.question;
        
        this.elements.currentQuestionNum.textContent = questionData.question_number;
        this.elements.totalQuestions.textContent = questionData.total_questions;
        this.elements.questionText.textContent = question.Text;
        
        if (question.Image) {
            this.elements.questionImage.src = `/images/${question.Image}`;
            this.elements.questionImage.classList.remove('hidden');
        } else {
            this.elements.questionImage.classList.add('hidden');
        }
        
        this.displayChoices(question.Choices, question.Correct);
        this.updateAnswerProgress();
    }

    displayChoices(choices, correctIndex) {
        this.elements.choicesDisplay.innerHTML = '';
        
        choices.forEach((choice, index) => {
            const choiceDiv = document.createElement('div');
            // Convert 0-based index to 1-based for comparison with 1-based correct answer
            choiceDiv.className = `choice-display ${(index + 1) === correctIndex ? 'correct' : ''}`;
            choiceDiv.innerHTML = `
                <span class="choice-letter">${String.fromCharCode(65 + index)}</span>
                ${choice}
            `;
            this.elements.choicesDisplay.appendChild(choiceDiv);
        });
    }

    updateAnswerProgress() {
        if (!this.currentQuestion) return;
        
        const totalParticipants = this.participants.size;
        const answeredCount = this.getAnsweredCount();
        const progress = totalParticipants > 0 ? (answeredCount / totalParticipants) * 100 : 0;
        
        this.elements.progressFill.style.width = `${progress}%`;
        this.elements.answerCount.textContent = `${answeredCount} / ${totalParticipants} 回答済み`;
    }

    getAnsweredCount() {
        return 5;
    }

    displayFinalResults(results) {
        results.sort((a, b) => b.score - a.score);
        
        this.elements.rankingsDisplay.innerHTML = '';
        
        results.slice(0, 10).forEach((user, index) => {
            const item = document.createElement('div');
            item.className = 'ranking-item';
            
            let trophy = '';
            if (index === 0) trophy = '🥇';
            else if (index === 1) trophy = '🥈';
            else if (index === 2) trophy = '🥉';
            
            item.innerHTML = `
                <span class="rank">${trophy} ${index + 1}位</span>
                <span class="name">${user.nickname}</span>
                <span class="score">${user.score}点</span>
            `;
            
            this.elements.rankingsDisplay.appendChild(item);
        });
    }

    showEmojiReaction(emoji) {
        const reaction = document.createElement('div');
        reaction.className = 'emoji-reaction';
        reaction.textContent = emoji;
        
        const startX = Math.random() * window.innerWidth;
        const startY = window.innerHeight - 100;
        
        reaction.style.left = `${startX}px`;
        reaction.style.top = `${startY}px`;
        
        this.elements.emojiReactions.appendChild(reaction);
        
        setTimeout(() => {
            if (reaction.parentNode) {
                reaction.parentNode.removeChild(reaction);
            }
        }, 3000);
    }

    setupEmojiCleanup() {
        setInterval(() => {
            const reactions = this.elements.emojiReactions.querySelectorAll('.emoji-reaction');
            if (reactions.length > 20) {
                for (let i = 0; i < reactions.length - 20; i++) {
                    reactions[i].remove();
                }
            }
        }, 1000);
    }

    showTimeAlert(data) {
        // Create alert overlay for large screen
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
            font-size: 5rem;
            font-weight: bold;
            text-align: center;
            z-index: 9999;
            animation: screenAlert 1s ease-in-out infinite alternate;
        `;

        alertOverlay.innerHTML = `
            <div>
                <div style="font-size: 8rem; margin-bottom: 2rem;">⏰</div>
                <div>${data.message}</div>
                <div style="font-size: 3rem; margin-top: 2rem; opacity: 0.8;">お急ぎください！</div>
            </div>
        `;

        // Add CSS animation for large screen
        const style = document.createElement('style');
        style.textContent = `
            @keyframes screenAlert {
                0% { 
                    opacity: 0.9; 
                    transform: scale(1);
                    background: rgba(255, 87, 34, 0.95);
                }
                100% { 
                    opacity: 1; 
                    transform: scale(1.02);
                    background: rgba(255, 152, 0, 0.95);
                }
            }
        `;
        document.head.appendChild(style);

        document.body.appendChild(alertOverlay);

        // Remove after 4 seconds (slightly longer for large screen)
        setTimeout(() => {
            if (alertOverlay.parentNode) {
                alertOverlay.parentNode.removeChild(alertOverlay);
            }
            if (style.parentNode) {
                style.parentNode.removeChild(style);
            }
        }, 4000);
    }

    updateConnectionStatus(connected) {
        if (connected) {
            this.elements.connectionStatus.className = 'connection-status connected';
            this.elements.connectionText.textContent = '接続済み';
        } else {
            this.elements.connectionStatus.className = 'connection-status disconnected';
            this.elements.connectionText.textContent = '接続中...';
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new QuizScreen();
});