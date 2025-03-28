package main

import (
	"flag"

	"github.com/zechtz/nyatictl/cli"
	"github.com/zechtz/nyatictl/logger"
	"github.com/zechtz/nyatictl/web"
)

const version = "0.1.2"

func main() {
	webMode := flag.Bool("web", false, "Run in web mode (starts a web server)")
	port := flag.String("port", "8080", "Port for the web server (used in web mode)")
	flag.Parse()

	// Initialize the logger
	logger.Init()

	if *webMode {
		// Start the web server
		server, err := web.NewServer()
		if err != nil {
			panic(err)
		}
		if err := server.Start(*port); err != nil {
			panic(err)
		}
	} else {
		// Run the CLI
		if err := cli.Execute(version); err != nil {
			panic(err)
		}
	}
}
