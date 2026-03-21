package runner

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/yjydist/dotbot-go/internal/output"
)

func TestRunUsesReviewUIForDryRunWhenInteractive(t *testing.T) {
	fixture := newRunnerFixture(t, true)
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	called := false
	withRunnerHooks(t,
		func(io.Reader, io.Writer) bool { return true },
		func(stdin io.Reader, stdout io.Writer, noColor bool, data output.ReviewData) error {
			called = true
			if data.Mode != output.ReviewModeDryRun {
				t.Fatalf("review mode = %q, want dry-run", data.Mode)
			}
			if len(data.Entries) == 0 {
				t.Fatal("review entries empty, want dry-run plan")
			}
			return nil
		},
		nil,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--dry-run"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(dry-run) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !called {
		t.Fatal("review UI not called")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no plain text when review UI handles output", stdout.String())
	}
}

func TestRunUsesReviewUIForCheckWhenInteractive(t *testing.T) {
	fixture := newRunnerFixture(t, true)
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	called := false
	withRunnerHooks(t,
		func(io.Reader, io.Writer) bool { return true },
		func(stdin io.Reader, stdout io.Writer, noColor bool, data output.ReviewData) error {
			called = true
			if data.Mode != output.ReviewModeCheck {
				t.Fatalf("review mode = %q, want check", data.Mode)
			}
			if data.Result != "check ok" {
				t.Fatalf("review result = %q, want check ok", data.Result)
			}
			return nil
		},
		nil,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !called {
		t.Fatal("review UI not called")
	}
}
