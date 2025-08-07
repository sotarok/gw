package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Override functions for custom behavior
	BranchExistsFn          func(string) (bool, error)
	ListAllBranchesFn       func() ([]string, error)
	GetCurrentBranchFn      func() (string, error)
	GetWorktreeForIssueFn   func(string) (*git.WorktreeInfo, error)
	HasUncommittedChangesFn func() (bool, error)
	HasUnpushedCommitsFn    func() (bool, error)
	IsMergedToOriginFn      func(string) (bool, error)
}

func (m *mockGit) IsGitRepository() bool {
	return m.isGitRepo
}

func (m *mockGit) GetRepositoryName() (string, error) {
	return "test-repo", nil
}

func (m *mockGit) GetCurrentBranch() (string, error) {
	if m.GetCurrentBranchFn != nil {
		return m.GetCurrentBranchFn()
	}
	return defaultBaseBranch, nil
}

func (m *mockGit) CreateWorktree(issueNumber, baseBranch string) (string, error) {
	if m.createWorktreeError != nil {
		return "", m.createWorktreeError
	}
	return m.worktreePath, nil
}

func (m *mockGit) CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error {
	// Create the worktree directory for the test
	absolutePath, _ := filepath.Abs(worktreePath)
	os.MkdirAll(absolutePath, 0755)
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
	if m.GetWorktreeForIssueFn != nil {
		return m.GetWorktreeForIssueFn(issueNumber)
	}
	if m.worktreeExists {
		return &git.WorktreeInfo{Path: "/existing/path"}, nil
	}
	return nil, nil
}

func (m *mockGit) BranchExists(branch string) (bool, error) {
	if m.BranchExistsFn != nil {
		return m.BranchExistsFn(branch)
	}
	return false, nil
}

func (m *mockGit) ListAllBranches() ([]string, error) {
	if m.ListAllBranchesFn != nil {
		return m.ListAllBranchesFn()
	}
	return []string{defaultBaseBranch, "feature"}, nil
}

func (m *mockGit) HasUncommittedChanges() (bool, error) {
	if m.HasUncommittedChangesFn != nil {
		return m.HasUncommittedChangesFn()
	}
	return false, nil
}

func (m *mockGit) HasUnpushedCommits() (bool, error) {
	if m.HasUnpushedCommitsFn != nil {
		return m.HasUnpushedCommitsFn()
	}
	return false, nil
}

func (m *mockGit) IsMergedToOrigin(targetBranch string) (bool, error) {
	if m.IsMergedToOriginFn != nil {
		return m.IsMergedToOriginFn(targetBranch)
	}
	// Default to merged unless overridden
	return true, nil
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
	// Simple sanitization for testing
	return strings.ReplaceAll(branch, "/", "_")
}

type mockUI struct {
	confirmResult bool
	confirmError  error

	// Override functions for custom behavior
	ShowSelectorFn   func(string, []ui.SelectorItem) (*ui.SelectorItem, error)
	SelectWorktreeFn func() (*git.WorktreeInfo, error)
}

func (m *mockUI) SelectWorktree() (*git.WorktreeInfo, error) {
	if m.SelectWorktreeFn != nil {
		return m.SelectWorktreeFn()
	}
	return nil, nil
}

func (m *mockUI) ShowSelector(title string, items []ui.SelectorItem) (*ui.SelectorItem, error) {
	if m.ShowSelectorFn != nil {
		return m.ShowSelectorFn(title, items)
	}
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

func TestEndCommand_PerformSafetyChecks(t *testing.T) {
	tests := []struct {
		name             string
		mockSetup        func() *mockGit
		expectedWarnings []string
		checkStderr      bool
	}{
		{
			name: "no warnings when everything is clean",
			mockSetup: func() *mockGit {
				return &mockGit{
					HasUncommittedChangesFn: func() (bool, error) { return false, nil },
					HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
				}
			},
			expectedWarnings: []string{},
		},
		{
			name: "warns about uncommitted changes",
			mockSetup: func() *mockGit {
				return &mockGit{
					HasUncommittedChangesFn: func() (bool, error) { return true, nil },
					HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
				}
			},
			expectedWarnings: []string{"You have uncommitted changes"},
		},
		{
			name: "warns about unpushed commits",
			mockSetup: func() *mockGit {
				return &mockGit{
					HasUncommittedChangesFn: func() (bool, error) { return false, nil },
					HasUnpushedCommitsFn:    func() (bool, error) { return true, nil },
				}
			},
			expectedWarnings: []string{"You have unpushed commits"},
		},
		{
			name: "multiple warnings",
			mockSetup: func() *mockGit {
				return &mockGit{
					HasUncommittedChangesFn: func() (bool, error) { return true, nil },
					HasUnpushedCommitsFn:    func() (bool, error) { return true, nil },
					IsMergedToOriginFn:      func(targetBranch string) (bool, error) { return false, nil },
				}
			},
			expectedWarnings: []string{
				"You have uncommitted changes",
				"You have unpushed commits",
				"Branch is not merged to origin/main",
			},
		},
		{
			name: "handles errors checking uncommitted changes",
			mockSetup: func() *mockGit {
				return &mockGit{
					HasUncommittedChangesFn: func() (bool, error) {
						return false, fmt.Errorf("git command failed")
					},
					HasUnpushedCommitsFn: func() (bool, error) { return false, nil },
				}
			},
			expectedWarnings: []string{}, // No warnings added on error
			checkStderr:      true,
		},
		{
			name: "handles errors checking unpushed commits",
			mockSetup: func() *mockGit {
				return &mockGit{
					HasUncommittedChangesFn: func() (bool, error) { return false, nil },
					HasUnpushedCommitsFn: func() (bool, error) {
						return false, fmt.Errorf("no upstream branch")
					},
				}
			},
			expectedWarnings: []string{}, // No warnings added on error
			checkStderr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			mockGit := tt.mockSetup()

			deps := &Dependencies{
				Git:    mockGit,
				Stderr: stderr,
			}

			cmd := NewEndCommand(deps, false)
			warnings := cmd.performSafetyChecks()

			// Check warnings count
			if len(warnings) != len(tt.expectedWarnings) {
				t.Errorf("Expected %d warnings, got %d: %v",
					len(tt.expectedWarnings), len(warnings), warnings)
			}

			// Check warning messages
			for i, expected := range tt.expectedWarnings {
				if i >= len(warnings) {
					break
				}
				if warnings[i] != expected {
					t.Errorf("Warning %d: expected %q, got %q", i, expected, warnings[i])
				}
			}

			// Check stderr output if there were errors
			if tt.checkStderr && !strings.Contains(stderr.String(), "⚠ Warning:") {
				t.Error("Expected warning in stderr for error cases")
			}
		})
	}
}

func TestDefaultDependencies(t *testing.T) {
	deps := DefaultDependencies()

	if deps == nil {
		t.Fatal("Expected non-nil dependencies")
	}

	if deps.Git == nil {
		t.Error("Expected Git dependency to be initialized")
	}

	if deps.UI == nil {
		t.Error("Expected UI dependency to be initialized")
	}

	if deps.Detect == nil {
		t.Error("Expected Detect dependency to be initialized")
	}

	if deps.Stdout != os.Stdout {
		t.Error("Expected Stdout to be os.Stdout")
	}

	if deps.Stderr != os.Stderr {
		t.Error("Expected Stderr to be os.Stderr")
	}
}

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
				if !contains(stdout, "✨ Worktree ready at:") {
					t.Error("Expected success message despite setup failure")
				}
			},
		},
		{
			name:        "error creating worktree",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				return &mockGit{
					isGitRepo:           true,
					createWorktreeError: fmt.Errorf("permission denied"),
				}, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "permission denied",
		},
		{
			name:        "error finding env files",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
					findEnvError: fmt.Errorf("failed to find env files"),
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stderr, "⚠ Failed to handle env files") {
					t.Error("Expected env files warning in stderr")
				}
				// Should still complete successfully
				if !contains(stdout, "✨ Worktree ready at:") {
					t.Error("Expected success message despite env files error")
				}
			},
		},
		{
			name:        "error copying env files",
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
					copyEnvError: fmt.Errorf("permission denied"),
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stderr, "⚠ Failed to handle env files") {
					t.Error("Expected env files warning in stderr")
				}
				// Should still complete successfully
				if !contains(stdout, "✨ Worktree ready at:") {
					t.Error("Expected success message despite copy error")
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

func TestCheckoutCommand_Execute(t *testing.T) {
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tests := []struct {
		name          string
		branch        string
		copyEnvs      bool
		mockSetup     func() (*mockGit, *mockUI, *mockDetect, func())
		expectedError string
		checkOutput   func(t *testing.T, stdout, stderr string)
	}{
		{
			name:   "branch does not exist",
			branch: "non-existent-branch",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				return &mockGit{
					isGitRepo: true,
				}, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "branch 'non-existent-branch' does not exist in the repository\nUse 'git branch -a' to see all available branches",
		},
		{
			name:   "successful checkout without env files",
			branch: "feature/test",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					isGitRepo: true,
					envFiles:  []git.EnvFile{},
				}
				// Override BranchExists to return true
				mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
					return true, nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Creating worktree for branch 'feature/test'") {
					t.Error("Expected creation message in stdout")
				}
				if !contains(stdout, "Worktree ready at:") {
					t.Error("Expected worktree ready message in stdout")
				}
			},
		},
		{
			name:     "successful checkout with env files - copyEnvs flag",
			branch:   "feature/test",
			copyEnvs: true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					isGitRepo: true,
					envFiles: []git.EnvFile{
						{Path: ".env", AbsolutePath: "/repo/.env"},
					},
				}
				mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
					return true, nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
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
			name:   "interactive branch selection",
			branch: "", // Empty branch triggers interactive mode
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					isGitRepo: true,
					envFiles:  []git.EnvFile{},
				}
				mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
					return true, nil
				}
				mockGitInstance.ListAllBranchesFn = func() ([]string, error) {
					return []string{"main", "feature/test", "bugfix/123"}, nil
				}
				mockGitInstance.GetCurrentBranchFn = func() (string, error) {
					return "main", nil
				}
				ui := &mockUI{
					ShowSelectorFn: func(title string, items []ui.SelectorItem) (*ui.SelectorItem, error) {
						// Simulate selecting "feature/test"
						for i := range items {
							if items[i].ID == "feature/test" {
								return &items[i], nil
							}
						}
						return &items[0], nil
					},
				}
				return mockGitInstance, ui, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Creating worktree for branch 'feature/test'") {
					t.Error("Expected creation message for selected branch")
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

			cmd := NewCheckoutCommand(deps, tt.copyEnvs)
			err = cmd.Execute(tt.branch)

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

func TestEndCommand_Execute(t *testing.T) {
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tests := []struct {
		name          string
		issueNumber   string
		force         bool
		mockSetup     func() (*mockGit, *mockUI, *mockDetect, func())
		expectedError string
		checkOutput   func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "worktree not found",
			issueNumber: "123",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				mockGitInstance := &mockGit{}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return nil, fmt.Errorf("worktree not found for issue %s", issueNumber)
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "worktree not found for issue 123",
		},
		{
			name:        "error changing to worktree directory",
			issueNumber: "123",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				mockGitInstance := &mockGit{}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					// Return a non-existent path
					return &git.WorktreeInfo{Path: "/nonexistent/path"}, nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "failed to change to worktree directory",
		},
		{
			name:        "successful removal without warnings",
			issueNumber: "123",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{Path: tempDir}, nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Checking worktree for issue #123") {
					t.Error("Expected checking message in stdout")
				}
				if !contains(stdout, "✓ Successfully removed worktree for issue #123") {
					t.Error("Expected success message in stdout")
				}
			},
		},
		{
			name:        "with warnings - user confirms",
			issueNumber: "123",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{Path: tempDir}, nil
				}
				mockGitInstance.HasUncommittedChangesFn = func() (bool, error) {
					return true, nil
				}
				ui := &mockUI{confirmResult: true}
				return mockGitInstance, ui, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Safety check warnings:") {
					t.Error("Expected warnings in stdout")
				}
				if !contains(stdout, "You have uncommitted changes") {
					t.Error("Expected uncommitted changes warning")
				}
				if !contains(stdout, "✓ Successfully removed worktree") {
					t.Error("Expected success message")
				}
			},
		},
		{
			name:        "with warnings - user aborts",
			issueNumber: "123",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{Path: tempDir}, nil
				}
				mockGitInstance.HasUnpushedCommitsFn = func() (bool, error) {
					return true, nil
				}
				ui := &mockUI{confirmResult: false}
				return mockGitInstance, ui, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Safety check warnings:") {
					t.Error("Expected warnings in stdout")
				}
				if !contains(stdout, "Aborted.") {
					t.Error("Expected abort message")
				}
				if contains(stdout, "Successfully removed") {
					t.Error("Should not have removed worktree")
				}
			},
		},
		{
			name:        "force removal bypasses warnings",
			issueNumber: "123",
			force:       true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{Path: tempDir}, nil
				}
				mockGitInstance.HasUncommittedChangesFn = func() (bool, error) {
					return true, nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if contains(stdout, "Safety check warnings:") {
					t.Error("Should not show warnings when forced")
				}
				if !contains(stdout, "✓ Successfully removed worktree") {
					t.Error("Expected success message")
				}
			},
		},
		{
			name: "interactive mode",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{}
				ui := &mockUI{}
				ui.SelectWorktreeFn = func() (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{
						Path:   tempDir,
						Branch: "123/impl",
					}, nil
				}
				return mockGitInstance, ui, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "No issue number provided, entering interactive mode") {
					t.Error("Expected interactive mode message")
				}
				if !contains(stdout, "✓ Successfully removed worktree for issue #123") {
					t.Error("Expected success message")
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

			cmd := NewEndCommand(deps, tt.force)
			err = cmd.Execute(tt.issueNumber)

			// Check error
			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %v", tt.expectedError, err)
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
