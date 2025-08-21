package detect

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// CommandExecutor is an interface for executing shell commands
type CommandExecutor interface {
	Execute(dir string, command string, args []string) error
}

// DefaultExecutor implements CommandExecutor using os/exec
type DefaultExecutor struct {
	Stdout io.Writer
	Stderr io.Writer
}

// NewDefaultExecutor creates a new default executor with os.Stdout and os.Stderr
func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Execute runs a command in the specified directory
func (e *DefaultExecutor) Execute(dir, command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr

	if err := cmd.Run(); err != nil {
		// Provide more context in the error message
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("command failed with exit code %d: %w", exitErr.ExitCode(), err)
		}
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

// MockExecutor is a test implementation of CommandExecutor
type MockExecutor struct {
	ExecuteCalls []ExecuteCall
	ReturnError  error
}

// ExecuteCall records a call to Execute for verification in tests
type ExecuteCall struct {
	Dir     string
	Command string
	Args    []string
}

// Execute records the call and returns the configured error
func (m *MockExecutor) Execute(dir, command string, args []string) error {
	m.ExecuteCalls = append(m.ExecuteCalls, ExecuteCall{
		Dir:     dir,
		Command: command,
		Args:    args,
	})
	return m.ReturnError
}

// Reset clears the recorded calls
func (m *MockExecutor) Reset() {
	m.ExecuteCalls = nil
}
