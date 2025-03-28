package main

import (
	"flag"

	"github.com/zechtz/nyatictl/cli"
	"github.com/zechtz/nyatictl/logger"
	"github.com/zechtz/nyatictl/web"
)

// version defines the current version of the application.
// This is passed into the CLI or web backend to enforce version compatibility.
const version = "0.1.2"

// main is the entry point of the Nyatictl application.
//
// It supports two execution modes:
//  1. CLI mode (default): Executes automation tasks via command-line flags.
//  2. Web mode (--web): Starts a web server that exposes a UI for managing configurations and tasks.
//
// Flags:
//
//	--web   : Enables web UI mode instead of CLI
//	--port  : Specifies the HTTP port for the web server (default: 8080)
//
// The logger is initialized early to capture log messages from both modes.
func main() {
	// Define CLI flags
	webMode := flag.Bool("web", false, "Run in web mode (starts a web server)")
	port := flag.String("port", "8080", "Port for the web server (used in web mode)")
	flag.Parse()

	// Initialize global logging system (used across CLI and Web)
	logger.Init()

	if *webMode {
		// WEB MODE: Start the HTTP server
		server, err := web.NewServer()
		if err != nil {
			// Fatal error: cannot initialize server (likely config issue)
			panic(err)
		}

		// Start the server and bind to the selected port
		if err := server.Start(*port); err != nil {
			panic(err)
		}
	} else {
		// CLI MODE: Run command-line automation flow
		if err := cli.Execute(version); err != nil {
			panic(err)
		}
	}
}
