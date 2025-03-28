package logger

import "sync"

// LogChan is a global channel for log messages.
var (
	LogChan chan string
	logLock sync.Mutex
)

// Init initializes the global log channel.
func Init() {
	logLock.Lock()
	defer logLock.Unlock()
	if LogChan == nil {
		LogChan = make(chan string, 100)
	}
}

// Log sends a message to the global log channel.
func Log(msg string) {
	logLock.Lock()
	defer logLock.Unlock()
	if LogChan != nil {
		LogChan <- msg
	}
}
