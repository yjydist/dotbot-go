package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLoadsDefaultConfig(t *testing.T) {
	// 不显式传 -c 时, runner 应自动加载当前目录默认配置文件.
	fixture := newRunnerFixture(t, true)
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run(nil, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[ok]") {
		t.Fatalf("stdout = %q, want operation output", stdout.String())
	}
	if !strings.Contains(stdout.String(), "summary:") {
		t.Fatalf("stdout = %q, want summary output", stdout.String())
	}
}

func TestRunQuietSuppressesSuccessOutput(t *testing.T) {
	// quiet 在真实执行模式下也应该压掉成功条目和 summary.
	fixture := newRunnerFixture(t, true)
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--quiet"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(quiet) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunQuietStillPrintsFailure(t *testing.T) {
	// quiet 不能吞掉失败, 否则脚本场景很难定位执行错误.
	fixture := newRunnerFixture(t, false)
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"missing/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--quiet"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(quiet) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[fail]") {
		t.Fatalf("stdout = %q, want failure line", stdout.String())
	}
}

func TestRunVerboseShowsConfigDetails(t *testing.T) {
	// verbose 要补充配置路径, base dir, 生效配置和阶段统计.
	fixture := newRunnerFixture(t, true)
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--verbose"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(verbose) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "config:") {
		t.Fatalf("stdout = %q, want config details", stdout.String())
	}
	if !strings.Contains(stdout.String(), "link:") {
		t.Fatalf("stdout = %q, want effective config summary", stdout.String())
	}
	if !strings.Contains(stdout.String(), "stages:") {
		t.Fatalf("stdout = %q, want stage summary", stdout.String())
	}
}

func TestRunVerboseShowsPerLinkDetailsWhenValuesMixed(t *testing.T) {
	fixture := newRunnerFixture(t, true)
	if err := os.WriteFile(filepath.Join(fixture.baseDir, "git", "zshrc"), []byte("export ZDOTDIR"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
		"create = true",
		"[[link]]",
		"target = \"~/.zshrc\"",
		"source = \"git/zshrc\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--verbose"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(verbose) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "link[1]: target=") || !strings.Contains(stdout.String(), "link[2]: target=") {
		t.Fatalf("stdout = %q, want per-link verbose details", stdout.String())
	}
}

func TestRunAllowsProtectedTargetWithOverride(t *testing.T) {
	// 显式 override 时, 受保护目标应该允许继续执行, 不再弹确认 UI.
	baseDir := t.TempDir()
	parentDir := t.TempDir()
	configPath := filepath.Join(parentDir, "dotbot-go.toml")
	source := filepath.Join(parentDir, "source.txt")
	protectedTarget := filepath.Join(parentDir, "protected")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(protectedTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(strings.Join([]string{
		"[[link]]",
		"target = " + quote(protectedTarget),
		"source = " + quote(source),
		"force = true",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", configPath, "--allow-protected-target"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if _, err := os.Readlink(protectedTarget); err != nil {
		t.Fatalf("protected target is not symlink: %v", err)
	}
}

// quote 只在测试里生成 TOML 字符串字面量, 避免每个用例重复写转义.
func quote(v string) string {
	return "\"" + v + "\""
}
