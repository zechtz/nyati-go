package main

import (
	"flag"
	"log"

	"github.com/zechtz/nyatictl/api"
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
// Flags:
//
//	--web           : Run in web mode, which starts the HTTP server
//	--port          : Port for the web server (used only in web mode, default is 8080)
//	--configs-path  : Path to the configuration JSON file (default is "configs.json")
//	--log-path      : Path to the persistent log output file (default is "nyatictl.log")
//
// Example Usage:
//
//	CLI Mode:
//	  go run main.go
//
//	Web Mode:
//	  go run main.go --web --port 3000 --configs-path ./data/configs.json --log-path ./logs/output.log
func main() {
	// -----------------------------
	// Flag Definitions
	// -----------------------------

	// Indicates whether to run in web UI mode
	webMode := flag.Bool("web", false, "Run in web mode (starts a web server)")

	// Defines the HTTP port the web server should listen on
	port := flag.String("port", "8080", "Port for the web server (used in web mode)")

	// Path to the configuration file that stores available deployment entries
	configsPath := flag.String("configs-path", "configs.json", "Path to the configs.json file")

	// Path where persistent logs will be stored
	logPath := flag.String("log-path", "nyatictl.log", "Path to the persistent log file")

	// Parse all defined flags
	flag.Parse()

	// -----------------------------
	// Logger Setup
	// -----------------------------

	// Set the file path where logs will be persisted BEFORE initializing the logger
	logger.SetLogFilePath(*logPath)

	// Initialize the logging system — this sets up:
	//   1. LogChan for streaming logs to WebSocket clients
	//   2. Persistent file logging to the configured path
	logger.Init()

	// -----------------------------
	// Config File Initialization
	// -----------------------------

	// Set the config path for the web layer (used globally in web package)
	api.ConfigFilePath = *configsPath

	// Ensure that the config file exists at the specified path.
	// If it does not exist, it will be created with an empty JSON array ([]).
	// This prevents "file not found" errors during web UI interactions.
	if err := api.EnsureConfigsFile(); err != nil {
		log.Fatalf("Failed to create config file at '%s': %v", *configsPath, err)
	}

	// -----------------------------
	// Run in Web or CLI Mode
	// -----------------------------

	if *webMode {
		// WEB MODE: Start the backend HTTP server for the web UI
		server, err := api.NewServer()
		if err != nil {
			panic(err) // Startup failed — cannot proceed
		}
		if err := server.Start(*port); err != nil {
			panic(err) // Could not bind or run HTTP server
		}
	} else {
		// CLI MODE: Execute automation tasks via the command line
		if err := cli.Execute(version); err != nil {
			panic(err) // CLI execution failed (bad config, missing tasks, etc.)
		}
	}
}
