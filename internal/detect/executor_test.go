package detect

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

const windowsOS = "windows"

func TestDefaultExecutor_Execute(t *testing.T) {
	t.Run("executes command successfully", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		executor := &DefaultExecutor{
			Stdout: &stdout,
			Stderr: &stderr,
		}

		// Use 'echo' command which should be available on all platforms
		err := executor.Execute(".", "echo", []string{"test"})
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		output := strings.TrimSpace(stdout.String())
		if output != "test" {
			t.Errorf("unexpected output: got %q, want %q", output, "test")
		}
	})

	t.Run("captures stderr output", func(t *testing.T) {
		if runtime.GOOS == windowsOS {
			t.Skip("sh not available on Windows")
		}

		var stdout, stderr bytes.Buffer
		executor := &DefaultExecutor{
			Stdout: &stdout,
			Stderr: &stderr,
		}

		// Use sh to write to stderr
		err := executor.Execute(".", "sh", []string{"-c", "echo error >&2"})
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		stderrOutput := strings.TrimSpace(stderr.String())
		if stderrOutput != "error" {
			t.Errorf("unexpected stderr: got %q, want %q", stderrOutput, "error")
		}
	})

	t.Run("returns error for non-existent command", func(t *testing.T) {
		executor := NewDefaultExecutor()

		err := executor.Execute(".", "this-command-does-not-exist-12345", []string{})
		if err == nil {
			t.Error("Execute() should have returned error for non-existent command")
		}

		// Check if it's wrapped with our custom message
		if !strings.Contains(err.Error(), "failed to execute command") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("returns error with exit code for failing command", func(t *testing.T) {
		executor := NewDefaultExecutor()

		// 'false' command always exits with error
		err := executor.Execute(".", "false", []string{})
		if err == nil {
			t.Error("Execute() should have returned error for failing command")
		}

		// Check if error message contains exit code information
		if !strings.Contains(err.Error(), "command failed with exit code") {
			t.Errorf("error should contain exit code information: %v", err)
		}
	})

	t.Run("executes in specified directory", func(t *testing.T) {
		if runtime.GOOS == windowsOS {
			t.Skip("pwd not available on Windows")
		}

		var stdout bytes.Buffer
		executor := &DefaultExecutor{
			Stdout: &stdout,
			Stderr: &bytes.Buffer{},
		}

		// Get current directory with pwd
		err := executor.Execute(".", "pwd", []string{})
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		// Output should contain current directory path
		output := strings.TrimSpace(stdout.String())
		if output == "" {
			t.Error("pwd should have returned current directory")
		}
	})
}

func TestNewDefaultExecutor(t *testing.T) {
	executor := NewDefaultExecutor()

	if executor == nil {
		t.Fatal("NewDefaultExecutor() returned nil")
	}

	if executor.Stdout == nil {
		t.Error("Stdout should be initialized")
	}

	if executor.Stderr == nil {
		t.Error("Stderr should be initialized")
	}
}

func TestMockExecutor(t *testing.T) {
	t.Run("records execute calls", func(t *testing.T) {
		mock := &MockExecutor{}

		// Make several calls
		_ = mock.Execute("/path1", "cmd1", []string{"arg1", "arg2"})
		_ = mock.Execute("/path2", "cmd2", []string{"arg3"})
		_ = mock.Execute("/path3", "cmd3", nil)

		// Verify calls were recorded
		if len(mock.ExecuteCalls) != 3 {
			t.Errorf("expected 3 calls, got %d", len(mock.ExecuteCalls))
		}

		// Verify first call
		call := mock.ExecuteCalls[0]
		if call.Dir != "/path1" || call.Command != "cmd1" || len(call.Args) != 2 {
			t.Errorf("first call not recorded correctly: %+v", call)
		}

		// Verify second call
		call = mock.ExecuteCalls[1]
		if call.Dir != "/path2" || call.Command != "cmd2" || len(call.Args) != 1 {
			t.Errorf("second call not recorded correctly: %+v", call)
		}
	})

	t.Run("returns configured error", func(t *testing.T) {
		expectedErr := exec.ErrNotFound
		mock := &MockExecutor{
			ReturnError: expectedErr,
		}

		err := mock.Execute(".", "test", nil)
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("reset clears calls", func(t *testing.T) {
		mock := &MockExecutor{}

		// Make some calls
		_ = mock.Execute(".", "test", nil)
		_ = mock.Execute(".", "test2", nil)

		if len(mock.ExecuteCalls) != 2 {
			t.Fatalf("expected 2 calls before reset")
		}

		// Reset
		mock.Reset()

		if len(mock.ExecuteCalls) != 0 {
			t.Errorf("expected 0 calls after reset, got %d", len(mock.ExecuteCalls))
		}
	})
}
