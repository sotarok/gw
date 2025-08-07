package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
)

const (
	expectedCwdWorktree = "worktree"
	expectedCwdOriginal = "original"
)

func TestStartCommand_WithConfig(t *testing.T) {
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tests := []struct {
		name        string
		config      *config.Config
		expectedCwd string // Expected working directory after command
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name: "auto-cd enabled",
			config: &config.Config{
				AutoCD: true,
			},
			expectedCwd: expectedCwdWorktree, // Should change to worktree directory
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Shell integration will change to this directory") {
					t.Error("Expected message about shell integration when auto-cd is enabled")
				}
			},
		},
		{
			name: "auto-cd disabled",
			config: &config.Config{
				AutoCD: false,
			},
			expectedCwd: expectedCwdOriginal, // Should stay in original directory
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if contains(stdout, "Shell integration will change to this directory") {
					t.Error("Should not show shell integration message when auto-cd is disabled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for testing
			tempDir, err := os.MkdirTemp("", "gw-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create worktree directory
			worktreeDir := filepath.Join(tempDir, "test-repo-123")
			if err := os.MkdirAll(worktreeDir, 0755); err != nil {
				t.Fatalf("Failed to create worktree dir: %v", err)
			}

			// Change to temp directory
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Setup mocks
			mockGitInstance := &mockGit{
				isGitRepo:    true,
				worktreePath: worktreeDir,
				envFiles:     []git.EnvFile{},
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			deps := &Dependencies{
				Git:    mockGitInstance,
				UI:     &mockUI{},
				Detect: &mockDetect{},
				Stdout: stdout,
				Stderr: stderr,
			}

			// Create command with config
			cmd := NewStartCommandWithConfig(deps, false, tt.config)
			err = cmd.Execute("123", "main")

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check current directory
			currentDir, _ := os.Getwd()
			currentDir, _ = filepath.EvalSymlinks(currentDir)
			worktreeDir, _ = filepath.EvalSymlinks(worktreeDir)
			tempDir, _ = filepath.EvalSymlinks(tempDir)

			if tt.expectedCwd == expectedCwdWorktree {
				if currentDir != worktreeDir {
					t.Errorf("Expected to be in worktree directory %s, but in %s", worktreeDir, currentDir)
				}
			} else {
				if currentDir != tempDir {
					t.Errorf("Expected to be in original directory %s, but in %s", tempDir, currentDir)
				}
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String(), stderr.String())
			}
		})
	}
}

func TestCheckoutCommand_WithConfig(t *testing.T) {
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tests := []struct {
		name        string
		config      *config.Config
		expectedCwd string // Expected working directory after command
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name: "auto-cd enabled",
			config: &config.Config{
				AutoCD: true,
			},
			expectedCwd: expectedCwdWorktree,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Shell integration will change to this directory") {
					t.Error("Expected message about shell integration when auto-cd is enabled")
				}
			},
		},
		{
			name: "auto-cd disabled",
			config: &config.Config{
				AutoCD: false,
			},
			expectedCwd: expectedCwdOriginal,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if contains(stdout, "Shell integration will change to this directory") {
					t.Error("Should not show shell integration message when auto-cd is disabled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for testing
			tempDir, err := os.MkdirTemp("", "gw-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create worktree directory
			// Checkout command creates worktree with sanitized branch name
			worktreeDir := filepath.Join(tempDir, "..", "test-repo-feature_test")
			if err := os.MkdirAll(worktreeDir, 0755); err != nil {
				t.Fatalf("Failed to create worktree dir: %v", err)
			}
			defer os.RemoveAll(worktreeDir)

			// Change to temp directory
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Setup mocks
			mockGitInstance := &mockGit{
				isGitRepo: true,
				envFiles:  []git.EnvFile{},
			}
			mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
				return true, nil
			}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			deps := &Dependencies{
				Git:    mockGitInstance,
				UI:     &mockUI{},
				Detect: &mockDetect{},
				Stdout: stdout,
				Stderr: stderr,
			}

			// Create command with config
			cmd := NewCheckoutCommandWithConfig(deps, false, tt.config)
			err = cmd.Execute("feature/test")

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check current directory
			currentDir, _ := os.Getwd()
			currentDir, _ = filepath.EvalSymlinks(currentDir)
			worktreeDir, _ = filepath.EvalSymlinks(worktreeDir)
			tempDir, _ = filepath.EvalSymlinks(tempDir)

			if tt.expectedCwd == expectedCwdWorktree {
				// Should be in the absolute path of worktree
				if currentDir != worktreeDir {
					t.Errorf("Expected to be in worktree directory %s, but in %s", worktreeDir, currentDir)
				}
			} else {
				if currentDir != tempDir {
					t.Errorf("Expected to be in original directory %s, but in %s", tempDir, currentDir)
				}
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String(), stderr.String())
			}
		})
	}
}
