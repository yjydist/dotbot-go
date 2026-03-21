package runner

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yjydist/dotbot-go/internal/output"
)

func withRunnerHooks(
	t *testing.T,
	interactive func(io.Reader, io.Writer) bool,
	review func(io.Reader, io.Writer, bool, output.ReviewData) error,
	confirm func(io.Reader, io.Writer, bool, []output.RiskItem) error,
) {
	t.Helper()
	oldInteractive := interactiveTerminal
	oldReview := runReviewUI
	oldConfirm := runConfirmUI
	if interactive != nil {
		interactiveTerminal = interactive
	}
	if review != nil {
		runReviewUI = review
	}
	if confirm != nil {
		runConfirmUI = confirm
	}
	t.Cleanup(func() {
		interactiveTerminal = oldInteractive
		runReviewUI = oldReview
		runConfirmUI = oldConfirm
	})
}

type runnerFixture struct {
	baseDir    string
	configPath string
	homeDir    string
}

// newRunnerFixture 负责准备最常见的测试环境:
// 独立工作目录, fake HOME, 配置文件路径, 以及可选的 git/gitconfig 源文件.
func newRunnerFixture(t *testing.T, withGitSource bool) runnerFixture {
	t.Helper()

	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if withGitSource {
		if err := os.MkdirAll(filepath.Join(baseDir, "git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(baseDir, "git", "gitconfig"), []byte("[user]"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("HOME", homeDir)
	t.Chdir(baseDir)

	return runnerFixture{
		baseDir:    baseDir,
		configPath: filepath.Join(baseDir, "dotbot-go.toml"),
		homeDir:    homeDir,
	}
}

func (f runnerFixture) writeConfig(t *testing.T, lines ...string) {
	t.Helper()
	if err := os.WriteFile(f.configPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}
