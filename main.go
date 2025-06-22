package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zechtz/nyatictl/api"
	"github.com/zechtz/nyatictl/appconfig"
	"github.com/zechtz/nyatictl/cli"
	"github.com/zechtz/nyatictl/logger"
)

// version represents the current release version of the application.
// This value is passed into CLI and web config validation for compatibility checks.
const version = "0.1.2"

// main is the entry point of the Nyatictl application.
//
// It supports two primary execution modes:
//   - CLI Mode (default): Runs deployment tasks and commands from the terminal
//   - Web Mode (--web): Starts a web server with a UI for managing and executing tasks
//
// Configuration is loaded from environment variables with command-line flag overrides.
// See appconfig package for all available configuration options.
//
// Flags (override environment variables):
//
//	--web           : Run in web mode, which starts the HTTP server
//	--port          : Port for the web server (used only in web mode)
//	--configs-path  : Path to the configuration JSON file
//	--log-path      : Path to the persistent log output file
//
// Example Usage:
//
//	CLI Mode:
//	  go run main.go
//
//	Web Mode with environment:
//	  NYATI_WEB_MODE=true NYATI_PORT=3000 go run main.go
//
//	Web Mode with flags:
//	  go run main.go --web --port 3000 --configs-path ./data/configs.json --log-path ./logs/output.log
func main() {
	// -----------------------------
	// Load Configuration
	// -----------------------------

	// Load configuration from environment variables first
	cfg, err := appconfig.Load()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		return
	}

	// -----------------------------
	// Flag Definitions (override config)
	// -----------------------------

	// Command-line flags can override environment variables
	webMode := flag.Bool("web", cfg.WebMode, "Run in web mode (starts a web server)")
	port := flag.String("port", cfg.Port, "Port for the web server (used in web mode)")
	configsPath := flag.String("configs-path", cfg.ConfigsPath, "Path to the configs.json file")
	logPath := flag.String("log-path", cfg.LogPath, "Path to the persistent log file")

	// Parse all defined flags
	flag.Parse()

	// Override config with command-line flags
	cfg.WebMode = *webMode
	cfg.Port = *port
	cfg.ConfigsPath = *configsPath
	cfg.LogPath = *logPath

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		log.Printf("Configuration validation failed: %v", err)
		return
	}

	// -----------------------------
	// Logger Setup
	// -----------------------------

	// Configure logger based on configuration
	logger.SetLogFilePath(cfg.LogPath)
	logger.SetLogLevel(cfg.GetLogLevel())
	logger.EnableStructuredLogging(cfg.StructuredLogging)

	// Initialize the logging system
	if err := logger.Init(); err != nil {
		log.Printf("Failed to initialize logger: %v", err)
		return
	}

	// Log the loaded configuration
	cfg.LogConfiguration()

	// -----------------------------
	// Config File Initialization
	// -----------------------------

	// Set the config path for the web layer (used globally in web package)
	api.ConfigFilePath = cfg.ConfigsPath

	// Ensure that the config file exists at the specified path.
	// If it does not exist, it will be created with an empty JSON array ([]).
	// This prevents "file not found" errors during web UI interactions.
	if err := api.EnsureConfigsFile(); err != nil {
		logger.Error("Failed to create config file", map[string]interface{}{
			"path": cfg.ConfigsPath,
			"error": err.Error(),
		})
		return
	}

	// -----------------------------
	// Run in Web or CLI Mode
	// -----------------------------

	if cfg.WebMode {
		// WEB MODE: Start the backend HTTP server for the web UI
		server, err := api.NewServerWithConfig(cfg)
		if err != nil {
			logger.Error("Failed to initialize web server", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		// Set up graceful shutdown handling
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

		// Start server in a goroutine
		go func() {
			logger.Info("Starting web server", map[string]interface{}{
				"port": cfg.Port,
				"mode": "web",
			})
			if err := server.Start(cfg.Port); err != nil {
				logger.Error("Web server error", map[string]interface{}{
					"error": err.Error(),
				})
				signalChan <- syscall.SIGTERM
			}
		}()

		// Wait for shutdown signal
		<-signalChan
		logger.Info("Shutdown signal received, cleaning up...")

		// Graceful shutdown with timeout
		shutdownDone := make(chan bool, 1)
		go func() {
			// Close server resources
			if err := server.Close(); err != nil {
				logger.Error("Error closing server", map[string]interface{}{
					"error": err.Error(),
				})
			} else {
				logger.Info("Server closed successfully")
			}
			shutdownDone <- true
		}()

		// Wait for graceful shutdown or timeout
		select {
		case <-shutdownDone:
			logger.Info("Graceful shutdown completed")
		case <-time.After(cfg.ShutdownTimeout):
			logger.Warn("Shutdown timeout reached, forcing exit")
		}

		// Close logger resources
		if err := logger.Close(); err != nil {
			log.Printf("Error closing logger: %v", err)
		}

		logger.Info("Shutdown complete")
	} else {
		// CLI MODE: Execute automation tasks via the command line
		logger.Info("Starting CLI mode", map[string]interface{}{
			"version": version,
		})
		if err := cli.Execute(version); err != nil {
			logger.Error("CLI execution failed", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		// Close logger resources after CLI execution
		if err := logger.Close(); err != nil {
			log.Printf("Error closing logger: %v", err)
		}
	}
}
