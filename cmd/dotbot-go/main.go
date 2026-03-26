package main

import (
	"os"

	"github.com/yjydist/dotbot-go/internal/runner"
)

// main 保持为极薄入口, 避免在 cmd 层混入配置解析或执行细节.
// 这样真正的业务流程都集中在 internal/runner 中, 后续 review 只需要盯住一条主链路.
func main() {
	os.Exit(runner.Run(os.Args[1:], os.Stdout, os.Stderr))
}
