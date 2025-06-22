package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSetLogFilePath(t *testing.T) {
	originalPath := logFilePath
	defer func() {
		logFilePath = originalPath
	}()

	testPath := "/tmp/test.log"
	SetLogFilePath(testPath)

	if logFilePath != testPath {
		t.Errorf("SetLogFilePath() = %v, want %v", logFilePath, testPath)
	}
}

func TestInit(t *testing.T) {
	// Clean up any existing state
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	LogChan = nil

	// Set up test directory
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test.log")
	SetLogFilePath(testLogPath)

	// Test Init
	err := Init()
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	// Verify LogChan is created
	if LogChan == nil {
		t.Error("Init() should create LogChan")
	}

	// Verify log file is created
	if logFile == nil {
		t.Error("Init() should open log file")
	}

	// Verify log file exists on disk
	if _, err := os.Stat(testLogPath); os.IsNotExist(err) {
		t.Error("Init() should create log file on disk")
	}

	// Clean up
	Close()
}

func TestInitWithInvalidPath(t *testing.T) {
	// Clean up any existing state
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	LogChan = nil

	// Set an invalid path (directory that can't be created)
	invalidPath := "/root/nonexistent/invalid.log"
	SetLogFilePath(invalidPath)

	// Test Init - should fail
	err := Init()
	if err == nil {
		t.Error("Init() should fail with invalid path")
	}

	// Verify error message
	if !strings.Contains(err.Error(), "failed to create log directory") {
		t.Errorf("Init() error = %v, should mention directory creation failure", err)
	}
}

func TestLog(t *testing.T) {
	// Set up clean test environment
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	LogChan = nil

	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test.log")
	SetLogFilePath(testLogPath)

	// Initialize
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	defer Close()

	testMessage := "Test log message"

	// Test logging
	Log(testMessage)

	// Give a small delay for the log to be written
	time.Sleep(10 * time.Millisecond)

	// Check if message appears in log file
	content, err := os.ReadFile(testLogPath)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), testMessage) {
		t.Errorf("Log file content = %v, should contain %v", string(content), testMessage)
	}

	// Check if message appears in LogChan
	select {
	case msg := <-LogChan:
		if !strings.Contains(msg, testMessage) {
			t.Errorf("LogChan message = %v, should contain %v", msg, testMessage)
		}
		if !strings.Contains(msg, "INFO") {
			t.Errorf("LogChan message = %v, should contain log level INFO", msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Message should appear in LogChan")
	}
}

func TestLogWithoutInit(t *testing.T) {
	// Clean up any existing state
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	LogChan = nil

	// Test logging without initialization - should not panic
	Log("Test message without init")

	// Should not panic, but also shouldn't do anything meaningful
}

func TestLogChannelFull(t *testing.T) {
	// Set up clean test environment
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	LogChan = nil

	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test.log")
	SetLogFilePath(testLogPath)

	// Initialize
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	defer Close()

	// Fill up the channel (it has capacity 100)
	for i := 0; i < 150; i++ {
		Log("Test message")
	}

	// Should not block or panic even when channel is full
	Log("Final message")

	// Verify we can still read from the channel
	messageCount := 0
	timeout := time.After(100 * time.Millisecond)
	
	for {
		select {
		case <-LogChan:
			messageCount++
		case <-timeout:
			// Should have received 100 messages (channel capacity)
			if messageCount != 100 {
				t.Errorf("Received %d messages, expected 100 (channel capacity)", messageCount)
			}
			return
		}
	}
}

func TestClose(t *testing.T) {
	// Set up test environment
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test.log")
	SetLogFilePath(testLogPath)

	// Initialize
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Verify log file is open
	if logFile == nil {
		t.Error("Log file should be open after Init()")
	}

	// Test Close
	err = Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify log file is closed
	if logFile != nil {
		t.Error("Log file should be nil after Close()")
	}

	// Test closing again - should not error
	err = Close()
	if err != nil {
		t.Errorf("Close() called twice should not error, got %v", err)
	}
}

func TestConcurrentLogging(t *testing.T) {
	// Set up clean test environment
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	LogChan = nil

	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test.log")
	SetLogFilePath(testLogPath)

	// Initialize
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	defer Close()

	// Launch multiple goroutines that log concurrently
	done := make(chan bool)
	numGoroutines := 10
	messagesPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < messagesPerGoroutine; j++ {
				Log("Message from goroutine %d, message %d")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Give a small delay for all messages to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify log file contains data (exact count is hard to verify due to timing)
	content, err := os.ReadFile(testLogPath)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file should contain data after concurrent logging")
	}

	// Count lines in the log file
	lines := strings.Split(string(content), "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	// Should have at least some messages (may be less than total due to channel dropping)
	if nonEmptyLines == 0 {
		t.Error("Log file should contain at least some messages")
	}
}

func TestStructuredLogging(t *testing.T) {
	// Set up clean test environment
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	LogChan = nil

	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test.log")
	SetLogFilePath(testLogPath)

	// Initialize
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	defer Close()

	// Test log levels
	tests := []struct {
		level LogLevel
		name  string
	}{
		{DEBUG, "debug"},
		{INFO, "info"},
		{WARN, "warn"},
		{ERROR, "error"},
		{FATAL, "fatal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string representation
			if tt.level.String() != strings.ToUpper(tt.name) {
				t.Errorf("LogLevel.String() = %v, want %v", tt.level.String(), strings.ToUpper(tt.name))
			}
		})
	}

	// Test convenience functions
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	// Test with fields
	Info("message with fields", map[string]interface{}{
		"user_id": 123,
		"action":  "login",
	})

	// Test log level filtering
	SetLogLevel(WARN)
	Debug("this should be filtered")
	Info("this should also be filtered")
	Warn("this should appear")
	Error("this should also appear")

	// Test structured logging toggle
	EnableStructuredLogging(true)
	Info("structured message")
	EnableStructuredLogging(false)
	Info("plain message")

	// Verify log level getters
	if GetLogLevel() != WARN {
		t.Errorf("GetLogLevel() = %v, want %v", GetLogLevel(), WARN)
	}

	EnableStructuredLogging(true)
	if !IsStructuredLoggingEnabled() {
		t.Error("IsStructuredLoggingEnabled() should return true")
	}
	EnableStructuredLogging(false)
	if IsStructuredLoggingEnabled() {
		t.Error("IsStructuredLoggingEnabled() should return false")
	}

	// Reset log level for other tests
	SetLogLevel(INFO)
}