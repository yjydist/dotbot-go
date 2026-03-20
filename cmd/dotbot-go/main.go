package main

import (
	"os"

	"github.com/yjydist/dotbot-go/internal/runner"
)

func main() {
	os.Exit(runner.Run(os.Args[1:], os.Stdout, os.Stderr))
}
