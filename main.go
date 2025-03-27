package main

import (
	"fmt"
	"os"

	"github.com/zechtz/nyatictl/cli"
)

const appVersion = "0.1.2"

func main() {
	if err := cli.Execute(appVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
