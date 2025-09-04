class QuizScreen {
    constructor() {
        this.ws = null;
        this.currentEvent = null;
        this.participants = new Map();
        this.currentQuestion = null;
        this.emojiAnimations = [];
        this.countdownInterval = null;
        this.timeUpTimeout = null;
        this.answersBlocked = false;

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
            qrcodeImage: document.getElementById('qrcode-image'),
            questionText: document.getElementById('question-text'),
            questionImage: document.getElementById('question-image'),
            choicesDisplay: document.getElementById('choices-display'),
            
            answerStats: document.getElementById('answer-stats'),
            progressFill: document.getElementById('progress-fill'),
            answerCount: document.getElementById('answer-count'),
            
            countdownDisplay: document.getElementById('countdown-display'),
            countdownNumber: document.getElementById('countdown-number'),
            timeUpDisplay: document.getElementById('time-up-display'),
            countdownBorder: document.getElementById('countdown-border'),
            
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
            this.loadQuizInfo();
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
                
            case 'title_display':
                this.handleTitleDisplay(message.data);
                break;
                
            case 'team_assignment':
                this.handleTeamAssignment(message.data);
                break;
                
            case 'question_start':
                this.handleQuestionStart(message.data);
                break;
                
            case 'answer_received':
                this.handleAnswerReceived(message.data);
                break;
                
            case 'countdown':
                this.showCountdown(message.data.seconds_left);
                break;
                
            case 'question_end':
                this.hideCountdown();
                this.blockAnswers();
                break;
                
            case 'answer_stats':
                this.handleAnswerStats(message.data);
                break;
                
            case 'answer_reveal':
                this.handleAnswerReveal(message.data);
                break;
                
            case 'final_results':
                this.handleFinalResults(message.data);
                break;
                
            case 'celebration':
                this.handleCelebration(message.data);
                break;
                
            case 'emoji':
                this.handleEmojiReaction(message.data);
                break;
                
            case 'state_changed':
                this.handleStateChanged(message.data);
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
        this.answersBlocked = false;
        this.hideCountdown();
        this.elements.timeUpDisplay.classList.add('hidden');
        this.showQuestionScreen();
        this.displayQuestion(data);
    }

    handleAnswerReceived(data) {
        this.updateAnswerProgress();
    }

    handleFinalResults(data) {
        this.showResultsScreen();
        
        if (data.team_mode && data.teams) {
            // チーム戦の場合はチーム結果のみ表示
            this.displayTeamResults(data.teams);
        } else {
            // 個人戦の場合は従来通り
            this.displayFinalResults(data.results);
        }
    }

    handleTitleDisplay(data) {
        this.showTitleScreen();
    }

    handleTeamAssignment(data) {
        this.showTeamAssignmentScreen(data.teams);
    }

    handleAnswerStats(data) {
        this.showAnswerStatsScreen(data);
    }

    handleAnswerReveal(data) {
        this.showAnswerRevealScreen(data);
    }

    handleCelebration(data) {
        this.showCelebrationScreen();
    }

    handleEmojiReaction(data) {
        this.showEmojiReaction(data.emoji);
    }

    async loadQuizInfo() {
        try {
            const response = await fetch('/api/screen/info');
            const data = await response.json();
            
            if (response.ok) {
                this.elements.eventTitle.textContent = data.title || '';
                this.elements.qrcodeImage.src = data.qrcode || '';
            }
        } catch (error) {
            console.error('Error loading quiz info:', error);
        }
    }

    async loadStatus() {
        try {
            const response = await fetch('/api/status');
            const data = await response.json();
            
            if (response.ok) {
                this.updateParticipants(data.users || []);
                
                if (data.event) {
                    this.currentEvent = data.event;
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
        
        // 動的に作成された画面も非表示
        const titleScreen = document.getElementById('title-screen');
        const teamScreen = document.getElementById('team-assignment-screen');
        if (titleScreen) titleScreen.classList.add('hidden');
        if (teamScreen) teamScreen.classList.add('hidden');
    }

    displayQuestion(questionData) {
        const question = questionData.question;
        
        this.elements.currentQuestionNum.textContent = questionData.question_number;
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
            choiceDiv.className = `choice-display`;
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
        return 5; // FIXME:
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

    displayTeamResults(teams) {
        // チームを得点順にソート
        teams.sort((a, b) => b.score - a.score);
        
        this.elements.rankingsDisplay.innerHTML = '';
        
        teams.forEach((team, index) => {
            const teamItem = document.createElement('div');
            teamItem.className = 'team-ranking-item';
            
            let trophy = '';
            if (index === 0) trophy = '🥇';
            else if (index === 1) trophy = '🥈';
            else if (index === 2) trophy = '🥉';
            
            // チーム情報のヘッダー
            const teamHeader = document.createElement('div');
            teamHeader.className = 'team-header';
            teamHeader.innerHTML = `
                <span class="rank">${trophy} ${index + 1}位</span>
                <span class="team-name">${team.name}</span>
                <span class="team-score">${team.score}点</span>
            `;
            teamItem.appendChild(teamHeader);
            
            // チームメンバーの詳細
            if (team.members && team.members.length > 0) {
                const membersDiv = document.createElement('div');
                membersDiv.className = 'team-members';
                
                // メンバーを得点順にソート
                const sortedMembers = [...team.members].sort((a, b) => b.score - a.score);
                
                sortedMembers.forEach((member, memberIndex) => {
                    const memberDiv = document.createElement('div');
                    memberDiv.className = 'team-member';
                    
                    let memberTrophy = '';
                    if (memberIndex === 0 && sortedMembers.length > 1) {
                        memberTrophy = '👑'; // チーム内1位
                    }
                    
                    memberDiv.innerHTML = `
                        <span class="member-name">${memberTrophy} ${member.nickname}</span>
                        <span class="member-score">${member.score}点</span>
                    `;
                    
                    membersDiv.appendChild(memberDiv);
                });
                
                teamItem.appendChild(membersDiv);
            }
            
            this.elements.rankingsDisplay.appendChild(teamItem);
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

    showCountdown(secondsLeft) {
        if (secondsLeft > 5 || secondsLeft < 1) {
            this.hideCountdown();
            return;
        }
        
        // Show the countdown number
        this.elements.countdownNumber.textContent = secondsLeft;
        this.elements.countdownNumber.classList.remove('hidden');
        
        // Show red border effect
        this.elements.countdownBorder.classList.remove('hidden');
        
        if (secondsLeft === 1) {
            // Show "Time Up!" after 1 second
            setTimeout(() => {
                this.hideCountdown();
                this.showTimeUp();
            }, 1000);
        }
    }
    
    hideCountdown() {
        if (this.elements.countdownNumber) {
            this.elements.countdownNumber.classList.add('hidden');
        }
        if (this.elements.countdownBorder) {
            this.elements.countdownBorder.classList.add('hidden');
        }
    }
    
    showTimeUp() {
        this.elements.timeUpDisplay.classList.remove('hidden');
        
        // Hide after 3 seconds
        setTimeout(() => {
            this.elements.timeUpDisplay.classList.add('hidden');
        }, 3000);
    }
    
    showTitleScreen() {
        this.hideAllScreens();
        // Create or show title display screen
        let titleScreen = document.getElementById('title-screen');
        if (!titleScreen) {
            titleScreen = document.createElement('div');
            titleScreen.id = 'title-screen';
            titleScreen.className = 'screen-section';
            titleScreen.innerHTML = `
                <div class="title-display">
                    <h1 class="main-title">${this.elements.eventTitle.textContent}</h1>
                    <p class="welcome-message">🎉 クイズ大会へようこそ！</p>
                </div>
            `;
            document.querySelector('.screen-content').appendChild(titleScreen);
        }
        titleScreen.classList.remove('hidden');
        this.elements.questionStatus.textContent = 'タイトル表示中';
    }

    showTeamAssignmentScreen(teams) {
        this.hideAllScreens();
        // Create or show team assignment screen
        let teamScreen = document.getElementById('team-assignment-screen');
        if (!teamScreen) {
            teamScreen = document.createElement('div');
            teamScreen.id = 'team-assignment-screen';
            teamScreen.className = 'screen-section';
            teamScreen.innerHTML = `
                <div class="team-assignment-display">
                    <h2>🏆 チーム発表</h2>
                    <div id="team-assignment-list" class="teams-display">
                        <!-- チーム一覧がここに表示されます -->
                    </div>
                </div>
            `;
            document.querySelector('.screen-content').appendChild(teamScreen);
        }
        
        const teamList = teamScreen.querySelector('#team-assignment-list');
        teamList.innerHTML = '';
        
        teams.forEach((team, index) => {
            const teamDiv = document.createElement('div');
            teamDiv.className = 'team-item';
            teamDiv.innerHTML = `
                <div class="team-header">
                    <h3>${team.name}</h3>
                </div>
                <div class="team-members">
                    ${team.members.map(member => `
                        <div class="member-name">${member.nickname}</div>
                    `).join('')}
                </div>
            `;
            teamList.appendChild(teamDiv);
        });
        
        teamScreen.classList.remove('hidden');
        this.elements.questionStatus.textContent = 'チーム発表中';
    }

    showAnswerStatsScreen(data) {
        // この情報を問題画面に重ねて表示
        this.elements.questionScreen.classList.remove('hidden');
        
        // 回答状況表示を更新
        const progressFill = this.elements.progressFill;
        const answerCount = this.elements.answerCount;
        
        const progress = data.total_participants > 0 ? 
            (data.answered_count / data.total_participants) * 100 : 0;
        
        progressFill.style.width = `${progress}%`;
        answerCount.textContent = `${data.answered_count} / ${data.total_participants} 回答済み`;
        
        // 正解率も表示
        if (!document.getElementById('correct-rate-display')) {
            const correctRateDiv = document.createElement('div');
            correctRateDiv.id = 'correct-rate-display';
            correctRateDiv.className = 'correct-rate-display';
            correctRateDiv.innerHTML = `
                <h3>📊 正解率: ${Math.round(data.correct_rate)}%</h3>
                <p>正解者: ${data.correct_count}人 / 回答者: ${data.answered_count}人</p>
            `;
            this.elements.answerStats.appendChild(correctRateDiv);
        } else {
            const correctRateDiv = document.getElementById('correct-rate-display');
            correctRateDiv.innerHTML = `
                <h3>📊 正解率: ${Math.round(data.correct_rate)}%</h3>
                <p>正解者: ${data.correct_count}人 / 回答者: ${data.answered_count}人</p>
            `;
        }
        
        this.elements.questionStatus.textContent = '回答状況表示中';
    }

    showAnswerRevealScreen(data) {
        // 問題画面で正解をハイライト表示
        this.elements.questionScreen.classList.remove('hidden');
        
        const choices = this.elements.choicesDisplay.querySelectorAll('.choice-display');
        choices.forEach((choice, index) => {
            choice.classList.remove('correct', 'revealed');
            // data.correct_indexは1ベースなので、0ベースに調整して比較
            if ((index + 1) === data.correct_index) {
                choice.classList.add('correct', 'revealed');
            }
        });
        
        // 正解率表示を非表示
        const correctRateDiv = document.getElementById('correct-rate-display');
        if (correctRateDiv) {
            correctRateDiv.remove();
        }
        
        this.elements.questionStatus.textContent = '正解発表中';
    }

    showCelebrationScreen() {
        this.hideAllScreens();
        // 結果画面を表示してクラッカーアニメーション開始
        this.elements.resultsScreen.classList.remove('hidden');
        this.startConfettiAnimation();
        this.elements.questionStatus.textContent = '🎉 お疲れ様でした！';
    }

    startConfettiAnimation() {
        // 左右のクラッカーを作成
        const leftCracker = this.createCracker('left');
        const rightCracker = this.createCracker('right');
        
        document.body.appendChild(leftCracker);
        document.body.appendChild(rightCracker);
        
        // 紙吹雪アニメーション開始
        setTimeout(() => {
            this.createConfetti('left');
            this.createConfetti('right');
        }, 500);
        
        // 5秒後にクリーンアップ
        setTimeout(() => {
            leftCracker.remove();
            rightCracker.remove();
            this.clearConfetti();
        }, 5000);
    }

    createCracker(side) {
        const cracker = document.createElement('div');
        cracker.className = `cracker cracker-${side}`;
        cracker.style.cssText = `
            position: fixed;
            ${side}: 20px;
            top: 50%;
            width: 60px;
            height: 120px;
            background: linear-gradient(45deg, #FFD700, #FFA500);
            border-radius: 10px 10px 30px 30px;
            z-index: 1000;
            transform: translateY(-50%);
            animation: crackerShake 0.5s ease-in-out;
        `;
        return cracker;
    }

    createConfetti(side) {
        const colors = ['#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8', '#FFD93D'];
        const startX = side === 'left' ? 50 : window.innerWidth - 50;
        
        for (let i = 0; i < 50; i++) {
            const confetti = document.createElement('div');
            confetti.className = 'confetti';
            confetti.style.cssText = `
                position: fixed;
                left: ${startX}px;
                top: 50%;
                width: 10px;
                height: 10px;
                background: ${colors[Math.floor(Math.random() * colors.length)]};
                border-radius: ${Math.random() > 0.5 ? '50%' : '0'};
                z-index: 999;
                pointer-events: none;
                animation: confettiFall ${2 + Math.random() * 3}s linear forwards;
                animation-delay: ${Math.random() * 2}s;
                transform: rotate(${Math.random() * 360}deg);
            `;
            
            // ランダムな方向に飛ばす
            const angle = (side === 'left' ? 0.3 : 2.8) + (Math.random() - 0.5) * 0.8;
            const velocity = 100 + Math.random() * 200;
            const endX = startX + Math.cos(angle) * velocity;
            const endY = window.innerHeight + 100;
            
            confetti.style.setProperty('--end-x', `${endX}px`);
            confetti.style.setProperty('--end-y', `${endY}px`);
            
            document.body.appendChild(confetti);
        }
    }

    clearConfetti() {
        document.querySelectorAll('.confetti').forEach(confetti => confetti.remove());
    }

    blockAnswers() {
        this.answersBlocked = true;
        // No direct action needed here since this is the screen display
        // The participant.js handles answer blocking
    }

    handleStateChanged(data) {
        console.log('State changed:', data.new_state);
        
        // Update question status display with Japanese labels
        const stateLabels = {
            'waiting': '参加者待ち',
            'started': 'イベント開始',
            'title_display': 'タイトル表示',
            'team_assignment': 'チーム分け',
            'question_active': '問題表示中',
            'countdown_active': 'カウントダウン中',
            'answer_stats': '回答状況表示',
            'answer_reveal': '回答発表',
            'results': '結果発表',
            'celebration': 'お疲れ様画面',
            'finished': '終了'
        };
        
        this.elements.questionStatus.textContent = stateLabels[data.new_state] || data.new_state;
        
        // Handle state-specific transitions
        switch (data.new_state) {
            case 'waiting':
                this.showWaitingScreen();
                break;
                
            case 'title_display':
                this.handleTitleDisplay({ title: this.elements.eventTitle.textContent });
                break;
                
            case 'team_assignment':
                // Trigger team display if teams exist
                this.loadStatus(); // This will reload teams
                break;
                
            case 'question_active':
                if (data.question) {
                    // Set current question and display it
                    this.currentQuestion = {
                        question_number: data.question_number,
                        question: data.question,
                        total_questions: data.total_questions
                    };
                    this.handleQuestionStart(this.currentQuestion);
                } else if (data.current_question > 0) {
                    // Load question from API if not provided
                    this.loadQuestionFromAPI(data.current_question);
                }
                break;
                
            case 'countdown_active':
                // Countdown will be handled by separate countdown messages
                break;
                
            case 'answer_stats':
                this.hideAllScreens();
                this.elements.answerStats.style.display = 'block';
                break;
                
            case 'answer_reveal':
                // Show question screen with answer revealed
                if (this.currentQuestion) {
                    this.showQuestionScreen();
                }
                break;
                
            case 'results':
                this.handleFinalResults({ results: [], teams: [], team_mode: false });
                break;
                
            case 'celebration':
                this.handleCelebration({});
                break;
                
            case 'finished':
                this.showWaitingScreen();
                this.elements.questionStatus.textContent = '終了';
                break;
        }
    }

    async loadQuestionFromAPI(questionNumber) {
        try {
            const response = await fetch('/api/status');
            if (response.ok) {
                const data = await response.json();
                if (data.config && data.config.questions && questionNumber <= data.config.questions.length) {
                    const question = data.config.questions[questionNumber - 1];
                    this.currentQuestion = {
                        question_number: questionNumber,
                        question: question,
                        total_questions: data.config.questions.length
                    };
                    this.handleQuestionStart(this.currentQuestion);
                }
            }
        } catch (error) {
            console.error('Failed to load question from API:', error);
        }
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
