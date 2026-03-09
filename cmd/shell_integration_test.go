package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sotarok/gw/internal/git"
)

func TestRunShellIntegration(t *testing.T) {
	// Save original values
	originalShowScript := shellIntegrationShowScript
	originalShell := shellIntegrationShell
	originalPrintPath := shellIntegrationPrintPath
	defer func() {
		shellIntegrationShowScript = originalShowScript
		shellIntegrationShell = originalShell
		shellIntegrationPrintPath = originalPrintPath
	}()

	tests := []struct {
		name       string
		showScript bool
		shell      string
		printPath  string
		wantError  bool
	}{
		{
			name:       "run with show-script flag",
			showScript: true,
			shell:      "bash",
			wantError:  false,
		},
		{
			name:      "run with print-path flag",
			printPath: "test-123",
			wantError: true, // Will error as we're not in a git repo
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global flags
			shellIntegrationShowScript = tt.showScript
			shellIntegrationShell = tt.shell
			shellIntegrationPrintPath = tt.printPath

			// Run the command
			err := runShellIntegration(nil, []string{})

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestShellIntegrationCommand_DetectShell(t *testing.T) {
	tests := []struct {
		name     string
		shellEnv string
		expected string
	}{
		{
			name:     "detects zsh",
			shellEnv: "/usr/local/bin/zsh",
			expected: "zsh",
		},
		{
			name:     "detects bash",
			shellEnv: "/bin/bash",
			expected: "bash",
		},
		{
			name:     "detects fish",
			shellEnv: "/opt/local/bin/fish",
			expected: "fish",
		},
		{
			name:     "defaults to bash for unknown",
			shellEnv: "/bin/sh",
			expected: "bash",
		},
		{
			name:     "handles empty SHELL env",
			shellEnv: "",
			expected: "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore SHELL env
			oldShell := os.Getenv("SHELL")
			os.Setenv("SHELL", tt.shellEnv)
			defer os.Setenv("SHELL", oldShell)

			cmd := &ShellIntegrationCommand{}
			result := cmd.detectShell()

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestShellIntegrationCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		showScript  bool
		shell       string
		printPath   string
		wantError   bool
		errorMsg    string
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:       "show bash script",
			showScript: true,
			shell:      "bash",
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "gw()") {
					t.Error("Expected shell function definition")
				}
				if !strings.Contains(output, "#!/bin/bash") {
					t.Error("Expected bash shebang")
				}
				if !strings.Contains(output, "auto_cd = true") {
					t.Error("Expected auto_cd check")
				}
			},
		},
		{
			name:       "show zsh script",
			showScript: true,
			shell:      "zsh",
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "gw()") {
					t.Error("Expected shell function definition")
				}
				if !strings.Contains(output, "#!/bin/zsh") {
					t.Error("Expected zsh shebang")
				}
			},
		},
		{
			name:       "show fish script",
			showScript: true,
			shell:      "fish",
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "function gw") {
					t.Error("Expected fish function definition")
				}
				if !strings.Contains(output, "#!/usr/bin/env fish") {
					t.Error("Expected fish shebang")
				}
			},
		},
		{
			name:       "auto-detect shell from environment",
			showScript: true,
			shell:      "", // Should auto-detect
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "gw") {
					t.Error("Expected shell function")
				}
			},
		},
		{
			name:       "unsupported shell",
			showScript: true,
			shell:      "tcsh",
			wantError:  true,
			errorMsg:   "unsupported shell: tcsh",
		},
		{
			name:      "no flags specified",
			wantError: true,
			errorMsg:  "either --show-script or --print-path must be specified",
		},
		{
			name:       "both flags specified",
			showScript: true,
			printPath:  "123",
			wantError:  true,
			errorMsg:   "cannot use both --show-script and --print-path",
		},
		{
			name:      "print-path with non-existent worktree",
			printPath: "99999", // Using a high number that's unlikely to exist
			wantError: true,
			// Don't check specific error message as it depends on whether we're in a git repo or not
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			cmd := NewShellIntegrationCommand(stdout, stderr)
			cmd.showScript = tt.showScript
			cmd.shell = tt.shell
			cmd.printPath = tt.printPath

			err := cmd.Execute()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String())
			}
		})
	}
}

func TestFindWorktreePath_DirectoryExists(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	worktreeDir := filepath.Join(tempDir, "test-repo-123")
	os.MkdirAll(repoDir, 0755)
	os.MkdirAll(worktreeDir, 0755)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	mock := &mockGit{
		isGitRepo: true,
	}

	path, err := findWorktreePath(mock, "123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Resolve symlinks for macOS /var -> /private/var
	resolvedPath, _ := filepath.EvalSymlinks(path)
	resolvedExpected, _ := filepath.EvalSymlinks(worktreeDir)
	if resolvedPath != resolvedExpected {
		t.Errorf("expected %q, got %q", resolvedExpected, resolvedPath)
	}
}

func TestFindWorktreePath_MatchesByBranch(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	mock := &mockGit{
		isGitRepo: true,
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/some/path/main", Branch: "main"},
				{Path: "/some/path/feature", Branch: "feature/login"},
			}, nil
		},
	}

	path, err := findWorktreePath(mock, "feature/login")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if path != "/some/path/feature" {
		t.Errorf("expected /some/path/feature, got %q", path)
	}
}

func TestFindWorktreePath_MatchesByIssuePrefix(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	mock := &mockGit{
		isGitRepo: true,
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/some/path/main", Branch: "main"},
				{Path: "/some/path/issue-456", Branch: "456/impl"},
			}, nil
		},
	}

	path, err := findWorktreePath(mock, "456")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if path != "/some/path/issue-456" {
		t.Errorf("expected /some/path/issue-456, got %q", path)
	}
}

func TestFindWorktreePath_SanitizedBranchDir(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	worktreeDir := filepath.Join(tempDir, "test-repo-feature-login")
	os.MkdirAll(repoDir, 0755)
	os.MkdirAll(worktreeDir, 0755)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	mock := &mockGit{
		isGitRepo: true,
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{}, nil
		},
		SanitizeBranchNameForDirFn: func(branch string) string {
			return strings.ReplaceAll(branch, "/", "-")
		},
	}

	path, err := findWorktreePath(mock, "feature/login")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Resolve symlinks for macOS /var -> /private/var
	resolvedPath, _ := filepath.EvalSymlinks(path)
	resolvedExpected, _ := filepath.EvalSymlinks(worktreeDir)
	if resolvedPath != resolvedExpected {
		t.Errorf("expected %q, got %q", resolvedExpected, resolvedPath)
	}
}

func TestFindWorktreePath_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	mock := &mockGit{
		isGitRepo: true,
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{}, nil
		},
	}

	_, err = findWorktreePath(mock, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent worktree")
	}
	if !strings.Contains(err.Error(), "worktree not found for") {
		t.Errorf("expected 'worktree not found for' error, got: %v", err)
	}
}

func TestFindWorktreePath_GetRepoNameError(t *testing.T) {
	customMock := &mockGitWithRepoError{}

	_, err := findWorktreePath(customMock, "123")
	if err == nil {
		t.Error("expected error when GetRepositoryName fails")
	}
}

// mockGitWithRepoError is a mock that returns an error from GetRepositoryName
type mockGitWithRepoError struct {
	mockGit
}

func (m *mockGitWithRepoError) GetRepositoryName() (string, error) {
	return "", fmt.Errorf("mock repo name error")
}

func TestFindWorktreePath_ListWorktreesError(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	mock := &mockGit{
		isGitRepo: true,
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return nil, fmt.Errorf("list worktrees error")
		},
	}

	_, err = findWorktreePath(mock, "999")
	if err == nil {
		t.Error("expected error when worktree not found")
	}
	if !strings.Contains(err.Error(), "worktree not found for") {
		t.Errorf("expected 'worktree not found for' error, got: %v", err)
	}
}

func TestShellIntegrationCommand_GetBashZshScript(t *testing.T) {
	cmd := &ShellIntegrationCommand{}

	t.Run("bash script has bash shebang", func(t *testing.T) {
		script := cmd.getBashZshScript("bash")
		if !strings.HasPrefix(script, "#!/bin/bash") {
			t.Error("expected bash shebang at start")
		}
		if !strings.Contains(script, "--shell=bash") {
			t.Error("expected --shell=bash in script")
		}
	})

	t.Run("zsh script has zsh shebang", func(t *testing.T) {
		script := cmd.getBashZshScript("zsh")
		if !strings.HasPrefix(script, "#!/bin/zsh") {
			t.Error("expected zsh shebang at start")
		}
		if !strings.Contains(script, "--shell=zsh") {
			t.Error("expected --shell=zsh in script")
		}
	})
}

func TestShellIntegrationCommand_GetFishScript(t *testing.T) {
	cmd := &ShellIntegrationCommand{}

	script := cmd.getFishScript()
	if !strings.Contains(script, "#!/usr/bin/env fish") {
		t.Error("expected fish shebang")
	}
	if !strings.Contains(script, "function gw") {
		t.Error("expected fish function definition")
	}
	if !strings.Contains(script, "command gw $argv") {
		t.Error("expected 'command gw $argv' in fish script")
	}
}

func TestNewShellIntegrationCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := NewShellIntegrationCommand(stdout, stderr)
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.stdout != stdout {
		t.Error("stdout not set correctly")
	}
	if cmd.stderr != stderr {
		t.Error("stderr not set correctly")
	}
	if cmd.showScript {
		t.Error("showScript should default to false")
	}
	if cmd.shell != "" {
		t.Error("shell should default to empty string")
	}
	if cmd.printPath != "" {
		t.Error("printPath should default to empty string")
	}
}
