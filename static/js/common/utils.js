/**
 * Common utility functions for the quiz application
 * This file contains shared utility functions used across all frontend components
 */

// State utilities
const StateUtils = {
    /**
     * Get Japanese label for a state
     * @param {string} state - The state value
     * @returns {string} Japanese label or original state if not found
     */
    getStateLabel(state) {
        const constants = window.QuizConstants || (typeof require !== 'undefined' ? require('./constants') : {});
        return constants.STATE_LABELS?.[state] || state;
    },

    /**
     * Check if a state is valid
     * @param {string} state - The state to validate
     * @returns {boolean} True if valid
     */
    isValidState(state) {
        const constants = window.QuizConstants || (typeof require !== 'undefined' ? require('./constants') : {});
        return Object.values(constants.EVENT_STATES || {}).includes(state);
    },

    /**
     * Get all states with labels for dropdowns
     * @returns {Array} Array of {value, label} objects
     */
    getAllStatesWithLabels() {
        const constants = window.QuizConstants || (typeof require !== 'undefined' ? require('./constants') : {});
        return Object.values(constants.EVENT_STATES || {}).map(state => ({
            value: state,
            label: this.getStateLabel(state)
        }));
    }
};

// Message utilities
const MessageUtils = {
    /**
     * Check if a message type is valid
     * @param {string} msgType - The message type to validate
     * @returns {boolean} True if valid
     */
    isValidMessageType(msgType) {
        const constants = window.QuizConstants || (typeof require !== 'undefined' ? require('./constants') : {});
        return Object.values(constants.MESSAGE_TYPES || {}).includes(msgType);
    },

    /**
     * Check if a message type is deprecated
     * @param {string} msgType - The message type to check
     * @returns {boolean} True if deprecated
     */
    isDeprecatedMessageType(msgType) {
        const constants = window.QuizConstants || (typeof require !== 'undefined' ? require('./constants') : {});
        const deprecated = [constants.MESSAGE_TYPES?.TIME_ALERT];
        return deprecated.includes(msgType);
    },

    /**
     * Create a standard message object
     * @param {string} type - Message type
     * @param {any} data - Message data
     * @param {object} options - Optional fields (userID, target)
     * @returns {object} Message object
     */
    createMessage(type, data, options = {}) {
        const message = {
            type,
            data
        };
        
        if (options.userID !== undefined) {
            message.user_id = options.userID;
        }
        
        if (options.target !== undefined) {
            message.target = options.target;
        }
        
        return message;
    }
};

// Error handling utilities
const ErrorUtils = {
    /**
     * Handle API errors with consistent error messages
     * @param {Error|Response} error - Error object or fetch response
     * @param {string} context - Context of the error (e.g., 'joining quiz')
     * @returns {Promise<string>} Error message
     */
    async handleApiError(error, context) {
        let message = `${context}中にエラーが発生しました`;
        
        try {
            if (error instanceof Response) {
                const data = await error.json();
                message += `: ${data.error || data.message || 'Unknown error'}`;
            } else if (error instanceof Error) {
                message += `: ${error.message}`;
            } else {
                message += `: ${String(error)}`;
            }
        } catch (parseError) {
            // If we can't parse the error, use the default message
            console.error('Error parsing API error:', parseError);
        }
        
        return message;
    },

    /**
     * Log error with context
     * @param {string} context - Context of the error
     * @param {Error} error - Error object
     * @param {object} additionalData - Additional data to log
     */
    logError(context, error, additionalData = {}) {
        console.error(`[${context}] Error:`, error);
        if (Object.keys(additionalData).length > 0) {
            console.error(`[${context}] Additional data:`, additionalData);
        }
    }
};

// Validation utilities
const ValidationUtils = {
    /**
     * Validate nickname
     * @param {string} nickname - Nickname to validate
     * @returns {object} {valid: boolean, message: string}
     */
    validateNickname(nickname) {
        if (!nickname || !nickname.trim()) {
            return { valid: false, message: 'ニックネームを入力してください' };
        }
        
        if (nickname.length > 20) {
            return { valid: false, message: 'ニックネームは20文字以内で入力してください' };
        }
        
        // Check for potentially problematic characters
        if (nickname.includes('<') || nickname.includes('>')) {
            return { valid: false, message: 'ニックネームに使用できない文字が含まれています' };
        }
        
        return { valid: true, message: '' };
    },

    /**
     * Validate question number
     * @param {number} questionNumber - Question number to validate
     * @param {number} totalQuestions - Total number of questions
     * @returns {object} {valid: boolean, message: string}
     */
    validateQuestionNumber(questionNumber, totalQuestions) {
        if (questionNumber < 0 || questionNumber > totalQuestions) {
            return { 
                valid: false, 
                message: `問題番号は0〜${totalQuestions}の範囲で入力してください` 
            };
        }
        
        return { valid: true, message: '' };
    }
};

// DOM utilities
const DOMUtils = {
    /**
     * Safely get element by ID
     * @param {string} id - Element ID
     * @returns {HTMLElement|null} Element or null if not found
     */
    getElementById(id) {
        const element = document.getElementById(id);
        if (!element) {
            console.warn(`Element with ID '${id}' not found`);
        }
        return element;
    },

    /**
     * Show/hide elements with class management
     * @param {HTMLElement} element - Element to show/hide
     * @param {boolean} show - Whether to show (true) or hide (false)
     * @param {string} hiddenClass - CSS class for hidden state (default: 'hidden')
     */
    toggleElement(element, show, hiddenClass = 'hidden') {
        if (!element) return;
        
        if (show) {
            element.classList.remove(hiddenClass);
        } else {
            element.classList.add(hiddenClass);
        }
    },

    /**
     * Create and add log entry
     * @param {HTMLElement} container - Log container element
     * @param {string} message - Log message
     * @param {string} type - Log type (info, success, warning, error)
     */
    addLogEntry(container, message, type = 'info') {
        if (!container) return;
        
        const timestamp = new Date().toLocaleTimeString();
        const logEntry = document.createElement('div');
        logEntry.className = 'log-entry';
        
        logEntry.innerHTML = `
            <span class="log-timestamp">${timestamp}</span>
            <span class="log-type-${type}">${message}</span>
        `;
        
        container.appendChild(logEntry);
        container.scrollTop = container.scrollHeight;
    }
};

// Time utilities
const TimeUtils = {
    /**
     * Format timestamp to locale string
     * @param {Date|number} timestamp - Timestamp to format
     * @returns {string} Formatted time string
     */
    formatTime(timestamp) {
        return new Date(timestamp).toLocaleTimeString();
    },

    /**
     * Calculate time difference in seconds
     * @param {Date|number} start - Start time
     * @param {Date|number} end - End time
     * @returns {number} Difference in seconds
     */
    getTimeDifferenceInSeconds(start, end) {
        return Math.floor((new Date(end) - new Date(start)) / 1000);
    }
};

// Export utilities
if (typeof module !== 'undefined' && module.exports) {
    // Node.js environment
    module.exports = {
        StateUtils,
        MessageUtils,
        ErrorUtils,
        ValidationUtils,
        DOMUtils,
        TimeUtils
    };
} else {
    // Browser environment - make utilities available globally
    window.QuizUtils = {
        StateUtils,
        MessageUtils,
        ErrorUtils,
        ValidationUtils,
        DOMUtils,
        TimeUtils
    };
}