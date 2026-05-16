package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarn
	LogLevelError
	LogLevelAudit
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel `json:"level"`
	Message   string   `json:"message"`
	Command   string   `json:"command,omitempty"`
	User      string   `json:"user,omitempty"`
	Error     string   `json:"error,omitempty"`
	Context   string   `json:"context,omitempty"`
}

// Logger handles structured logging for the application
type Logger struct {
	logFile *os.File
	user    string
}

// NewLogger creates a new logger instance
func NewLogger() (*Logger, error) {
	// Create logs directory if it doesn't exist
	logDir := filepath.Join(os.TempDir(), "gitbook", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("gitbook-%s.log", timestamp))
	
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Get current user for audit trail
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	if user == "" {
		user = "unknown"
	}

	return &Logger{
		logFile: logFile,
		user:    user,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// log writes a structured log entry
func (l *Logger) log(level LogLevel, message, command, context, error string) {
	if l == nil || l.logFile == nil {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
		Command:   command,
		User:      l.user,
		Error:     error,
		Context:   context,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON marshaling fails
		l.logFile.WriteString(fmt.Sprintf("[%s] ERROR: Failed to marshal log entry: %v\n", 
			time.Now().Format(time.RFC3339), err))
		return
	}

	l.logFile.WriteString(string(jsonData) + "\n")
	l.logFile.Sync() // Ensure log is written to disk immediately
}

// Info logs an informational message
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message, "", "", "")
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(LogLevelWarn, message, "", "", "")
}

// Error logs an error message
func (l *Logger) Error(message, err string) {
	l.log(LogLevelError, message, "", "", err)
}

// Audit logs an audit trail entry for important actions
func (l *Logger) Audit(command, context string) {
	l.log(LogLevelAudit, "Command executed", command, context, "")
}

// AuditError logs an audit trail entry for failed actions
func (l *Logger) AuditError(command, context, errMsg string) {
	l.log(LogLevelAudit, "Command failed", command, context, errMsg)
}

// Global logger instance
var globalLogger *Logger

// InitLogger initializes the global logger
func InitLogger() error {
	var err error
	globalLogger, err = NewLogger()
	return err
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	return globalLogger
}

// CleanupLogger closes the global logger
func CleanupLogger() {
	if globalLogger != nil {
		globalLogger.Close()
	}
}
