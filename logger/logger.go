package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents different log levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time            `json:"timestamp"`
	Level     string               `json:"level"`
	Message   string               `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Source    string               `json:"source,omitempty"`
}

// LogChan is a globally available channel for streaming log messages.
var (
	LogChan     chan string      // Used to stream logs to WebSocket clients
	logLock     sync.Mutex       // Protects concurrent access to log resources
	logFile     *os.File         // File handle for writing logs to disk
	logFilePath = "nyatictl.log" // Default log file path; override using SetLogFilePath()
	currentLevel LogLevel = INFO  // Current minimum log level
	structuredLogging bool = false // Whether to use structured JSON logging
)

// SetLogFilePath overrides the default log file path.
//
// Must be called before Init() to take effect.
//
// Parameters:
//   - path: absolute or relative path to the log file (e.g. "logs/out.log")
func SetLogFilePath(path string) {
	logFilePath = path
}

// SetLogLevel sets the minimum log level
func SetLogLevel(level LogLevel) {
	logLock.Lock()
	defer logLock.Unlock()
	currentLevel = level
}

// EnableStructuredLogging enables JSON-formatted structured logging
func EnableStructuredLogging(enabled bool) {
	logLock.Lock()
	defer logLock.Unlock()
	structuredLogging = enabled
}

// Init sets up the logging system.
//
// Responsibilities:
//  1. Initializes LogChan for in-memory log streaming.
//  2. Ensures the directory for logFilePath exists (creates if missing).
//  3. Opens or creates the log file in append mode.
//  4. Makes logging via Log() available throughout the app.
//
// Returns:
//   - error: if directory creation or file opening fails
func Init() error {
	logLock.Lock()
	defer logLock.Unlock()

	// Step 1: Create log streaming channel
	if LogChan == nil {
		LogChan = make(chan string, 100)
	}

	// Step 2: Ensure the log directory exists
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create log directory %s: %v", logDir, err)
	}

	// Step 3: Open or create the log file for writing (append mode)
	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %v", logFilePath, err)
	}

	return nil
}

// Log sends a message to the global LogChan and also writes it to the log file.
// This function is safe for concurrent use and non-blocking.
// Parameters:
//   - msg: the log message to emit
func Log(msg string) {
	LogWithLevel(INFO, msg, nil)
}

// LogWithLevel logs a message with a specific level and optional fields
func LogWithLevel(level LogLevel, msg string, fields map[string]interface{}) {
	logLock.Lock()
	defer logLock.Unlock()

	// Skip if below current log level
	if level < currentLevel {
		return
	}

	var logMessage string
	if structuredLogging {
		entry := LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     level.String(),
			Message:   msg,
			Fields:    fields,
		}
		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			// Fallback to plain text if JSON marshaling fails
			logMessage = fmt.Sprintf("[%s] %s %s", time.Now().UTC().Format(time.RFC3339), level.String(), msg)
		} else {
			logMessage = string(jsonBytes)
		}
	} else {
		logMessage = fmt.Sprintf("[%s] %s %s", time.Now().UTC().Format(time.RFC3339), level.String(), msg)
	}

	// Send to in-memory channel (if initialized)
	if LogChan != nil {
		select {
		case LogChan <- logMessage:
		default:
			// Channel full — drop message to avoid blocking
		}
	}

	// Append message to log file (if initialized)
	if logFile != nil {
		if _, err := logFile.WriteString(logMessage + "\n"); err != nil {
			// Log the error to standard error to avoid infinite recursion
			log.Printf("Failed to write to log file: %v", err)
		}
	}
}

// Convenience functions for different log levels

// Debug logs a debug message
func Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	LogWithLevel(DEBUG, msg, f)
}

// Info logs an info message
func Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	LogWithLevel(INFO, msg, f)
}

// Warn logs a warning message
func Warn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	LogWithLevel(WARN, msg, f)
}

// Error logs an error message
func Error(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	LogWithLevel(ERROR, msg, f)
}

// Fatal logs a fatal message
func Fatal(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	LogWithLevel(FATAL, msg, f)
}

// Close closes the log file handle and cleans up resources
func Close() error {
	logLock.Lock()
	defer logLock.Unlock()

	if logFile != nil {
		err := logFile.Close()
		logFile = nil
		return err
	}
	return nil
}

// GetLogLevel returns the current log level
func GetLogLevel() LogLevel {
	logLock.Lock()
	defer logLock.Unlock()
	return currentLevel
}

// IsStructuredLoggingEnabled returns whether structured logging is enabled
func IsStructuredLoggingEnabled() bool {
	logLock.Lock()
	defer logLock.Unlock()
	return structuredLogging
}
