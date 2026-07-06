package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// withStdinStdoutStderr temporarily replaces os.Stdin/os.Stdout/os.Stderr for
// the duration of fn, feeding stdinContent as stdin input, and returns
// whatever was written to the substituted stdout and stderr.
//
// TrustPrompt intentionally has no writer/reader parameters (its ui.Interface
// signature matches ConfirmPrompt, which also has none) — it must read/write
// via the real os.Stdin/Stdout/Stderr. Swapping the package-level os.Std*
// variables is the only way to observe that behavior, and is also the
// technique the spec requires for verifying zero stdout contamination
// (checking only an injected io.Writer would miss a real leak).
func withStdinStdoutStderr(t *testing.T, stdinContent string, fn func()) (stdout, stderr string) {
	t.Helper()

	origStdin, origStdout, origStderr := os.Stdin, os.Stdout, os.Stderr
	defer func() {
		os.Stdin, os.Stdout, os.Stderr = origStdin, origStdout, origStderr
	}()

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdin, os.Stdout, os.Stderr = stdinR, stdoutW, stderrW

	go func() {
		_, _ = stdinW.WriteString(stdinContent)
		_ = stdinW.Close()
	}()

	var wg sync.WaitGroup
	var stdoutBuf, stderrBuf bytes.Buffer
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&stdoutBuf, stdoutR) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&stderrBuf, stderrR) }()

	fn()

	_ = stdoutW.Close()
	_ = stderrW.Close()
	wg.Wait()

	return stdoutBuf.String(), stderrBuf.String()
}

func TestDefaultUI_TrustPrompt_DefaultsToNoOnEmptyInput(t *testing.T) {
	u := NewDefaultUI()
	var approved bool
	var err error
	withStdinStdoutStderr(t, "\n", func() {
		approved, err = u.TrustPrompt("/repo/.gwrc", []string{"post_start_hook = pnpm dev"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected default (bare enter) to decline trust")
	}
}

func TestDefaultUI_TrustPrompt_YesApproves(t *testing.T) {
	u := NewDefaultUI()
	var approved bool
	var err error
	withStdinStdoutStderr(t, "y\n", func() {
		approved, err = u.TrustPrompt("/repo/.gwrc", []string{"post_start_hook = pnpm dev"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected 'y' to approve trust")
	}
}

func TestDefaultUI_TrustPrompt_NoDeclines(t *testing.T) {
	u := NewDefaultUI()
	var approved bool
	withStdinStdoutStderr(t, "n\n", func() {
		approved, _ = u.TrustPrompt("/repo/.gwrc", []string{"post_start_hook = pnpm dev"})
	})
	if approved {
		t.Error("expected 'n' to decline trust")
	}
}

func TestDefaultUI_TrustPrompt_GarbageInputDeclines(t *testing.T) {
	u := NewDefaultUI()
	var approved bool
	withStdinStdoutStderr(t, "sure whatever\n", func() {
		approved, _ = u.TrustPrompt("/repo/.gwrc", []string{"post_start_hook = pnpm dev"})
	})
	if approved {
		t.Error("expected non-y/yes input to decline trust (fail closed)")
	}
}

func TestDefaultUI_TrustPrompt_DoesNotWriteToStdout(t *testing.T) {
	u := NewDefaultUI()
	stdout, _ := withStdinStdoutStderr(t, "y\n", func() {
		_, _ = u.TrustPrompt("/repo/.gwrc", []string{"post_start_hook = pnpm dev"})
	})
	if stdout != "" {
		t.Errorf("expected zero bytes written to stdout, got %q", stdout)
	}
}

func TestDefaultUI_TrustPrompt_WritesPromptDetailsToStderr(t *testing.T) {
	u := NewDefaultUI()
	_, stderr := withStdinStdoutStderr(t, "n\n", func() {
		_, _ = u.TrustPrompt("/repo/.gwrc", []string{"post_start_hook = pnpm dev"})
	})
	if !strings.Contains(stderr, "/repo/.gwrc") {
		t.Errorf("expected stderr to mention the project path, got %q", stderr)
	}
	if !strings.Contains(stderr, "post_start_hook = pnpm dev") {
		t.Errorf("expected stderr to show the hook line being trusted, got %q", stderr)
	}
}
