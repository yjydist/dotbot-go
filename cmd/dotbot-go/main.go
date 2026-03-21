package main

import (
	"os"

	"github.com/yjydist/dotbot-go/internal/runner"
)

// main 只负责把 CLI 参数和标准流转交给 runner.
func main() {
	os.Exit(runner.Run(os.Args[1:], os.Stdout, os.Stderr))
}
