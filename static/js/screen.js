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

      // eventTitle: document.getElementById('event-title'),
      // questionStatus: document.getElementById('question-status'),
      // participantCount: document.getElementById('participant-count'),
      questionHeader: document.getElementById('question-header'),
      headerWaiting: document.getElementById('header-waiting'),
      // headerFinalResult: document.getElementById('header-final-result'),

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
      // progressFill: document.getElementById('progress-fill'),
      // answerCount: document.getElementById('answer-count'),

      countdownDisplay: document.getElementById('countdown-display'),
      countdownNumber: document.getElementById('countdown-number'),
      timeUpDisplay: document.getElementById('time-up-display'),
      countdownBorder: document.getElementById('countdown-border'),

      rankingsDisplay: document.getElementById('rankings-display'),
      emojiReactions: document.getElementById('emoji-reactions'),

      answerRevealImageContainer: document.getElementById(
        'answer-reveal-image-container'
      ),
      answerRevealImage: document.getElementById('answer-reveal-image'),
    };

    this.elements.joinUrl.textContent = window.location.origin;
  }

  /**
   * 音声ファイルを再生する（エラーが発生しても無視）
   * @param {string} audioFileName - 再生する音声ファイル名
   */
  playAudio(audioFileName) {
    try {
      const audio = new Audio(`/audio/${audioFileName}`);
      audio.play().catch((error) => {
        // 音声ファイルが見つからない場合やブラウザが再生を拒否した場合は無視
        console.log(`Audio playback skipped: ${audioFileName}`, error.message);
      });
    } catch (error) {
      // エラーが発生しても無視
      console.log(
        `Audio initialization failed: ${audioFileName}`,
        error.message
      );
    }
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
      case 'initial_sync':
        this.handleInitialSync(message.data);
        break;

      case 'user_joined':
        this.handleUserJoined(message.data);
        break;

      case 'user_left':
        this.handleUserLeft(message.data);
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
        // do nothing because running 'showCountdown(1)' to set 1sec timer
        // this.hideCountdown();
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

  handleInitialSync(data) {
    console.log('Initial sync received:', data);

    if (!data) {
      console.warn('No sync data received');
      return;
    }

    // Update participants
    this.loadStatus();

    // Sync to current state with appropriate data
    const { EVENT_STATES } = QuizConstants;

    switch (data.event_state) {
      case EVENT_STATES.WAITING:
      case EVENT_STATES.STARTED:
        this.showWaitingScreen();
        break;

      case EVENT_STATES.TITLE_DISPLAY:
        // this.showTitleScreen({ title: this.elements.eventTitle.textContent });
        break;

      case EVENT_STATES.TEAM_ASSIGNMENT:
        if (data.team && data.team.length > 0) {
          this.showTeamAssignmentScreen(data.team);
        } else {
          this.showWaitingScreen();
        }
        break;

      case EVENT_STATES.QUESTION_ACTIVE:
        if (data.question) {
          this.currentQuestion = {
            question_number: data.question_number,
            question: data.question,
            total_questions: data.total_questions,
          };
          this.displayQuestion(data);
          this.showQuestionScreen();
        } else {
          this.showWaitingScreen();
        }
        break;

      case EVENT_STATES.COUNTDOWN_ACTIVE:
        if (data.question) {
          this.currentQuestion = {
            question_number: data.question_number,
            question: data.question,
            total_questions: data.total_questions,
          };
          this.displayQuestion(data);
          this.showQuestionScreen();
          // カウントダウンは表示しない（リロード時には既に終わっている可能性が高い）
        } else {
          this.showWaitingScreen();
        }
        break;

      case EVENT_STATES.ANSWER_STATS:
        if (data.question && data.participant_data) {
          this.currentQuestion = {
            question_number: data.question_number,
            question: data.question,
            total_questions: data.total_questions,
          };
          // 回答統計を表示
          const totalParticipants = data.participant_data.length;
          const choicesCounts = this.calculateChoicesCounts(
            data.answer_data,
            data.question.choices.length
          );
          this.displayQuestion(data);
          this.showQuestionScreen();
          this.showChoicesWithCounts(totalParticipants, choicesCounts);
        } else {
          this.showWaitingScreen();
        }
        break;

      case EVENT_STATES.ANSWER_REVEAL:
        if (data.question) {
          this.currentQuestion = {
            question_number: data.question_number,
            question: data.question,
            total_questions: data.total_questions,
          };
          this.displayQuestion(data);
          this.showQuestionScreen();

          // 回答統計を先に表示
          if (data.participant_data && data.answer_data) {
            const totalParticipants = data.participant_data.length;
            const choicesCounts = this.calculateChoicesCounts(
              data.answer_data,
              data.question.choices.length
            );
            this.showChoicesWithCounts(totalParticipants, choicesCounts);
          }

          // その後に正解をハイライト表示（選択肢再描画後に実行）
          if (data.question && data.question.correct !== undefined) {
            this.showAnswerRevealScreen({ correct: data.question.correct });
          }
        } else {
          this.showWaitingScreen();
        }
        break;

      case EVENT_STATES.RESULTS:
      case EVENT_STATES.CELEBRATION:
        // 結果画面を表示
        if (data.team && data.team.length > 0) {
          // チーム戦の結果
          this.handleFinalResults({
            team_mode: true,
            teams: data.team,
            results: [],
          });
        } else if (data.participant_data) {
          // 個人戦の結果
          this.handleFinalResults({
            team_mode: false,
            teams: [],
            results: data.participant_data,
          });
        } else {
          this.showWaitingScreen();
        }
        break;

      case EVENT_STATES.FINISHED:
        this.showWaitingScreen();
        break;

      default:
        console.log('Unknown event state in sync:', data.event_state);
        this.showWaitingScreen();
    }

    console.log('Screen synchronized with server state:', data.event_state);
  }

  handleUserJoined(data) {
    this.loadStatus();
  }

  handleUserLeft(data) {
    this.loadStatus();
  }

  handleEventStarted(data) {
    this.currentEvent = data.event;
    // this.elements.eventTitle.textContent = data.title;
    this.showWaitingScreen();
  }

  handleQuestionStart(data) {
    this.currentQuestion = data;
    this.answersBlocked = false;
    this.hideCountdown();
    this.elements.timeUpDisplay.classList.add('hidden');
    this.hideAnswerRevealImage();
    this.displayQuestion(data);
    this.elements.questionHeader.classList.remove('hidden');
    this.showQuestionScreen();

    // 問題表示時の音声再生
    this.playAudio('question_start.mp3');
  }

  handleAnswerReceived(data) {
    this.updateAnswerProgress();
  }

  handleFinalResults(data) {
    this.elements.questionHeader.classList.add('hidden');
    // this.elements.headerFinalResult.classList.remove('hidden');

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
    this.showTitleScreen(data);
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
        // this.elements.eventTitle.textContent = data.title || '';
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
    users.forEach((user) => this.participants.set(user.id, user));

    this.elements.participantsGrid.innerHTML = '';
    users.forEach((user) => {
      const card = document.createElement('div');
      card.className = 'participant-card';
      card.innerHTML = `
                <div class="participant-name">${user.nickname}</div>
            `;
      this.elements.participantsGrid.appendChild(card);
    });

    // 新規参加者が追加されたときに最上部にスクロール
    // this.elements.participantsGrid.scrollTop = 0;
    this.elements.participantsGrid.scrollTo(0, -1000000);

    this.updateAnswerProgress();
  }

  calculateChoicesCounts(answerData, choicesCount) {
    const counts = new Array(choicesCount).fill(0);

    if (!answerData) return counts;

    // answerData は { "user_id": answer_index } の形式
    Object.values(answerData).forEach((answerIndex) => {
      const index = answerIndex - 1; // Convert 1-based to 0-based
      if (index >= 0 && index < choicesCount) {
        counts[index]++;
      }
    });

    return counts;
  }

  showWaitingScreen() {
    this.hideAllScreens();
    this.elements.waitingScreen.classList.remove('hidden');
  }

  showQuestionScreen() {
    this.hideAllScreens();
    this.elements.questionScreen.classList.remove('hidden');
  }

  showResultsScreen() {
    this.hideAllScreens();
    this.elements.resultsScreen.classList.remove('hidden');
  }

  hideAllScreens() {
    this.elements.waitingScreen.classList.add('hidden');
    this.elements.questionScreen.classList.add('hidden');

    // 動的に作成された画面も非表示
    const titleScreen = document.getElementById('title-screen');
    const teamScreen = document.getElementById('team-assignment-screen');
    if (titleScreen) titleScreen.classList.add('hidden');
    if (teamScreen) teamScreen.classList.add('hidden');
  }

  displayQuestion(questionData) {
    const question = questionData.question;

    this.elements.currentQuestionNum.textContent =
      questionData.question_number - 1; // FIXME: 0問目スタートのための暫定対応
    this.elements.questionText.textContent = question.text;

    // 問題出題中は画像を表示しない（正答発表後のみ表示）
    this.elements.questionImage.hidden = true;

    this.displayChoices(question.choices);
    this.updateAnswerProgress();
  }

  displayChoices(choices) {
    this.elements.choicesDisplay.innerHTML = '';

    choices.forEach((choice, index) => {
      const choiceDiv = document.createElement('div');
      // Convert 0-based index to 1-based for comparison with 1-based correct answer
      choiceDiv.className = `choice-display choice-with-stats`;
      choiceDiv.innerHTML = `
                <span class="choice-letter">${String.fromCharCode(
                  65 + index
                )}</span>
                <span class="choice-text">${choice}</span>
                <span class="choice-count" style="visibility: hidden;">X人</span>
            `;
      this.elements.choicesDisplay.appendChild(choiceDiv);
    });
  }

  showChoicesWithCounts(totalParticipants, choicesCounts) {
    if (!totalParticipants || !choicesCounts) return;

    this.elements.choicesDisplay.innerHTML = '';

    // 最も多く選ばれた人数を見つける
    const maxCount = Math.max(...choicesCounts);

    this.currentQuestion.question.choices.forEach((choice, index) => {
      const choiceDiv = document.createElement('div');
      const count = choicesCounts[index] || 0;
      choiceDiv.className = `choice-display choice-with-stats`;

      // 最多回答の選択肢かどうか判定
      const isMostPopular = count === maxCount && maxCount > 0;
      const countClass = isMostPopular
        ? 'choice-count most-popular'
        : 'choice-count';

      choiceDiv.innerHTML = `
                <span class="choice-letter">${String.fromCharCode(
                  65 + index
                )}</span>
                <span class="choice-text">${choice}</span>
                <span class="${countClass}">${count}人</span>
            `;
      this.elements.choicesDisplay.appendChild(choiceDiv);
    });
  }

  updateAnswerProgress() {
    // if (!this.currentQuestion) return;
    // const totalParticipants = this.participants.size;
    // const answeredCount = this.getAnsweredCount();
    // const progress = totalParticipants > 0 ? (answeredCount / totalParticipants) * 100 : 0;
    // this.elements.progressFill.style.width = `${progress}%`;
    // this.elements.answerCount.textContent = `${answeredCount} / ${totalParticipants} 回答済み`;
  }

  getAnsweredCount() {
    return 5; // FIXME:
  }

  displayFinalResults(results) {
    results.sort((a, b) => b.score - a.score);

    // 表彰台（1-3位）を表示
    if (results.length >= 1) {
      document.getElementById('first-place-team').textContent =
        results[0].nickname;
      document.getElementById('first-place-score').textContent =
        results[0].score;
    }
    if (results.length >= 2) {
      document.getElementById('second-place-team').textContent =
        results[1].nickname;
      document.getElementById('second-place-score').textContent =
        results[1].score;
    }
    if (results.length >= 3) {
      document.getElementById('third-place-team').textContent =
        results[2].nickname;
      document.getElementById('third-place-score').textContent =
        results[2].score;
    }

    // 一般順位（4位以下）をグリッドに表示
    const generalRankings = document.getElementById('general-rankings');
    generalRankings.innerHTML = '';

    // 4位以下、最大47位まで（11×5グリッド - 3 = 52 - 3 = 47位まで）
    results.slice(3, 50).forEach((user, index) => {
      const rank = index + 4; // 4位からスタート
      const item = document.createElement('div');
      item.className = 'ranking-item';

      item.innerHTML = `
                <div class="rank">${rank}位</div>
                <div class="team-name">${user.nickname}</div>
                <div class="team-score">${user.score}点</div>
            `;

      generalRankings.appendChild(item);
    });
  }

  displayTeamResults(teams) {
    // チームを得点順にソート（最下位から表示するため逆順）
    teams.sort((a, b) => b.score - a.score);

    // 結果表示エリアをクリア
    this.elements.rankingsDisplay.innerHTML = `
            <div id="team-results-container" class="team-results-container">
                <!-- チーム結果がここに順次表示されます -->
            </div>
        `;

    const container = document.getElementById('team-results-container');

    // 最下位から順次表示するため配列を逆順にする
    const reversedTeams = [...teams].reverse();

    // 各チームを順次表示
    this.displayTeamsSequentially(reversedTeams, container, 0);
  }

  displayTeamsSequentially(teams, container, index) {
    if (index >= teams.length) {
      // 全チーム表示完了後、紙吹雪を開始
      setTimeout(() => {
        this.startFullScreenConfetti();
      }, 1000);
      return;
    }

    const team = teams[index];
    const rank = teams.length - index; // 最下位から表示するので順位を計算

    // チーム要素を作成
    const teamElement = document.createElement('div');
    teamElement.className = `team-result-item rank-${rank}`;

    // 順位に応じたクラスを追加
    if (rank === 1) {
      teamElement.classList.add('first-place');
    } else if (rank === 2 || rank === 3 || rank === 4 || rank === 5) {
      teamElement.classList.add('podium-place');
    } else {
      teamElement.classList.add('general-place');
    }

    teamElement.innerHTML = `
            <div class="team-rank">${rank}位</div>
            <div class="team-info">
                <div class="team-name">${team.name}</div>
                <div class="team-score">${team.score}点</div>
                <div class="team-members">
                    ${team.members
                      .map(
                        (member) =>
                          `<span class="member-name">${member.nickname}</span>`
                      )
                      .join('')}
                </div>
            </div>
        `;

    // column-reverseにより、appendChildで追加すると視覚的には上に表示される
    container.appendChild(teamElement);

    // // results-screenを一番上にスクロール
    // if (this.elements.resultsScreen) {
    //   this.elements.resultsScreen.scrollTop = 0;
    // }
    // containerも一番上にスクロール
    container.scrollTo(0, -1000000);
    // container.scrollTop = 0;

    // スライドインアニメーションを適用
    setTimeout(() => {
      teamElement.classList.add('slide-in');
    }, 50);

    // 次のチームの表示間隔を設定
    const delay = rank <= 6 ? 4000 : 1500; // 3位以上は3秒、4位以下は1秒

    setTimeout(() => {
      this.displayTeamsSequentially(teams, container, index + 1);
    }, delay);
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
      const reactions =
        this.elements.emojiReactions.querySelectorAll('.emoji-reaction');
      if (reactions.length > 20) {
        for (let i = 0; i < reactions.length - 20; i++) {
          reactions[i].remove();
        }
      }
    }, 1000);
  }

  showCountdown(secondsLeft) {
    if (secondsLeft > 10 || secondsLeft < 0) {
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
      }, 1300);
    } else {
      // Show count
      setTimeout(() => {
        this.showCountdown(secondsLeft - 1);
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
    this.elements.timeUpDisplay.classList.add('transparent-overlay');

    // カウントダウン終了時の音声再生
    this.playAudio('time_up.mp3');

    // Hide after 5 seconds
    setTimeout(() => {
      this.elements.timeUpDisplay.classList.add('hidden');
      this.elements.timeUpDisplay.classList.remove('transparent-overlay');
    }, 5000);
  }

  showTitleScreen(data) {
    // this.hideAllScreens();
    // // Create or show title display screen
    // let titleScreen = document.getElementById('title-screen');
    // if (!titleScreen) {
    //   titleScreen = document.createElement('div');
    //   titleScreen.id = 'title-screen';
    //   titleScreen.className = 'screen-section';
    //   titleScreen.innerHTML = `
    //             <div class="title-display">
    //                 <h1 class="main-title">${data.title}</h1>
    //                 <p class="welcome-message"></p>
    //             </div>
    //         `;
    //   document.querySelector('.screen-content').appendChild(titleScreen);
    // }
    // titleScreen.classList.remove('hidden');
    // this.elements.headerWaiting.classList.add('header-waiting');
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
                    <!-- <h2>🏆 チーム発表</h2> -->
                    <div id="team-assignment-list" class="teams-display">
                        <!-- チーム一覧がここに表示されます -->
                    </div>
                </div>
            `;
      document.querySelector('.screen-content').appendChild(teamScreen);
    }

    const teamList = teamScreen.querySelector('#team-assignment-list');
    teamList.innerHTML = '';

    teams.forEach((team, _index) => {
      const teamDiv = document.createElement('div');
      teamDiv.className = 'team-item';
      teamDiv.innerHTML = `
                <div class="team-header">
                    <h3>${team.name}</h3>
                </div>
                <div class="team-members">
                    ${team.members
                      .map(
                        (member) => `
                        <div class="member-name">${member.nickname}</div>
                    `
                      )
                      .join('')}
                </div>
            `;
      teamList.appendChild(teamDiv);
    });

    teamScreen.classList.remove('hidden');
  }

  showAnswerStatsScreen(data) {
    // この情報を問題画面に重ねて表示
    this.elements.questionScreen.classList.remove('hidden');

    // 各選択肢に回答人数を表示
    this.showChoicesWithCounts(data.total_participants, data.choices_counts);
  }

  showAnswerRevealScreen(data) {
    // 問題画面で正解をハイライト表示
    this.elements.questionScreen.classList.remove('hidden');

    const choices =
      this.elements.choicesDisplay.querySelectorAll('.choice-display');
    choices.forEach((choice, index) => {
      choice.classList.remove('correct', 'revealed');
      if (index === data.correct - 1) {
        choice.classList.add('correct', 'revealed');
      }
    });

    // 正答発表時の音声再生
    this.playAudio('answer_reveal.mp3');

    // 1秒後に画像を表示（設定があれば）
    setTimeout(() => {
      this.showAnswerRevealImage(data.correct);
    }, 1000);
  }

  /**
   * 正解発表時の画像を表示する
   * @param {number} correctIndex - 正解の選択肢インデックス（1-based）
   */
  showAnswerRevealImage(correctIndex) {
    if (!this.currentQuestion || !this.currentQuestion.question) {
      return;
    }

    const question = this.currentQuestion.question;
    const imageFileName = question.image;

    // 画像設定がない、または空文字列の場合は何もしない
    if (!imageFileName || imageFileName.trim() === '') {
      return;
    }

    // 画像パスを設定
    this.elements.answerRevealImage.src = `/images/${imageFileName}`;

    // 正解の選択肢の位置に応じて、画像の表示位置を決定
    // 選択肢は2x2グリッド: [0: 左上, 1: 右上, 2: 左下, 3: 右下]
    // 正解が左側（0, 2）なら右側に画像を表示、右側（1, 3）なら左側に画像を表示
    const correctIndexZeroBased = correctIndex - 1; // Convert to 0-based
    const isCorrectOnLeft = correctIndexZeroBased % 2 === 0;

    this.elements.answerRevealImageContainer.classList.remove(
      'position-left',
      'position-right'
    );
    if (isCorrectOnLeft) {
      // 正解が左側なので、画像は右側に表示
      this.elements.answerRevealImageContainer.classList.add('position-right');
    } else {
      // 正解が右側なので、画像は左側に表示
      this.elements.answerRevealImageContainer.classList.add('position-left');
    }

    // 画像を表示
    this.elements.answerRevealImageContainer.classList.remove('hidden');
  }

  /**
   * 正解発表時の画像を非表示にする
   */
  hideAnswerRevealImage() {
    if (this.elements.answerRevealImageContainer) {
      this.elements.answerRevealImageContainer.classList.add('hidden');
    }
  }

  showCelebrationScreen() {
    this.hideAllScreens();
    // 結果画面を表示してクラッカーアニメーション開始
    this.elements.resultsScreen.classList.remove('hidden');
    // this.startConfettiAnimation();
  }

  // startConfettiAnimation() {
  //     // 左右のクラッカーを作成
  //     const leftCracker = this.createCracker('left');
  //     const rightCracker = this.createCracker('right');

  //     document.body.appendChild(leftCracker);
  //     document.body.appendChild(rightCracker);

  //     // 紙吹雪アニメーション開始
  //     setTimeout(() => {
  //         this.createConfetti('left');
  //         this.createConfetti('right');
  //     }, 500);

  //     // 5秒後にクリーンアップ
  //     setTimeout(() => {
  //         leftCracker.remove();
  //         rightCracker.remove();
  //         this.clearConfetti();
  //     }, 5000);
  // }

  // // FIXME: 形を整える
  // createCracker(side) {
  //     const cracker = document.createElement('div');
  //     cracker.className = `cracker cracker-${side}`;
  //     cracker.style.cssText = `
  //         position: fixed;
  //         ${side}: 20px;
  //         top: 50%;
  //         width: 60px;
  //         height: 120px;
  //         background: linear-gradient(45deg, #FFD700, #FFA500);
  //         border-radius: 10px 10px 30px 30px;
  //         z-index: 1000;
  //         transform: translateY(-50%);
  //         animation: crackerShake 0.5s ease-in-out;
  //     `;
  //     return cracker;
  // }

  // // FIXME: ちゃんとした方向に飛ばす
  // createConfetti(side) {
  //     const colors = ['#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8', '#FFD93D'];
  //     const startX = side === 'left' ? 50 : window.innerWidth - 50;

  //     for (let i = 0; i < 50; i++) {
  //         const confetti = document.createElement('div');
  //         confetti.className = 'confetti';
  //         confetti.style.cssText = `
  //             position: fixed;
  //             left: ${startX}px;
  //             top: 50%;
  //             width: 10px;
  //             height: 10px;
  //             background: ${colors[Math.floor(Math.random() * colors.length)]};
  //             border-radius: ${Math.random() > 0.5 ? '50%' : '0'};
  //             z-index: 999;
  //             pointer-events: none;
  //             animation: confettiFall ${2 + Math.random() * 3}s linear forwards;
  //             animation-delay: ${Math.random() * 2}s;
  //             transform: rotate(${Math.random() * 360}deg);
  //         `;

  //         // ランダムな方向に飛ばす
  //         const angle = (side === 'left' ? 0.3 : -2.8) + (Math.random() - 0.5) * 0.8;
  //         const velocity = 100 + Math.random() * 200;
  //         const endX = startX + Math.cos(angle) * velocity;
  //         const endY = window.innerHeight + 100;

  //         confetti.style.setProperty('--end-x', `${endX}px`);
  //         confetti.style.setProperty('--end-y', `${endY}px`);

  //         document.body.appendChild(confetti);
  //     }
  // }

  startFullScreenConfetti() {
    // 画面全体から紙吹雪を降らせるコンテナを作成
    const confettiContainer = document.createElement('div');
    confettiContainer.className = 'full-screen-confetti';
    confettiContainer.id = 'full-screen-confetti-container';
    document.body.appendChild(confettiContainer);

    const colors = [
      '#FF6B6B',
      '#4ECDC4',
      '#45B7D1',
      '#FFA07A',
      '#98D8C8',
      '#FFD93D',
      '#FFB6C1',
      '#87CEEB',
      '#DDA0DD',
      '#F0E68C',
    ];
    const shapes = ['circle', 'square', 'triangle'];

    // 10秒間継続的に紙吹雪を生成
    const confettiInterval = setInterval(() => {
      this.createFullScreenConfettiPieces(confettiContainer, colors, shapes);
    }, 200); // 200ms間隔で新しい紙吹雪を生成

    // 10秒後に停止
    setTimeout(() => {
      clearInterval(confettiInterval);

      // さらに5秒後にコンテナを削除（落下アニメーション完了待ち）
      setTimeout(() => {
        if (confettiContainer.parentNode) {
          confettiContainer.remove();
        }
      }, 5000);
    }, 10000);
  }

  createFullScreenConfettiPieces(container, colors, shapes) {
    const piecesPerBatch = 15; // 一度に生成する紙吹雪の数

    for (let i = 0; i < piecesPerBatch; i++) {
      const confetti = document.createElement('div');
      const shape = shapes[Math.floor(Math.random() * shapes.length)];
      const color = colors[Math.floor(Math.random() * colors.length)];

      confetti.className = `confetti-piece confetti-${shape}`;
      confetti.style.backgroundColor = color;
      confetti.style.color = color; // triangleの場合に使用

      // 画面上部からランダムな横位置で開始
      const startX = Math.random() * window.innerWidth;
      const rotation = Math.random() * 720 + 360; // 1-2回転

      // 空気抵抗を表現する左右の揺れを設定（6段階の揺れポイント）
      const swayAmplitude = 15 + Math.random() * 25; // 揺れの振幅（15-40px）
      const sway1 = (Math.random() - 0.5) * swayAmplitude;
      const sway2 = (Math.random() - 0.5) * swayAmplitude;
      const sway3 = (Math.random() - 0.5) * swayAmplitude;
      const sway4 = (Math.random() - 0.5) * swayAmplitude;
      const sway5 = (Math.random() - 0.5) * swayAmplitude;
      const sway6 = (Math.random() - 0.5) * swayAmplitude;
      const finalDrift = (Math.random() - 0.5) * 100; // 最終的な横方向ドリフト

      confetti.style.left = `${startX}px`;
      confetti.style.top = '-20px';

      // 各段階の揺れを設定
      confetti.style.setProperty('--sway-1', `${sway1}px`);
      confetti.style.setProperty('--sway-2', `${sway2}px`);
      confetti.style.setProperty('--sway-3', `${sway3}px`);
      confetti.style.setProperty('--sway-4', `${sway4}px`);
      confetti.style.setProperty('--sway-5', `${sway5}px`);
      confetti.style.setProperty('--sway-6', `${sway6}px`);
      confetti.style.setProperty('--drift-x', `${finalDrift}px`);
      confetti.style.setProperty('--rotation', `${rotation}deg`);

      // アニメーション時間をランダム化（3-5秒）
      confetti.style.animationDuration = `${3 + Math.random() * 2}s`;
      confetti.style.animationDelay = `${Math.random() * 0.5}s`;

      container.appendChild(confetti);

      // 紙吹雪が画面外に出たら削除
      setTimeout(() => {
        if (confetti.parentNode) {
          confetti.remove();
        }
      }, 6000);
    }
  }

  clearConfetti() {
    document
      .querySelectorAll('.confetti')
      .forEach((confetti) => confetti.remove());
  }

  blockAnswers() {
    this.answersBlocked = true;
    // No direct action needed here since this is the screen display
    // The participant.js handles answer blocking
  }

  handleStateChanged(data) {
    console.log('State changed:', data.new_state);

    // Handle state-specific transitions using constants
    const { EVENT_STATES } = QuizConstants;

    switch (data.new_state) {
      case EVENT_STATES.WAITING:
        this.showWaitingScreen();
        break;

      case EVENT_STATES.TITLE_DISPLAY:
        this.handleTitleDisplay({
          // title: this.elements.eventTitle.textContent,
        });
        break;

      case EVENT_STATES.TEAM_ASSIGNMENT:
        // Trigger team display if teams exist
        this.loadStatus(); // This will reload teams
        break;

      case EVENT_STATES.QUESTION_ACTIVE:
        if (data.question) {
          // Set current question and display it
          this.currentQuestion = {
            question_number: data.question_number,
            question: data.question,
            total_questions: data.total_questions,
          };
          this.handleQuestionStart(this.currentQuestion);
        } else if (data.question_number > 0) {
          // Load question from API if not provided
          this.loadQuestionFromAPI(data.question_number);
        }
        break;

      case EVENT_STATES.COUNTDOWN_ACTIVE:
        // Countdown will be handled by separate countdown messages
        break;

      case EVENT_STATES.ANSWER_STATS:
        // this.hideAllScreens();
        // this.elements.answerStats.style.display = 'block';
        // NOTE: NO IMPLEMENTED
        break;

      case EVENT_STATES.ANSWER_REVEAL:
        // Show question screen with answer revealed
        if (this.currentQuestion) {
          this.showQuestionScreen();
        }
        break;

      case EVENT_STATES.RESULTS:
        this.handleFinalResults({ results: [], teams: [], team_mode: false });
        break;

      case EVENT_STATES.CELEBRATION:
        this.handleCelebration({});
        break;

      case EVENT_STATES.FINISHED:
        this.showWaitingScreen();
        break;
    }
  }

  async loadQuestionFromAPI(questionNumber) {
    try {
      const response = await fetch('/api/status');
      if (response.ok) {
        const data = await response.json();
        if (
          data.config &&
          data.config.questions &&
          questionNumber <= data.config.questions.length
        ) {
          const question = data.config.questions[questionNumber - 1];
          this.currentQuestion = {
            question_number: questionNumber,
            question: question,
            total_questions: data.config.questions.length,
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
      this.elements.connectionStatus.className =
        'connection-status disconnected';
      this.elements.connectionText.textContent = '接続中...';
    }
  }
}

document.addEventListener('DOMContentLoaded', () => {
  new QuizScreen();
});
