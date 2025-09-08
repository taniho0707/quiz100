package models

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarning
	LogLevelError
	LogLevelDebug
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

type QuizLogger struct {
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
	debugLogger   *log.Logger
	logFile       *os.File
	logDir        string
}

func NewQuizLogger(logDir string) (*QuizLogger, error) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("quiz_%s.log", timestamp)
	logPath := filepath.Join(logDir, filename)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %v", err)
	}

	// Create multi-writer to write to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	return &QuizLogger{
		infoLogger:    log.New(multiWriter, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		warningLogger: log.New(multiWriter, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger:   log.New(multiWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		debugLogger:   log.New(multiWriter, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
		logFile:       logFile,
		logDir:        logDir,
	}, nil
}

func (ql *QuizLogger) Close() error {
	if ql.logFile != nil {
		return ql.logFile.Close()
	}
	return nil
}

func (ql *QuizLogger) Info(format string, v ...any) {
	ql.infoLogger.Printf(format, v...)
}

func (ql *QuizLogger) Warning(format string, v ...any) {
	ql.warningLogger.Printf(format, v...)
}

func (ql *QuizLogger) Error(format string, v ...any) {
	ql.errorLogger.Printf(format, v...)
}

func (ql *QuizLogger) Debug(format string, v ...any) {
	ql.debugLogger.Printf(format, v...)
}

// Event-specific logging methods
func (ql *QuizLogger) LogEventStart(title string, teamMode bool, participantCount int) {
	ql.Info("=== EVENT STARTED ===")
	ql.Info("Title: %s", title)
	ql.Info("Team Mode: %v", teamMode)
	ql.Info("Initial Participants: %d", participantCount)
}

func (ql *QuizLogger) LogEventEnd(title string, participantCount int, totalQuestions int) {
	ql.Info("=== EVENT ENDED ===")
	ql.Info("Title: %s", title)
	ql.Info("Final Participants: %d", participantCount)
	ql.Info("Total Questions: %d", totalQuestions)
}

func (ql *QuizLogger) LogUserJoin(nickname string, sessionID string) {
	ql.Info("User joined: %s (Session: %s)", nickname, sessionID[:8])
}

func (ql *QuizLogger) LogUserReconnect(nickname string, sessionID string) {
	ql.Info("User reconnected: %s (Session: %s)", nickname, sessionID[:8])
}

func (ql *QuizLogger) LogUserSessionReset(nickname string, sessionID string) {
	ql.Info("User session reset: %s (Session: %s)", nickname, sessionID[:8])
}

func (ql *QuizLogger) LogQuestionStart(questionNumber int, questionText string) {
	ql.Info("--- QUESTION %d STARTED ---", questionNumber)
	ql.Info("Question: %s", questionText)
}

func (ql *QuizLogger) LogAnswer(nickname string, questionNumber int, answerIndex int, isCorrect bool) {
	status := "INCORRECT"
	if isCorrect {
		status = "CORRECT"
	}
	ql.Info("Answer received: %s - Q%d, Choice %d (%s)", nickname, questionNumber, answerIndex, status)
}

func (ql *QuizLogger) LogTeamAssignment(teamCount int, totalUsers int) {
	ql.Info("=== TEAMS CREATED ===")
	ql.Info("Number of teams: %d", teamCount)
	ql.Info("Total users assigned: %d", totalUsers)
}

func (ql *QuizLogger) LogTeamResult(teamName string, score int, members []string) {
	ql.Info("Team Result: %s - Score: %d, Members: %v", teamName, score, members)
}

func (ql *QuizLogger) LogEmojiReaction(nickname string, emoji string) {
	ql.Debug("Emoji reaction: %s sent %s", nickname, emoji)
}

func (ql *QuizLogger) LogWebSocketConnection(clientType string, userInfo string) {
	ql.Info("WebSocket connected: %s (%s)", clientType, userInfo)
}

func (ql *QuizLogger) LogWebSocketDisconnection(clientType string, userInfo string) {
	ql.Info("WebSocket disconnected: %s (%s)", clientType, userInfo)
}

func (ql *QuizLogger) LogError(operation string, err error) {
	ql.Error("Error in %s: %v", operation, err)
}

func (ql *QuizLogger) LogAlert(message string) {
	ql.Info("ALERT: %s", message)
}

func (ql *QuizLogger) LogStateTransition(from, to EventState) {
	ql.Info("State transition: %s -> %s", from, to)
}
