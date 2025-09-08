/**
 * Common constants for the quiz application
 * This file contains shared constants used across all frontend components
 */

// Event State Constants - matches models/constants.go
const EVENT_STATES = {
    WAITING: 'waiting',
    STARTED: 'started',
    TITLE_DISPLAY: 'title_display',
    TEAM_ASSIGNMENT: 'team_assignment',
    QUESTION_ACTIVE: 'question_active',
    COUNTDOWN_ACTIVE: 'countdown_active',
    ANSWER_STATS: 'answer_stats',
    ANSWER_REVEAL: 'answer_reveal',
    RESULTS: 'results',
    CELEBRATION: 'celebration',
    FINISHED: 'finished'
};

// Japanese labels for states - matches models/constants.go StateLabels
const STATE_LABELS = {
    [EVENT_STATES.WAITING]: '参加者待ち',
    [EVENT_STATES.STARTED]: 'イベント開始',
    [EVENT_STATES.TITLE_DISPLAY]: 'タイトル表示',
    [EVENT_STATES.TEAM_ASSIGNMENT]: 'チーム分け',
    [EVENT_STATES.QUESTION_ACTIVE]: '問題表示中',
    [EVENT_STATES.COUNTDOWN_ACTIVE]: 'カウントダウン中',
    [EVENT_STATES.ANSWER_STATS]: '回答状況表示',
    [EVENT_STATES.ANSWER_REVEAL]: '回答発表',
    [EVENT_STATES.RESULTS]: '結果発表',
    [EVENT_STATES.CELEBRATION]: 'お疲れ様画面',
    [EVENT_STATES.FINISHED]: '終了'
};

// WebSocket Message Types - matches websocket/message_types.go
const MESSAGE_TYPES = {
    // Event messages
    EVENT_STARTED: 'event_started',
    TITLE_DISPLAY: 'title_display',
    TEAM_ASSIGNMENT: 'team_assignment',
    QUESTION_START: 'question_start',
    QUESTION_END: 'question_end',
    FINAL_RESULTS: 'final_results',
    CELEBRATION: 'celebration',
    
    // User interaction messages
    USER_JOINED: 'user_joined',
    USER_LEFT: 'user_left',
    ANSWER_RECEIVED: 'answer_received',
    EMOJI_REACTION: 'emoji',
    TEAM_MEMBER_ADDED: 'team_member_added',
    
    // Quiz progress messages
    COUNTDOWN: 'countdown',
    ANSWER_STATS: 'answer_stats',
    ANSWER_REVEAL: 'answer_reveal',
    STATE_CHANGED: 'state_changed',
    
    // Legacy/deprecated
    TIME_ALERT: 'time_alert' // DEPRECATED: use countdown instead
};

// Admin Actions - matches the available actions from EventStateManager
const ADMIN_ACTIONS = {
    START_EVENT: 'start_event',
    SHOW_TITLE: 'show_title',
    ASSIGN_TEAMS: 'assign_teams',
    NEXT_QUESTION: 'next_question',
    COUNTDOWN_ALERT: 'countdown_alert',
    SHOW_ANSWER_STATS: 'show_answer_stats',
    REVEAL_ANSWER: 'reveal_answer',
    SHOW_RESULTS: 'show_results',
    CELEBRATION: 'celebration'
};

// Client Types - matches websocket/websocket.go ClientType
const CLIENT_TYPES = {
    PARTICIPANT: 'participant',
    ADMIN: 'admin',
    SCREEN: 'screen'
};

// Connection Status
const CONNECTION_STATUS = {
    CONNECTED: 'connected',
    DISCONNECTED: 'disconnected',
    CONNECTING: 'connecting'
};

// Log Types
const LOG_TYPES = {
    INFO: 'info',
    SUCCESS: 'success',
    WARNING: 'warning',
    ERROR: 'error'
};

// Export constants
if (typeof module !== 'undefined' && module.exports) {
    // Node.js environment
    module.exports = {
        EVENT_STATES,
        STATE_LABELS,
        MESSAGE_TYPES,
        ADMIN_ACTIONS,
        CLIENT_TYPES,
        CONNECTION_STATUS,
        LOG_TYPES
    };
} else {
    // Browser environment - make constants available globally
    window.QuizConstants = {
        EVENT_STATES,
        STATE_LABELS,
        MESSAGE_TYPES,
        ADMIN_ACTIONS,
        CLIENT_TYPES,
        CONNECTION_STATUS,
        LOG_TYPES
    };
}