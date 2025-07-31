package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/ui"
)

// Mock implementations

type mockGit struct {
	isGitRepo           bool
	worktreeExists      bool
	createWorktreeError error
	worktreePath        string
	envFiles            []git.EnvFile
	findEnvError        error
	copyEnvError        error
}

func (m *mockGit) IsGitRepository() bool {
	return m.isGitRepo
}

func (m *mockGit) GetRepositoryName() (string, error) {
	return "test-repo", nil
}

func (m *mockGit) GetCurrentBranch() (string, error) {
	return "main", nil
}

func (m *mockGit) CreateWorktree(issueNumber, baseBranch string) (string, error) {
	if m.createWorktreeError != nil {
		return "", m.createWorktreeError
	}
	return m.worktreePath, nil
}

func (m *mockGit) CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error {
	return nil
}

func (m *mockGit) RemoveWorktree(issueNumber string) error {
	return nil
}

func (m *mockGit) RemoveWorktreeByPath(worktreePath string) error {
	return nil
}

func (m *mockGit) ListWorktrees() ([]git.WorktreeInfo, error) {
	return nil, nil
}

func (m *mockGit) GetWorktreeForIssue(issueNumber string) (*git.WorktreeInfo, error) {
	if m.worktreeExists {
		return &git.WorktreeInfo{Path: "/existing/path"}, nil
	}
	return nil, nil
}

func (m *mockGit) BranchExists(branch string) (bool, error) {
	return true, nil
}

func (m *mockGit) ListAllBranches() ([]string, error) {
	return []string{"main", "feature"}, nil
}

func (m *mockGit) HasUncommittedChanges() (bool, error) {
	return false, nil
}

func (m *mockGit) HasUnpushedCommits() (bool, error) {
	return false, nil
}

func (m *mockGit) FindUntrackedEnvFiles(repoPath string) ([]git.EnvFile, error) {
	if m.findEnvError != nil {
		return nil, m.findEnvError
	}
	return m.envFiles, nil
}

func (m *mockGit) CopyEnvFiles(envFiles []git.EnvFile, sourceRoot, destRoot string) error {
	return m.copyEnvError
}

func (m *mockGit) RunCommand(command string) error {
	return nil
}

func (m *mockGit) SanitizeBranchNameForDirectory(branch string) string {
	return branch
}

type mockUI struct {
	confirmResult bool
	confirmError  error
}

func (m *mockUI) SelectWorktree() (*git.WorktreeInfo, error) {
	return nil, nil
}

func (m *mockUI) ShowSelector(title string, items []ui.SelectorItem) (*ui.SelectorItem, error) {
	return nil, nil
}

func (m *mockUI) ConfirmPrompt(message string) (bool, error) {
	return m.confirmResult, m.confirmError
}

func (m *mockUI) ShowEnvFilesList(files []string) {
	// Mock implementation - do nothing
}

type mockDetect struct {
	setupError error
}

func (m *mockDetect) DetectPackageManager(path string) (*detect.PackageManager, error) {
	return nil, nil
}

func (m *mockDetect) RunSetup(path string) error {
	return m.setupError
}

// Tests

func TestStartCommand_Execute(t *testing.T) {
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tests := []struct {
		name          string
		issueNumber   string
		baseBranch    string
		copyEnvs      bool
		mockSetup     func() (*mockGit, *mockUI, *mockDetect, func())
		expectedError string
		checkOutput   func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "not in git repository",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				return &mockGit{isGitRepo: false}, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "not in a git repository",
		},
		{
			name:        "worktree already exists",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				return &mockGit{
					isGitRepo:      true,
					worktreeExists: true,
				}, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "worktree for issue 123 already exists at /existing/path",
		},
		{
			name:        "successful creation without env files",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				// Create a real temporary directory for worktree path
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
					envFiles:     []git.EnvFile{},
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Creating worktree for issue #123") {
					t.Error("Expected creation message in stdout")
				}
				if !contains(stdout, "✓ Created worktree at") {
					t.Error("Expected success message in stdout")
				}
			},
		},
		{
			name:        "successful creation with env files - user confirms",
			issueNumber: "123",
			baseBranch:  "main",
			copyEnvs:    false,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
					envFiles: []git.EnvFile{
						{Path: ".env", AbsolutePath: "/repo/.env"},
					},
				}, &mockUI{confirmResult: true}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Found 1 untracked environment file(s):") {
					t.Error("Expected env files message in stdout")
				}
				if !contains(stdout, "✓ Environment files copied successfully") {
					t.Error("Expected copy success message in stdout")
				}
			},
		},
		{
			name:        "successful creation with env files - copyEnvs flag set",
			issueNumber: "123",
			baseBranch:  "main",
			copyEnvs:    true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
					envFiles: []git.EnvFile{
						{Path: ".env", AbsolutePath: "/repo/.env"},
					},
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Copying environment files:") {
					t.Error("Expected copying message in stdout")
				}
				if !contains(stdout, "✓ Environment files copied successfully") {
					t.Error("Expected copy success message in stdout")
				}
			},
		},
		{
			name:        "setup failure warns but doesn't fail",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
				}, &mockUI{}, &mockDetect{setupError: fmt.Errorf("npm install failed")}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stderr, "⚠ Setup failed: npm install failed") {
					t.Error("Expected setup warning in stderr")
				}
				if !contains(stdout, "✨ Worktree ready!") {
					t.Error("Expected success message despite setup failure")
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

			// Change to temp directory
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Setup mocks
			mockGit, mockUI, mockDetect, cleanup := tt.mockSetup()
			defer cleanup()
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			deps := &Dependencies{
				Git:    mockGit,
				UI:     mockUI,
				Detect: mockDetect,
				Stdout: stdout,
				Stderr: stderr,
			}

			cmd := NewStartCommand(deps, tt.copyEnvs)
			err = cmd.Execute(tt.issueNumber, tt.baseBranch)

			// Check error
			if tt.expectedError != "" {
				if err == nil || err.Error() != tt.expectedError {
					t.Errorf("Expected error %q, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String(), stderr.String())
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
