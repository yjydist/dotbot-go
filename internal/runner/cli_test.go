package runner

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--help"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(help) = %d, want %d", got, want)
	}
	if !strings.Contains(stdout.String(), "dotbot-go") {
		t.Fatalf("stdout = %q, want help output", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRejectsVerboseQuietTogether(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--verbose", "--quiet"}, strings.NewReader(""), &stdout, &stderr), 2; got != want {
		t.Fatalf("Run(verbose+quiet) = %d, want %d", got, want)
	}
	if !strings.Contains(stderr.String(), "cannot be used together") {
		t.Fatalf("stderr = %q, want mutual exclusion error", stderr.String())
	}
}

func TestRunMissingConfigReturnsConfigError(t *testing.T) {
	fixture := newRunnerFixture(t, false)
	_ = fixture

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run(nil, strings.NewReader(""), &stdout, &stderr), 2; got != want {
		t.Fatalf("Run() = %d, want %d", got, want)
	}
	if !strings.Contains(stderr.String(), "decode config") {
		t.Fatalf("stderr = %q, want decode config error", stderr.String())
	}
}
