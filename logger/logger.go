package logger

import "sync"

// LogChan is a globally accessible buffered channel for log messages.
//
// It is initialized by calling Init() and is used to send log strings
// from different parts of the application in a thread-safe manner.
var (
	LogChan chan string // Channel used for asynchronous logging
	logLock sync.Mutex  // Mutex used to protect concurrent access to LogChan
)

// Init initializes the global LogChan channel if it hasn't been already.
//
// This function is thread-safe and uses a mutex to prevent race conditions
// when initializing the shared LogChan resource. It creates a buffered
// channel with capacity 100 to allow some non-blocking message queuing.
//
// Best practice: Call this once at the start of your application.
func Init() {
	logLock.Lock()
	defer logLock.Unlock()

	// Prevent re-initialization if already set
	if LogChan == nil {
		LogChan = make(chan string, 100)
	}
}

// Log safely sends a message string to the global LogChan.
//
// If the channel has been initialized, the message is pushed into it.
// This function is thread-safe and guarded by a mutex to avoid panics
// from writing to a nil or closed channel.
//
// Parameters:
//   - msg: The string message to be logged
func Log(msg string) {
	logLock.Lock()
	defer logLock.Unlock()

	if LogChan != nil {
		LogChan <- msg
	}
}
