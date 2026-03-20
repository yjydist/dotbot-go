package main

import (
	"os"

	"dotbot-go/internal/runner"
)

func main() {
	os.Exit(runner.Run(os.Args[1:], os.Stdout, os.Stderr))
}
