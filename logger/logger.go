package logger

import (
	"log"
	"os"
	"path/filepath"
	"sync"
)

// LogChan is a globally available channel for streaming log messages.
var (
	LogChan     chan string      // Used to stream logs to WebSocket clients
	logLock     sync.Mutex       // Protects concurrent access to log resources
	logFile     *os.File         // File handle for writing logs to disk
	logFilePath = "nyatictl.log" // Default log file path; override using SetLogFilePath()
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

// Init sets up the logging system.
//
// Responsibilities:
//  1. Initializes LogChan for in-memory log streaming.
//  2. Ensures the directory for logFilePath exists (creates if missing).
//  3. Opens or creates the log file in append mode.
//  4. Makes logging via Log() available throughout the app.
func Init() {
	logLock.Lock()
	defer logLock.Unlock()

	// Step 1: Create log streaming channel
	if LogChan == nil {
		LogChan = make(chan string, 100)
	}

	// Step 2: Ensure the log directory exists
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatalf("logger: failed to create log directory %s: %v", logDir, err)
	}

	// Step 3: Open or create the log file for writing (append mode)
	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("logger: failed to open log file %s: %v", logFilePath, err)
	}
}

// Log sends a message to the global LogChan and also writes it to the log file.
//
// This function is safe for concurrent use and non-blocking.
//
// Parameters:
//   - msg: the log message to emit
func Log(msg string) {
	logLock.Lock()
	defer logLock.Unlock()

	// Send to in-memory channel (if initialized)
	if LogChan != nil {
		select {
		case LogChan <- msg:
		default:
			// Channel full — drop message to avoid blocking
		}
	}

	// Append message to log file (if initialized)
	if logFile != nil {
		logFile.WriteString(msg + "\n")
	}
}
