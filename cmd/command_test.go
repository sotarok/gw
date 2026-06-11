package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/ui"
)

// Mock implementations

const testBranch123 = "123/impl"
const testBranchFeature = "feature/test"
const testRepoNameShort = "repo"
const testRepoName = "test-repo"

type mockGit struct {
	isGitRepo           bool
	worktreeExists      bool
	createWorktreeError error
	worktreePath        string
	envFiles            []git.EnvFile
	findEnvError        error
	copyEnvError        error

	// Override functions for custom behavior
	FetchAllFn              func() error
	BranchExistsFn          func(string) (bool, error)
	ListAllBranchesFn       func() ([]string, error)
	GetCurrentBranchFn      func() (string, error)
	GetWorktreeForIssueFn   func(string) (*git.WorktreeInfo, error)
	HasUncommittedChangesFn func() (bool, error)
	HasUnpushedCommitsFn    func() (bool, error)
	IsMergedToBaseBranchFn  func(string) (bool, error)
	// "*AtFn" callbacks receive the same args as the real Git interface
	// methods. Use them when a test needs to vary results by worktree path or
	// branch (the simpler Fn forms above still work for fixed return values).
	HasUncommittedChangesAtFn   func(worktreePath string) (bool, error)
	HasUnpushedCommitsAtFn      func(worktreePath, currentBranch string) (bool, error)
	IsMergedToBaseBranchAtFn    func(worktreePath, currentBranch, targetBranch string) (bool, error)
	DeleteBranchFn              func(string) error
	ListWorktreesFn             func() ([]git.WorktreeInfo, error)
	RemoveWorktreeByPathFn      func(string) error
	GetRepositoryNameFn         func() (string, error)
	GetOriginalRepositoryNameFn func() (string, error)
	GetRepositoryRootFn         func() (string, error)
	CreateWorktreeFromBranchFn  func(string, string, string) error
	FindUntrackedEnvFilesFn     func(string) ([]git.EnvFile, error)
	SanitizeBranchNameForDirFn  func(string) string
}

func (m *mockGit) IsGitRepository() bool {
	return m.isGitRepo
}

func (m *mockGit) GetRepositoryName() (string, error) {
	if m.GetRepositoryNameFn != nil {
		return m.GetRepositoryNameFn()
	}
	return testRepoName, nil
}

func (m *mockGit) GetOriginalRepositoryName() (string, error) {
	if m.GetOriginalRepositoryNameFn != nil {
		return m.GetOriginalRepositoryNameFn()
	}
	// Default to the same value as GetRepositoryName: outside a worktree the
	// original repo name and the current repo name are identical, so tests that
	// only set GetRepositoryNameFn keep working.
	return m.GetRepositoryName()
}

func (m *mockGit) GetRepositoryRoot() (string, error) {
	if m.GetRepositoryRootFn != nil {
		return m.GetRepositoryRootFn()
	}
	// Default to cwd so existing tests that rely on relative `../<name>`
	// behavior keep working — they typically chdir into a temp dir first.
	cwd, _ := os.Getwd()
	return cwd, nil
}

func (m *mockGit) FetchAll() error {
	if m.FetchAllFn != nil {
		return m.FetchAllFn()
	}
	return nil
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
	if m.CreateWorktreeFromBranchFn != nil {
		return m.CreateWorktreeFromBranchFn(worktreePath, sourceBranch, targetBranch)
	}
	// Create the worktree directory for the test
	absolutePath, _ := filepath.Abs(worktreePath)
	os.MkdirAll(absolutePath, 0755)
	return nil
}

func (m *mockGit) RemoveWorktree(issueNumber string) error {
	return nil
}

func (m *mockGit) RemoveWorktreeByPath(worktreePath string) error {
	if m.RemoveWorktreeByPathFn != nil {
		return m.RemoveWorktreeByPathFn(worktreePath)
	}
	return nil
}

func (m *mockGit) ListWorktrees() ([]git.WorktreeInfo, error) {
	if m.ListWorktreesFn != nil {
		return m.ListWorktreesFn()
	}
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

func (m *mockGit) HasUncommittedChanges(worktreePath string) (bool, error) {
	if m.HasUncommittedChangesAtFn != nil {
		return m.HasUncommittedChangesAtFn(worktreePath)
	}
	if m.HasUncommittedChangesFn != nil {
		return m.HasUncommittedChangesFn()
	}
	return false, nil
}

func (m *mockGit) HasUnpushedCommits(worktreePath, currentBranch string) (bool, error) {
	if m.HasUnpushedCommitsAtFn != nil {
		return m.HasUnpushedCommitsAtFn(worktreePath, currentBranch)
	}
	if m.HasUnpushedCommitsFn != nil {
		return m.HasUnpushedCommitsFn()
	}
	return false, nil
}

func (m *mockGit) IsMergedToBaseBranch(worktreePath, currentBranch, targetBranch string) (bool, error) {
	if m.IsMergedToBaseBranchAtFn != nil {
		return m.IsMergedToBaseBranchAtFn(worktreePath, currentBranch, targetBranch)
	}
	if m.IsMergedToBaseBranchFn != nil {
		return m.IsMergedToBaseBranchFn(targetBranch)
	}
	// Default to merged unless overridden
	return true, nil
}

func (m *mockGit) FindUntrackedEnvFiles(repoPath string) ([]git.EnvFile, error) {
	if m.FindUntrackedEnvFilesFn != nil {
		return m.FindUntrackedEnvFilesFn(repoPath)
	}
	if m.findEnvError != nil {
		return nil, m.findEnvError
	}
	return m.envFiles, nil
}

func (m *mockGit) CopyEnvFiles(envFiles []git.EnvFile, sourceRoot, destRoot string) error {
	if m.copyEnvError != nil {
		return m.copyEnvError
	}
	// Actually copy files for testing
	return git.CopyEnvFiles(envFiles, sourceRoot, destRoot)
}

func (m *mockGit) RunCommand(command string) error {
	return nil
}

func (m *mockGit) SanitizeBranchNameForDirectory(branch string) string {
	if m.SanitizeBranchNameForDirFn != nil {
		return m.SanitizeBranchNameForDirFn(branch)
	}
	// Simple sanitization for testing
	return strings.ReplaceAll(branch, "/", "_")
}

func (m *mockGit) DeleteBranch(branch string) error {
	if m.DeleteBranchFn != nil {
		return m.DeleteBranchFn(branch)
	}
	return nil
}

type mockUI struct {
	confirmResult bool
	confirmError  error
	confirmCalled bool

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
	m.confirmCalled = true
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
					IsMergedToBaseBranchFn:  func(targetBranch string) (bool, error) { return false, nil },
				}
			},
			expectedWarnings: []string{
				"You have uncommitted changes",
				"You have unpushed commits",
				"Branch is not merged to main",
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

			cmd := NewEndCommand(deps, false, true)
			warnings := cmd.performSafetyChecks("/test/worktree", "feature/test")

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
		name            string
		issueNumber     string
		baseBranch      string
		copyEnvs        bool
		updateITerm2Tab bool
		mockSetup       func() (*mockGit, *mockUI, *mockDetect, func())
		expectedError   string
		checkOutput     func(t *testing.T, stdout, stderr string)
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
				// Create a real env file in the current directory for testing
				envFile := ".env.test"
				os.WriteFile(envFile, []byte("TEST=value"), 0600)
				return &mockGit{
						isGitRepo:    true,
						worktreePath: tempDir,
						envFiles: []git.EnvFile{
							{Path: ".env.test", AbsolutePath: envFile},
						},
					}, &mockUI{confirmResult: true}, &mockDetect{}, func() {
						os.RemoveAll(tempDir)
						os.Remove(envFile)
					}
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
				// Create a real env file in the current directory for testing
				envFile := ".env.test2"
				os.WriteFile(envFile, []byte("TEST=value"), 0600)
				return &mockGit{
						isGitRepo:    true,
						worktreePath: tempDir,
						envFiles: []git.EnvFile{
							{Path: ".env.test2", AbsolutePath: envFile},
						},
					}, &mockUI{}, &mockDetect{}, func() {
						os.RemoveAll(tempDir)
						os.Remove(envFile)
					}
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Copying environment files:") {
					t.Errorf("Expected copying message in stdout, got:\n%s", stdout)
				}
				if !contains(stdout, "✓ Environment files copied successfully") {
					t.Errorf("Expected copy success message in stdout, got:\n%s", stdout)
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
		{
			name:            "updates iTerm2 tab when enabled",
			issueNumber:     "789",
			baseBranch:      "main",
			updateITerm2Tab: true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				// Check for iTerm2 escape sequence in output
				// Format: "\033]0;test-repo 789\007"
				// Note: We can't test the actual escape sequence output without mocking
				// but we can verify the command completes successfully
				if !contains(stdout, "✨ Worktree ready at:") {
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

			// Create config for test
			cfg := &config.Config{
				AutoCD:          false,
				UpdateITerm2Tab: tt.updateITerm2Tab,
			}
			cmd := NewStartCommandWithConfig(deps, tt.copyEnvs, true, cfg)
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
			branch: testBranchFeature,
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
			branch:   testBranchFeature,
			copyEnvs: true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				// Create a real env file in the current directory for testing
				envFile := ".env.test3"
				os.WriteFile(envFile, []byte("TEST=value"), 0600)
				mockGitInstance := &mockGit{
					isGitRepo: true,
					envFiles: []git.EnvFile{
						{Path: ".env.test3", AbsolutePath: envFile},
					},
				}
				mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
					return true, nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() {
					os.RemoveAll(tempDir)
					os.Remove(envFile)
				}
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
					return []string{defaultBaseBranch, testBranchFeature, "bugfix/123"}, nil
				}
				mockGitInstance.GetCurrentBranchFn = func() (string, error) {
					return defaultBaseBranch, nil
				}
				ui := &mockUI{
					ShowSelectorFn: func(title string, items []ui.SelectorItem) (*ui.SelectorItem, error) {
						// Simulate selecting testBranchFeature
						for i := range items {
							if items[i].ID == testBranchFeature {
								return &items[i], nil
							}
						}
						return &items[0], nil
					},
				}
				return mockGitInstance, ui, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Creating worktree for branch '"+testBranchFeature+"'") {
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

			cmd := NewCheckoutCommandWithConfig(deps, tt.copyEnvs, true, &config.Config{})
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

func TestCheckoutCommand_Execute_GetRepositoryNameError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}
	mockGitInstance.GetRepositoryNameFn = func() (string, error) {
		return "", fmt.Errorf("not a git repository")
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err = cmd.Execute(testBranchFeature)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get repository name") {
		t.Errorf("Expected 'failed to get repository name' error, got: %v", err)
	}
}

func TestCheckoutCommand_Execute_BranchExistsError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return false, fmt.Errorf("git command failed")
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err = cmd.Execute(testBranchFeature)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to check branch existence") {
		t.Errorf("Expected 'failed to check branch existence' error, got: %v", err)
	}
}

func TestCheckoutCommand_Execute_CreateWorktreeFromBranchError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}
	mockGitInstance.CreateWorktreeFromBranchFn = func(worktreePath, sourceBranch, targetBranch string) error {
		return fmt.Errorf("worktree already exists")
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err = cmd.Execute(testBranchFeature)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create worktree") {
		t.Errorf("Expected 'failed to create worktree' error, got: %v", err)
	}
}

func TestCheckoutCommand_Execute_WorktreePathAnchoredToRepoRoot(t *testing.T) {
	// Regression: running `gw checkout` from a sub directory of the repo used
	// to create the worktree as a sibling of the sub directory because the
	// command joined "../<name>" relative to the current working directory.
	// The worktree must be anchored to the repository root instead.
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoRoot := filepath.Join(tempDir, "repo")
	subDir := filepath.Join(repoRoot, "apps", "admin")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub directory: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to chdir into sub directory: %v", err)
	}

	var capturedWorktreePath string
	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles:  []git.EnvFile{},
	}
	mockGitInstance.GetRepositoryNameFn = func() (string, error) { return testRepoNameShort, nil }
	mockGitInstance.GetRepositoryRootFn = func() (string, error) { return repoRoot, nil }
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) { return true, nil }
	mockGitInstance.CreateWorktreeFromBranchFn = func(worktreePath, sourceBranch, targetBranch string) error {
		capturedWorktreePath = worktreePath
		os.MkdirAll(worktreePath, 0755)
		return nil
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	if err := cmd.Execute(testBranchFeature); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "repo-feature_test")
	if filepath.Clean(capturedWorktreePath) != expected {
		t.Errorf("worktree path not anchored to repo root.\n  got:  %s\n  want: %s", capturedWorktreePath, expected)
	}
}

func TestCheckoutCommand_Execute_FromInsideWorktreeUsesOriginalRepoName(t *testing.T) {
	// Regression: running `gw checkout` from *inside* a worktree used the worktree
	// directory name (via GetRepositoryName) when building the new worktree path,
	// producing a doubled name like `<repo>-<branch>-<newbranch>`. The new worktree
	// must be named after the original repository, e.g. `<repo>-<newbranch>`.
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// We are currently sitting inside an existing worktree of "repo".
	currentWorktree := filepath.Join(tempDir, "repo-existing")
	if err := os.MkdirAll(currentWorktree, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}
	if err := os.Chdir(currentWorktree); err != nil {
		t.Fatalf("Failed to chdir into worktree: %v", err)
	}

	var capturedWorktreePath string
	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles:  []git.EnvFile{},
	}
	// Inside a worktree these two differ: GetRepositoryName yields the worktree
	// directory name while GetOriginalRepositoryName yields the real repo name.
	mockGitInstance.GetRepositoryNameFn = func() (string, error) { return "repo-existing", nil }
	mockGitInstance.GetOriginalRepositoryNameFn = func() (string, error) { return testRepoNameShort, nil }
	mockGitInstance.GetRepositoryRootFn = func() (string, error) { return currentWorktree, nil }
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) { return true, nil }
	mockGitInstance.CreateWorktreeFromBranchFn = func(worktreePath, sourceBranch, targetBranch string) error {
		capturedWorktreePath = worktreePath
		os.MkdirAll(worktreePath, 0755)
		return nil
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	if err := cmd.Execute(testBranchFeature); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "repo-feature_test")
	if filepath.Clean(capturedWorktreePath) != expected {
		t.Errorf("worktree path should use the original repo name, not the worktree dir name.\n  got:  %s\n  want: %s",
			capturedWorktreePath, expected)
	}
}

func TestCheckoutCommand_Execute_OriginPrefixStripped(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	var capturedTargetBranch string
	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles:  []git.EnvFile{},
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}
	mockGitInstance.CreateWorktreeFromBranchFn = func(worktreePath, sourceBranch, targetBranch string) error {
		capturedTargetBranch = targetBranch
		absolutePath, _ := filepath.Abs(worktreePath)
		os.MkdirAll(absolutePath, 0755)
		return nil
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err = cmd.Execute("origin/feature/test")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedTargetBranch != testBranchFeature {
		t.Errorf("Expected target branch 'feature/test', got %q", capturedTargetBranch)
	}
}

func TestCheckoutCommand_Execute_SetupFailureWarnsButDoesNotFail(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

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
		Detect: &mockDetect{setupError: fmt.Errorf("yarn install failed")},
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err = cmd.Execute(testBranchFeature)

	if err != nil {
		t.Fatalf("Expected no error (setup failure should not fail), got: %v", err)
	}

	if !contains(stderr.String(), "Setup failed: yarn install failed") {
		t.Error("Expected setup warning in stderr")
	}
	if !contains(stdout.String(), "Worktree ready at:") {
		t.Error("Expected success message despite setup failure")
	}
}

func TestCheckoutCommand_Execute_FetchBeforeCommand(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	fetchCalled := false
	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles:  []git.EnvFile{},
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}
	mockGitInstance.FetchAllFn = func() error {
		fetchCalled = true
		return nil
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

	// noFetch=false with FetchBeforeCommand=true should call fetch
	cmd := NewCheckoutCommandWithConfig(deps, false, false, &config.Config{FetchBeforeCommand: true})
	err = cmd.Execute(testBranchFeature)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !fetchCalled {
		t.Error("Expected FetchAll to be called when FetchBeforeCommand is true")
	}
}

func TestCheckoutCommand_Execute_FetchErrorWarnsButDoesNotFail(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles:  []git.EnvFile{},
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}
	mockGitInstance.FetchAllFn = func() error {
		return fmt.Errorf("network error")
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

	cmd := NewCheckoutCommandWithConfig(deps, false, false, &config.Config{FetchBeforeCommand: true})
	err = cmd.Execute(testBranchFeature)

	if err != nil {
		t.Fatalf("Expected no error (fetch failure should warn), got: %v", err)
	}
	if !contains(stderr.String(), "Could not fetch from remotes") {
		t.Error("Expected fetch warning in stderr")
	}
}

func TestCheckoutCommand_SelectBranch_ListAllBranchesError(t *testing.T) {
	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	mockGitInstance.ListAllBranchesFn = func() ([]string, error) {
		return nil, fmt.Errorf("git branch failed")
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err := cmd.Execute("") // empty branch triggers selectBranch

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list branches") {
		t.Errorf("Expected 'failed to list branches' error, got: %v", err)
	}
}

func TestCheckoutCommand_SelectBranch_NoBranches(t *testing.T) {
	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	mockGitInstance.ListAllBranchesFn = func() ([]string, error) {
		return []string{}, nil
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err := cmd.Execute("")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no branches found") {
		t.Errorf("Expected 'no branches found' error, got: %v", err)
	}
}

func TestCheckoutCommand_SelectBranch_GetCurrentBranchError(t *testing.T) {
	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	mockGitInstance.ListAllBranchesFn = func() ([]string, error) {
		return []string{defaultBaseBranch, testBranchFeature}, nil
	}
	mockGitInstance.GetCurrentBranchFn = func() (string, error) {
		return "", fmt.Errorf("detached HEAD")
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err := cmd.Execute("")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get current branch") {
		t.Errorf("Expected 'failed to get current branch' error, got: %v", err)
	}
}

func TestCheckoutCommand_SelectBranch_AllBranchesFiltered(t *testing.T) {
	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	// Only main/master branches + current branch, all get filtered
	mockGitInstance.ListAllBranchesFn = func() ([]string, error) {
		return []string{defaultBaseBranch, "master", "origin/main", "origin/master"}, nil
	}
	mockGitInstance.GetCurrentBranchFn = func() (string, error) {
		return defaultBaseBranch, nil
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err := cmd.Execute("")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no branches available for checkout") {
		t.Errorf("Expected 'no branches available for checkout' error, got: %v", err)
	}
}

func TestCheckoutCommand_SelectBranch_ShowSelectorError(t *testing.T) {
	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	mockGitInstance.ListAllBranchesFn = func() ([]string, error) {
		return []string{defaultBaseBranch, testBranchFeature}, nil
	}
	mockGitInstance.GetCurrentBranchFn = func() (string, error) {
		return defaultBaseBranch, nil
	}

	mockUIInstance := &mockUI{
		ShowSelectorFn: func(title string, items []ui.SelectorItem) (*ui.SelectorItem, error) {
			return nil, fmt.Errorf("user canceled")
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     mockUIInstance,
		Detect: &mockDetect{},
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err := cmd.Execute("")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "user canceled") {
		t.Errorf("Expected 'user canceled' error, got: %v", err)
	}
}

func TestCheckoutCommand_SelectBranch_FiltersCorrectly(t *testing.T) {
	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles:  []git.EnvFile{},
	}
	mockGitInstance.ListAllBranchesFn = func() ([]string, error) {
		return []string{defaultBaseBranch, "master", "origin/main", "origin/master", "feature/a", "bugfix/b", "origin/feature/c"}, nil
	}
	mockGitInstance.GetCurrentBranchFn = func() (string, error) {
		return "feature/a", nil // current branch should also be filtered
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}
	mockGitInstance.CreateWorktreeFromBranchFn = func(worktreePath, sourceBranch, targetBranch string) error {
		// Do not create real directories; this test only cares about selector filtering.
		return nil
	}

	var selectorItems []ui.SelectorItem
	mockUIInstance := &mockUI{
		ShowSelectorFn: func(title string, items []ui.SelectorItem) (*ui.SelectorItem, error) {
			selectorItems = items
			return &items[0], nil
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     mockUIInstance,
		Detect: &mockDetect{},
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	_ = cmd.Execute("")

	// Should only have "bugfix/b" and "origin/feature/c" (feature/a is current, main/master/origin/main/origin/master are filtered)
	expectedBranches := map[string]bool{"bugfix/b": true, "origin/feature/c": true}
	if len(selectorItems) != len(expectedBranches) {
		t.Errorf("Expected %d filtered branches, got %d: %v", len(expectedBranches), len(selectorItems), selectorItems)
	}
	for _, item := range selectorItems {
		if !expectedBranches[item.ID] {
			t.Errorf("Unexpected branch in selector: %s", item.ID)
		}
	}
}

func TestCheckoutCommand_Execute_EnvFilesWithUserConfirm(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	// Create a real env file
	envFile := ".env.checkout-test"
	os.WriteFile(envFile, []byte("KEY=value"), 0600)
	defer os.Remove(envFile)

	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles: []git.EnvFile{
			{Path: envFile, AbsolutePath: filepath.Join(tempDir, envFile)},
		},
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}

	mockUIInstance := &mockUI{confirmResult: true}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     mockUIInstance,
		Detect: &mockDetect{},
		Stdout: stdout,
		Stderr: stderr,
	}

	// copyEnvs=false and CopyEnvs=nil should prompt user
	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{})
	err = cmd.Execute(testBranchFeature)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !mockUIInstance.confirmCalled {
		t.Error("Expected user to be prompted for env file copy")
	}
	if !contains(stdout.String(), "Found 1 untracked environment file(s):") {
		t.Error("Expected env files message in stdout")
	}
}

func TestStartCommand_Execute_FetchBeforeCommand(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	fetchCalled := false
	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}
	mockGitInstance.FetchAllFn = func() error {
		fetchCalled = true
		return nil
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

	// noFetch=false with FetchBeforeCommand=true should call fetch
	cmd := NewStartCommandWithConfig(deps, false, false, &config.Config{FetchBeforeCommand: true})
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !fetchCalled {
		t.Error("Expected FetchAll to be called when FetchBeforeCommand is true")
	}
}

func TestStartCommand_Execute_FetchErrorWarnsButDoesNotFail(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}
	mockGitInstance.FetchAllFn = func() error {
		return fmt.Errorf("network error")
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

	cmd := NewStartCommandWithConfig(deps, false, false, &config.Config{FetchBeforeCommand: true})
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Expected no error (fetch failure should warn), got: %v", err)
	}
	if !contains(stderr.String(), "Could not fetch from remotes") {
		t.Error("Expected fetch warning in stderr")
	}
	if !contains(stdout.String(), "Worktree ready at:") {
		t.Error("Expected success message despite fetch failure")
	}
}

func TestStartCommand_Execute_NoFetchSkipsFetch(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	fetchCalled := false
	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}
	mockGitInstance.FetchAllFn = func() error {
		fetchCalled = true
		return nil
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

	// noFetch=true should skip fetch even with FetchBeforeCommand=true
	cmd := NewStartCommandWithConfig(deps, false, true, &config.Config{FetchBeforeCommand: true})
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if fetchCalled {
		t.Error("Expected FetchAll NOT to be called when noFetch is true")
	}
}

func TestStartCommand_Execute_AutoCDEnabled(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
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

	cmd := NewStartCommandWithConfig(deps, false, true, &config.Config{AutoCD: true})
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !contains(stdout.String(), "Shell integration will change to this directory") {
		t.Error("Expected shell integration message when AutoCD is enabled")
	}
}

func TestStartCommand_Execute_ITerm2Tab(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
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

	cmd := NewStartCommandWithConfig(deps, false, true, &config.Config{UpdateITerm2Tab: true})
	err = cmd.Execute("456", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Verify command completes successfully with iTerm2 tab update enabled
	if !contains(stdout.String(), "Worktree ready at:") {
		t.Error("Expected success message")
	}
}

func TestStartCommand_Execute_EnvFilesUserDeclines(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	// Create a real env file
	envFile := ".env.decline-test"
	os.WriteFile(envFile, []byte("KEY=value"), 0600)
	defer os.Remove(envFile)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
		envFiles: []git.EnvFile{
			{Path: envFile, AbsolutePath: filepath.Join(tempDir, envFile)},
		},
	}

	mockUIInstance := &mockUI{confirmResult: false}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     mockUIInstance,
		Detect: &mockDetect{},
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewStartCommandWithConfig(deps, false, true, &config.Config{})
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !mockUIInstance.confirmCalled {
		t.Error("Expected user to be prompted")
	}
	// Files should NOT be copied
	if contains(stdout.String(), "Environment files copied successfully") {
		t.Error("Files should not have been copied when user declines")
	}
}

func TestStartCommand_Execute_ConfigCopyEnvsTrue(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	envFile := ".env.configtest"
	os.WriteFile(envFile, []byte("KEY=value"), 0600)
	defer os.Remove(envFile)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
		envFiles: []git.EnvFile{
			{Path: envFile, AbsolutePath: filepath.Join(tempDir, envFile)},
		},
	}

	mockUIInstance := &mockUI{}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     mockUIInstance,
		Detect: &mockDetect{},
		Stdout: stdout,
		Stderr: stderr,
	}

	// copyEnvs flag=false but config.CopyEnvs=true should copy without prompting
	cmd := NewStartCommandWithConfig(deps, false, true, &config.Config{CopyEnvs: boolPtr(true)})
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if mockUIInstance.confirmCalled {
		t.Error("Should not prompt user when config.CopyEnvs is true")
	}
	if !contains(stdout.String(), "Copying environment files:") {
		t.Error("Expected auto-copy message")
	}
	if !contains(stdout.String(), "Environment files copied successfully") {
		t.Error("Expected copy success message")
	}
}

func TestStartCommand_Execute_EnvSourceAnchoredToRepoRoot(t *testing.T) {
	// Regression: env file scan used cwd, so running `gw start` from a sub
	// directory missed env files at the repo root and copied sub-dir env files
	// to the worktree root with wrong relative paths.
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoRoot := filepath.Join(tempDir, "repo")
	subDir := filepath.Join(repoRoot, "apps", "admin")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub directory: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to chdir into sub directory: %v", err)
	}

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	var capturedScanRoot string
	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}
	mockGitInstance.GetRepositoryRootFn = func() (string, error) { return repoRoot, nil }
	mockGitInstance.FindUntrackedEnvFilesFn = func(p string) ([]git.EnvFile, error) {
		capturedScanRoot = p
		return nil, nil
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

	cmd := NewStartCommandWithConfig(deps, true, true, &config.Config{})
	if err := cmd.Execute("123", "main"); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedScanRoot != repoRoot {
		t.Errorf("FindUntrackedEnvFiles called with wrong root.\n  got:  %s\n  want: %s", capturedScanRoot, repoRoot)
	}
}

func TestCheckoutCommand_Execute_EnvSourceAnchoredToRepoRoot(t *testing.T) {
	// Regression: env file scan used cwd, so running `gw checkout` from a sub
	// directory missed env files at the repo root.
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoRoot := filepath.Join(tempDir, "repo")
	subDir := filepath.Join(repoRoot, "apps", "admin")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub directory: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to chdir into sub directory: %v", err)
	}

	var capturedScanRoot string
	mockGitInstance := &mockGit{
		isGitRepo: true,
	}
	mockGitInstance.GetRepositoryNameFn = func() (string, error) { return testRepoNameShort, nil }
	mockGitInstance.GetRepositoryRootFn = func() (string, error) { return repoRoot, nil }
	mockGitInstance.BranchExistsFn = func(string) (bool, error) { return true, nil }
	mockGitInstance.CreateWorktreeFromBranchFn = func(worktreePath, _, _ string) error {
		return os.MkdirAll(worktreePath, 0755)
	}
	mockGitInstance.FindUntrackedEnvFilesFn = func(p string) ([]git.EnvFile, error) {
		capturedScanRoot = p
		return nil, nil
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

	cmd := NewCheckoutCommandWithConfig(deps, true, true, &config.Config{})
	if err := cmd.Execute(testBranchFeature); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedScanRoot != repoRoot {
		t.Errorf("FindUntrackedEnvFiles called with wrong root.\n  got:  %s\n  want: %s", capturedScanRoot, repoRoot)
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

			cmd := NewEndCommand(deps, tt.force, true)
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

func TestEndCommand_BranchDeletion(t *testing.T) {
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tests := []struct {
		name               string
		issueNumber        string
		autoRemoveBranch   bool
		force              bool
		mockSetup          func() (*mockGit, *mockUI, *mockDetect, func())
		expectedBranchCall string // expected branch name passed to DeleteBranch
		expectNoDeletion   bool   // if true, DeleteBranch should not be called
		expectedError      string
	}{
		{
			name:             "auto-remove disabled - branch not deleted",
			issueNumber:      "123",
			autoRemoveBranch: false,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{
						Path:   tempDir,
						Branch: "123/impl",
					}, nil
				}
				deleteCalled := false
				mockGitInstance.DeleteBranchFn = func(branch string) error {
					deleteCalled = true
					t.Error("DeleteBranch should not be called when auto-remove is disabled")
					return nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() {
					os.RemoveAll(tempDir)
					if deleteCalled {
						t.Error("DeleteBranch was called when it shouldn't have been")
					}
				}
			},
			expectNoDeletion: true,
		},
		{
			name:             "auto-remove enabled - branch deleted successfully",
			issueNumber:      "123",
			autoRemoveBranch: true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{
						Path:   tempDir,
						Branch: "123/impl",
					}, nil
				}
				var deletedBranch string
				mockGitInstance.DeleteBranchFn = func(branch string) error {
					deletedBranch = branch
					return nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() {
					os.RemoveAll(tempDir)
					if deletedBranch != "123/impl" {
						t.Errorf("Expected branch '123/impl' to be deleted, got '%s'", deletedBranch)
					}
				}
			},
			expectedBranchCall: "123/impl",
		},
		{
			name:             "auto-remove enabled - branch deletion fails",
			issueNumber:      "123",
			autoRemoveBranch: true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{
						Path:   tempDir,
						Branch: "123/impl",
					}, nil
				}
				mockGitInstance.DeleteBranchFn = func(branch string) error {
					return fmt.Errorf("branch is checked out in another worktree")
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() {
					os.RemoveAll(tempDir)
				}
			},
			expectedBranchCall: "123/impl",
			// Branch deletion error should not fail the command, just show warning
		},
		{
			name:             "auto-remove with custom branch name",
			issueNumber:      "456",
			autoRemoveBranch: true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{
						Path:   tempDir,
						Branch: "feature/issue-456",
					}, nil
				}
				var deletedBranch string
				mockGitInstance.DeleteBranchFn = func(branch string) error {
					deletedBranch = branch
					return nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() {
					os.RemoveAll(tempDir)
					if deletedBranch != "feature/issue-456" {
						t.Errorf("Expected branch 'feature/issue-456' to be deleted, got '%s'", deletedBranch)
					}
				}
			},
			expectedBranchCall: "feature/issue-456",
		},
		{
			name:             "auto-remove with safety checks and force",
			issueNumber:      "789",
			autoRemoveBranch: true,
			force:            true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				mockGitInstance := &mockGit{
					worktreePath: tempDir,
				}
				mockGitInstance.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
					return &git.WorktreeInfo{
						Path:   tempDir,
						Branch: "789/impl",
					}, nil
				}
				mockGitInstance.HasUncommittedChangesFn = func() (bool, error) { return true, nil }
				var deletedBranch string
				mockGitInstance.DeleteBranchFn = func(branch string) error {
					deletedBranch = branch
					return nil
				}
				return mockGitInstance, &mockUI{}, &mockDetect{}, func() {
					os.RemoveAll(tempDir)
					if deletedBranch != "789/impl" {
						t.Errorf("Expected branch '789/impl' to be deleted with force, got '%s'", deletedBranch)
					}
				}
			},
			expectedBranchCall: "789/impl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change to test directory
			testDir := t.TempDir()
			err := os.Chdir(testDir)
			if err != nil {
				t.Fatalf("Failed to change to test directory: %v", err)
			}

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

			// Create config with auto-remove setting
			cfg := &config.Config{
				AutoRemoveBranch: tt.autoRemoveBranch,
			}

			cmd := NewEndCommandWithConfig(deps, tt.force, true, cfg)
			err = cmd.Execute(tt.issueNumber)

			// Check error
			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify success message appears
			if err == nil && !strings.Contains(stdout.String(), "Successfully removed worktree") {
				t.Error("Expected success message in output")
			}
		})
	}
}

// Test copy_envs configuration priority
func TestHandleEnvFiles_ConfigPriority(t *testing.T) {
	tests := []struct {
		name           string
		configCopyEnvs *bool
		copyEnvsFlag   bool
		expectedPrompt bool
		expectedCopy   bool
		userResponse   bool
	}{
		{
			name:           "flag set to true - always copy",
			configCopyEnvs: nil,
			copyEnvsFlag:   true,
			expectedPrompt: false,
			expectedCopy:   true,
		},
		{
			name:           "flag set, config is false - flag overrides config",
			configCopyEnvs: boolPtr(false),
			copyEnvsFlag:   true,
			expectedPrompt: false,
			expectedCopy:   true,
		},
		{
			name:           "config is true, flag not set - use config",
			configCopyEnvs: boolPtr(true),
			copyEnvsFlag:   false,
			expectedPrompt: false,
			expectedCopy:   true,
		},
		{
			name:           "config is false, flag not set - use config (don't copy)",
			configCopyEnvs: boolPtr(false),
			copyEnvsFlag:   false,
			expectedPrompt: false,
			expectedCopy:   false,
		},
		{
			name:           "neither config nor flag set - prompt user (yes)",
			configCopyEnvs: nil,
			copyEnvsFlag:   false,
			expectedPrompt: true,
			expectedCopy:   true,
			userResponse:   true,
		},
		{
			name:           "neither config nor flag set - prompt user (no)",
			configCopyEnvs: nil,
			copyEnvsFlag:   false,
			expectedPrompt: true,
			expectedCopy:   false,
			userResponse:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary directories
			tempDir := t.TempDir()
			originalDir := filepath.Join(tempDir, "original")
			worktreeDir := filepath.Join(tempDir, "worktree")
			os.MkdirAll(originalDir, 0755)
			os.MkdirAll(worktreeDir, 0755)

			// Create a test env file
			envFile := filepath.Join(originalDir, ".env.local")
			os.WriteFile(envFile, []byte("TEST=value"), 0600)

			// Setup mocks
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			mockGit := &mockGit{
				isGitRepo: true,
				envFiles: []git.EnvFile{
					{Path: ".env.local", AbsolutePath: envFile},
				},
			}

			mockUI := &mockUI{
				confirmResult: tt.userResponse,
			}

			deps := &Dependencies{
				Git:    mockGit,
				UI:     mockUI,
				Detect: &mockDetect{},
				Stdout: stdout,
				Stderr: stderr,
			}

			cfg := &config.Config{
				CopyEnvs: tt.configCopyEnvs,
			}

			// Execute handleEnvFiles
			err := handleEnvFiles(deps, cfg, tt.copyEnvsFlag, originalDir, worktreeDir)
			if err != nil {
				t.Fatalf("handleEnvFiles failed: %v", err)
			}

			// Verify prompt behavior
			if tt.expectedPrompt && !mockUI.confirmCalled {
				t.Error("Expected user to be prompted, but wasn't")
			}
			if !tt.expectedPrompt && mockUI.confirmCalled {
				t.Error("Expected no prompt, but user was prompted")
			}

			// Verify copy behavior
			copiedFile := filepath.Join(worktreeDir, ".env.local")
			fileExists := false
			if _, err := os.Stat(copiedFile); err == nil {
				fileExists = true
			}

			if tt.expectedCopy && !fileExists {
				t.Errorf("Expected file to be copied, but it wasn't. Output:\n%s", stdout.String())
			}
			if !tt.expectedCopy && fileExists {
				t.Error("Expected file not to be copied, but it was")
			}
		})
	}
}

// Helper function to create a bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper functions

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// CleanCommand tests

func TestCleanCommand_Execute_NoWorktrees(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			// Only the main worktree
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
			}, nil
		},
	}

	mockUI := &mockUI{}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	output := stdout.String()
	if !contains(output, "No worktrees to remove") {
		t.Errorf("Expected 'No worktrees to remove' message, got: %s", output)
	}
}

func TestCleanCommand_Execute_AllRemovable(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	wt2 := filepath.Join(tmpDir, "wt2")
	os.MkdirAll(wt1, 0755)
	os.MkdirAll(wt2, 0755)

	removedPaths := []string{}

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
				{Path: wt2, Branch: "456/impl"},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) {
			return false, nil
		},
		HasUnpushedCommitsFn: func() (bool, error) {
			return false, nil
		},
		IsMergedToBaseBranchFn: func(branch string) (bool, error) {
			return true, nil
		},
		RemoveWorktreeByPathFn: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	mockUI := &mockUI{
		confirmResult: true,
	}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	// Save and restore current directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	output := stdout.String()
	if !contains(output, "Removable (2)") {
		t.Errorf("Expected 'Removable (2)', got: %s", output)
	}

	if !contains(output, "Successfully removed 2 worktree(s)") {
		t.Errorf("Expected success message for 2 worktrees, got: %s", output)
	}

	if len(removedPaths) != 2 {
		t.Errorf("Expected 2 worktrees to be removed, got: %d", len(removedPaths))
	}
}

func TestCleanCommand_Execute_MixedRemovability(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	wt2 := filepath.Join(tmpDir, "wt2")
	wt3 := filepath.Join(tmpDir, "wt3")
	os.MkdirAll(wt1, 0755)
	os.MkdirAll(wt2, 0755)
	os.MkdirAll(wt3, 0755)

	removedPaths := []string{}

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123}, // Removable
				{Path: wt2, Branch: "456/impl"},    // Has uncommitted changes
				{Path: wt3, Branch: "789/impl"},    // Not merged
			}, nil
		},
		HasUncommittedChangesAtFn: func(worktreePath string) (bool, error) {
			if strings.Contains(worktreePath, "wt2") {
				return true, nil
			}
			return false, nil
		},
		HasUnpushedCommitsFn: func() (bool, error) {
			return false, nil
		},
		IsMergedToBaseBranchAtFn: func(worktreePath, _, _ string) (bool, error) {
			if strings.Contains(worktreePath, "wt3") {
				return false, nil
			}
			return true, nil
		},
		RemoveWorktreeByPathFn: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	mockUI := &mockUI{
		confirmResult: true,
	}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	output := stdout.String()
	if !contains(output, "Removable (1)") {
		t.Errorf("Expected 'Removable (1)', got: %s", output)
	}

	if !contains(output, "Non-removable (2)") {
		t.Errorf("Expected 'Non-removable (2)', got: %s", output)
	}

	if !contains(output, "uncommitted changes") {
		t.Errorf("Expected 'uncommitted changes' warning, got: %s", output)
	}

	if !contains(output, "not merged") {
		t.Errorf("Expected 'not merged' warning, got: %s", output)
	}

	if len(removedPaths) != 1 {
		t.Errorf("Expected 1 worktree to be removed, got: %d", len(removedPaths))
	}

	if len(removedPaths) > 0 && removedPaths[0] != wt1 {
		t.Errorf("Expected wt1 to be removed, got: %s", removedPaths[0])
	}
}

func TestCleanCommand_Execute_DryRun(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	removedPaths := []string{}

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) {
			return false, nil
		},
		HasUnpushedCommitsFn: func() (bool, error) {
			return false, nil
		},
		IsMergedToBaseBranchFn: func(branch string) (bool, error) {
			return true, nil
		},
		RemoveWorktreeByPathFn: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	mockUI := &mockUI{}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, true, true, &config.Config{})

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	output := stdout.String()
	if !contains(output, "Dry-run mode: no changes made") {
		t.Errorf("Expected 'Dry-run mode' message, got: %s", output)
	}

	if len(removedPaths) != 0 {
		t.Errorf("Expected no worktrees to be removed in dry-run, got: %d", len(removedPaths))
	}
}

func TestCleanCommand_Execute_UserDeclines(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	removedPaths := []string{}

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) {
			return false, nil
		},
		HasUnpushedCommitsFn: func() (bool, error) {
			return false, nil
		},
		IsMergedToBaseBranchFn: func(branch string) (bool, error) {
			return true, nil
		},
		RemoveWorktreeByPathFn: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	mockUI := &mockUI{
		confirmResult: false, // User declines
	}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	output := stdout.String()
	if !contains(output, "Aborted") {
		t.Errorf("Expected 'Aborted' message, got: %s", output)
	}

	if len(removedPaths) != 0 {
		t.Errorf("Expected no worktrees to be removed when user declines, got: %d", len(removedPaths))
	}
}

func TestCleanCommand_Execute_Force(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	removedPaths := []string{}

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) {
			return false, nil
		},
		HasUnpushedCommitsFn: func() (bool, error) {
			return false, nil
		},
		IsMergedToBaseBranchFn: func(branch string) (bool, error) {
			return true, nil
		},
		RemoveWorktreeByPathFn: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	mockUI := &mockUI{
		confirmResult: true,
	}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, true, false, true, &config.Config{}) // force = true

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// When force is true, confirmCalled should be false
	if mockUI.confirmCalled {
		t.Error("Expected prompt not to be called when force is true")
	}

	if len(removedPaths) != 1 {
		t.Errorf("Expected 1 worktree to be removed, got: %d", len(removedPaths))
	}
}

func TestCleanCommand_Execute_WithBranchDeletion(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	removedPaths := []string{}
	deletedBranches := []string{}

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) {
			return false, nil
		},
		HasUnpushedCommitsFn: func() (bool, error) {
			return false, nil
		},
		IsMergedToBaseBranchFn: func(branch string) (bool, error) {
			return true, nil
		},
		RemoveWorktreeByPathFn: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
		DeleteBranchFn: func(branch string) error {
			deletedBranches = append(deletedBranches, branch)
			return nil
		},
	}

	mockUI := &mockUI{
		confirmResult: true,
	}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cfg := &config.Config{
		AutoRemoveBranch: true,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, cfg)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(deletedBranches) != 1 {
		t.Errorf("Expected 1 branch to be deleted, got: %d", len(deletedBranches))
	}

	if len(deletedBranches) > 0 && deletedBranches[0] != testBranch123 {
		t.Errorf("Expected branch '%s' to be deleted, got: %s", testBranch123, deletedBranches[0])
	}

	output := stdout.String()
	if !contains(output, "Deleted branch "+testBranch123) {
		t.Errorf("Expected branch deletion message, got: %s", output)
	}
}

func TestCleanCommand_Execute_RemovalError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	wt2 := filepath.Join(tmpDir, "wt2")
	os.MkdirAll(wt1, 0755)
	os.MkdirAll(wt2, 0755)

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
				{Path: wt2, Branch: "456/impl"},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) {
			return false, nil
		},
		HasUnpushedCommitsFn: func() (bool, error) {
			return false, nil
		},
		IsMergedToBaseBranchFn: func(branch string) (bool, error) {
			return true, nil
		},
		RemoveWorktreeByPathFn: func(path string) error {
			if path == wt1 {
				return fmt.Errorf("failed to remove")
			}
			return nil
		},
	}

	mockUI := &mockUI{
		confirmResult: true,
	}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error when removal fails, got nil")
	}

	stderrOutput := stderr.String()
	if !contains(stderrOutput, "Failed to remove") {
		t.Errorf("Expected error message in stderr, got: %s", stderrOutput)
	}

	stdoutOutput := stdout.String()
	if !contains(stdoutOutput, "Successfully removed 1 worktree(s)") {
		t.Errorf("Expected partial success message, got: %s", stdoutOutput)
	}

	if !contains(stderrOutput, "Failed to remove 1 worktree(s)") {
		t.Errorf("Expected failure count in stderr, got: %s", stderrOutput)
	}
}

func TestCleanCommand_Execute_BrokenWorktree(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) {
			// Simulate broken worktree with exit status 128
			return false, fmt.Errorf("fatal: not a git repository: exit status 128")
		},
	}

	mockUI := &mockUI{}

	deps := &Dependencies{
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	output := stdout.String()
	if !contains(output, "Non-removable (1)") {
		t.Errorf("Expected 'Non-removable (1)', got: %s", output)
	}

	if !contains(output, "invalid git repository") {
		t.Errorf("Expected user-friendly error message about broken worktree, got: %s", output)
	}

	if !contains(output, "No worktrees to remove") {
		t.Errorf("Expected 'No worktrees to remove' message, got: %s", output)
	}
}

// Additional EndCommand tests for uncovered paths

func TestEndCommand_Execute_InteractiveSelectError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	ui := &mockUI{}
	ui.SelectWorktreeFn = func() (*git.WorktreeInfo, error) {
		return nil, fmt.Errorf("user canceled selection")
	}

	deps := &Dependencies{
		Git:    &mockGit{},
		UI:     ui,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewEndCommand(deps, false, true)
	err := cmd.Execute("") // empty issue = interactive mode

	if err == nil || !strings.Contains(err.Error(), "user canceled selection") {
		t.Errorf("Expected selection error, got: %v", err)
	}
}

func TestEndCommand_Execute_EmptyIssueFromBranch(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	ui := &mockUI{}
	ui.SelectWorktreeFn = func() (*git.WorktreeInfo, error) {
		return &git.WorktreeInfo{
			Path:   tempDir,
			Branch: "", // empty branch leads to empty issue number
		}, nil
	}

	deps := &Dependencies{
		Git:    &mockGit{},
		UI:     ui,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewEndCommand(deps, false, true)
	err := cmd.Execute("")

	if err == nil || !strings.Contains(err.Error(), "could not determine issue number") {
		t.Errorf("Expected 'could not determine issue number' error, got: %v", err)
	}
}

func TestEndCommand_Execute_ConfirmPromptError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	mg := &mockGit{}
	mg.GetWorktreeForIssueFn = func(issueNumber string) (*git.WorktreeInfo, error) {
		return &git.WorktreeInfo{Path: tempDir, Branch: "123/impl"}, nil
	}
	mg.HasUncommittedChangesFn = func() (bool, error) { return true, nil }

	ui := &mockUI{
		confirmError: fmt.Errorf("prompt failed"),
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     ui,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewEndCommand(deps, false, true)
	err := cmd.Execute("123")

	if err == nil || !strings.Contains(err.Error(), "failed to read response") {
		t.Errorf("Expected 'failed to read response' error, got: %v", err)
	}
}

func TestEndCommand_Execute_RemoveWorktreeByPathError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	mg := &mockGit{}
	ui := &mockUI{}
	ui.SelectWorktreeFn = func() (*git.WorktreeInfo, error) {
		return &git.WorktreeInfo{
			Path:   tempDir,
			Branch: "123/impl",
		}, nil
	}

	mg.RemoveWorktreeByPathFn = func(path string) error {
		return fmt.Errorf("removal failed: directory busy")
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     ui,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewEndCommand(deps, true, true) // force to skip safety checks
	err := cmd.Execute("")                 // interactive mode

	if err == nil || !strings.Contains(err.Error(), "removal failed") {
		t.Errorf("Expected removal error, got: %v", err)
	}
}

func TestEndCommand_PerformSafetyChecks_MergeStatusError(t *testing.T) {
	stderr := &bytes.Buffer{}
	mg := &mockGit{
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
		IsMergedToBaseBranchFn: func(targetBranch string) (bool, error) {
			return false, fmt.Errorf("could not check merge status")
		},
	}

	deps := &Dependencies{
		Git:    mg,
		Stderr: stderr,
	}

	cmd := NewEndCommand(deps, false, true)
	warnings := cmd.performSafetyChecks("/test/worktree", "feature/test")

	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings on error, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(stderr.String(), "Could not check merge status") {
		t.Error("Expected merge status error in stderr")
	}
}

// Additional CleanCommand tests for uncovered paths

func TestCleanCommand_Execute_ListWorktreesError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	mg := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return nil, fmt.Errorf("git command failed")
		},
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     &mockUI{},
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})
	err := cmd.Execute()

	if err == nil || !strings.Contains(err.Error(), "failed to list worktrees") {
		t.Errorf("Expected 'failed to list worktrees' error, got: %v", err)
	}
}

func TestCleanCommand_Execute_ConfirmPromptError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	mg := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
		IsMergedToBaseBranchFn:  func(branch string) (bool, error) { return true, nil },
	}

	ui := &mockUI{
		confirmError: fmt.Errorf("prompt error"),
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     ui,
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})
	err := cmd.Execute()

	if err == nil || !strings.Contains(err.Error(), "failed to read response") {
		t.Errorf("Expected 'failed to read response' error, got: %v", err)
	}
}

func TestCleanCommand_CheckWorktree_UnpushedCommitsError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	mg := &mockGit{
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn: func() (bool, error) {
			return false, fmt.Errorf("no upstream branch configured")
		},
		IsMergedToBaseBranchFn: func(branch string) (bool, error) { return true, nil },
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	info := &git.WorktreeInfo{Path: wt1, Branch: "test/impl"}
	status := cmd.checkWorktree(info)

	if status.CanRemove {
		t.Error("Expected CanRemove to be false when unpushed check errors")
	}
	found := false
	for _, w := range status.Warnings {
		if strings.Contains(w, "Could not check unpushed commits") {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 'Could not check unpushed commits' warning, got: %v", status.Warnings)
	}
}

func TestCleanCommand_CheckWorktree_MergeStatusError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	mg := &mockGit{
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
		IsMergedToBaseBranchFn: func(branch string) (bool, error) {
			return false, fmt.Errorf("merge check failed")
		},
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	info := &git.WorktreeInfo{Path: wt1, Branch: "test/impl"}
	status := cmd.checkWorktree(info)

	if status.CanRemove {
		t.Error("Expected CanRemove to be false when merge check errors")
	}
	found := false
	for _, w := range status.Warnings {
		if strings.Contains(w, "Could not check merge status") {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 'Could not check merge status' warning, got: %v", status.Warnings)
	}
}

func TestCleanCommand_CheckWorktree_UncommittedChangesNon128Error(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	mg := &mockGit{
		HasUncommittedChangesFn: func() (bool, error) {
			return false, fmt.Errorf("some other git error")
		},
		HasUnpushedCommitsFn:   func() (bool, error) { return false, nil },
		IsMergedToBaseBranchFn: func(branch string) (bool, error) { return true, nil },
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	info := &git.WorktreeInfo{Path: wt1, Branch: "test/impl"}
	status := cmd.checkWorktree(info)

	if status.CanRemove {
		t.Error("Expected CanRemove to be false when uncommitted check errors")
	}
	found := false
	for _, w := range status.Warnings {
		if strings.Contains(w, "Could not check uncommitted changes") {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 'Could not check uncommitted changes' warning, got: %v", status.Warnings)
	}
}

func TestCleanCommand_CheckWorktree_UnpushedCommitsTrue(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	mg := &mockGit{
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn:    func() (bool, error) { return true, nil },
		IsMergedToBaseBranchFn:  func(branch string) (bool, error) { return true, nil },
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	info := &git.WorktreeInfo{Path: wt1, Branch: "test/impl"}
	status := cmd.checkWorktree(info)

	if status.CanRemove {
		t.Error("Expected CanRemove to be false when there are unpushed commits")
	}
	found := false
	for _, w := range status.Warnings {
		if w == "unpushed commits" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 'unpushed commits' warning, got: %v", status.Warnings)
	}
}

func TestCleanCommand_CheckWorktree_NotMerged(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	mg := &mockGit{
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
		IsMergedToBaseBranchFn:  func(branch string) (bool, error) { return false, nil },
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})

	info := &git.WorktreeInfo{Path: wt1, Branch: "test/impl"}
	status := cmd.checkWorktree(info)

	if status.CanRemove {
		t.Error("Expected CanRemove to be false when not merged")
	}
	found := false
	for _, w := range status.Warnings {
		if w == "not merged" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 'not merged' warning, got: %v", status.Warnings)
	}
}

func TestCleanCommand_RemoveWorktrees_BranchDeletionError(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wt1, 0755)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	mg := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
		IsMergedToBaseBranchFn:  func(branch string) (bool, error) { return true, nil },
		RemoveWorktreeByPathFn:  func(path string) error { return nil },
		DeleteBranchFn: func(branch string) error {
			return fmt.Errorf("branch deletion failed")
		},
	}

	ui := &mockUI{confirmResult: true}

	deps := &Dependencies{
		Git:    mg,
		UI:     ui,
		Stdout: stdout,
		Stderr: stderr,
	}

	cfg := &config.Config{AutoRemoveBranch: true}
	cmd := NewCleanCommandWithConfig(deps, false, false, true, cfg)
	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error (branch deletion failure should not fail command), got: %v", err)
	}

	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "Failed to delete branch") {
		t.Errorf("Expected branch deletion warning in stderr, got: %s", stderrOutput)
	}
}

func TestCleanCommand_Execute_SkipsMasterAndEmptyBranch(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	mg := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: "/repo2", Branch: "master"},
				{Path: "/repo3", Branch: ""},
			}, nil
		},
	}

	deps := &Dependencies{
		Git:    mg,
		UI:     &mockUI{},
		Stdout: stdout,
		Stderr: stderr,
	}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, &config.Config{})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No worktrees to remove") {
		t.Errorf("Expected 'No worktrees to remove' (all filtered), got: %s", output)
	}
}

func TestNewCleanCommand(t *testing.T) {
	deps := &Dependencies{
		Git:    &mockGit{},
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	cmd := NewCleanCommand(deps, true, false, true)
	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}
	if !cmd.force {
		t.Error("Expected force to be true")
	}
	if cmd.dryRun {
		t.Error("Expected dryRun to be false")
	}
	if !cmd.noFetch {
		t.Error("Expected noFetch to be true")
	}
}

func TestStartCommand_Execute_PostHook(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
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

	cmd := NewStartCommandWithConfig(deps, false, true, &config.Config{
		PostStartHook: `echo "HOOK_OUTPUT:$GW_WORKTREE_PATH:$GW_COMMAND"`,
	})
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !contains(output, "HOOK_OUTPUT:"+worktreeDir+":start") {
		t.Errorf("Expected hook output with worktree path and command, got:\n%s", output)
	}
}

func TestStartCommand_Execute_PostHookFailure(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
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

	cmd := NewStartCommandWithConfig(deps, false, true, &config.Config{
		PostStartHook: "exit 1",
	})
	err = cmd.Execute("123", "main")

	// Command should succeed even if hook fails
	if err != nil {
		t.Fatalf("Expected no error when hook fails, got: %v", err)
	}
	if !contains(stderr.String(), "Post-start hook failed") {
		t.Errorf("Expected warning about hook failure in stderr, got:\n%s", stderr.String())
	}
	// Should still show completion message
	if !contains(stdout.String(), "Worktree ready at:") {
		t.Error("Expected success message even when hook fails")
	}
}

func TestCheckoutCommand_Execute_PostHook(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles:  []git.EnvFile{},
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}
	mockGitInstance.CreateWorktreeFromBranchFn = func(worktreePath, sourceBranch, targetBranch string) error {
		absolutePath, _ := filepath.Abs(worktreePath)
		os.MkdirAll(absolutePath, 0755)
		return nil
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{
		PostCheckoutHook: `echo "HOOK_OUTPUT:$GW_WORKTREE_PATH:$GW_BRANCH_NAME:$GW_COMMAND"`,
	})
	err = cmd.Execute("feature/test")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !contains(output, "HOOK_OUTPUT:") {
		t.Errorf("Expected hook output, got:\n%s", output)
	}
	if !contains(output, ":feature/test:checkout") {
		t.Errorf("Expected branch name and command in hook output, got:\n%s", output)
	}
}

func TestCheckoutCommand_Execute_PostHookFailure(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	mockGitInstance := &mockGit{
		isGitRepo: true,
		envFiles:  []git.EnvFile{},
	}
	mockGitInstance.BranchExistsFn = func(branch string) (bool, error) {
		return true, nil
	}
	mockGitInstance.CreateWorktreeFromBranchFn = func(worktreePath, sourceBranch, targetBranch string) error {
		absolutePath, _ := filepath.Abs(worktreePath)
		os.MkdirAll(absolutePath, 0755)
		return nil
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

	cmd := NewCheckoutCommandWithConfig(deps, false, true, &config.Config{
		PostCheckoutHook: "exit 1",
	})
	err = cmd.Execute("feature/test")

	if err != nil {
		t.Fatalf("Expected no error when hook fails, got: %v", err)
	}
	if !contains(stderr.String(), "Post-checkout hook failed") {
		t.Errorf("Expected warning about hook failure in stderr, got:\n%s", stderr.String())
	}
	if !contains(stdout.String(), "Worktree ready at:") {
		t.Error("Expected success message even when hook fails")
	}
}

func TestEndCommand_Execute_PreEndHook(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	testDir := t.TempDir()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	worktreeDir := t.TempDir()

	mockGitInstance := &mockGit{
		GetWorktreeForIssueFn: func(string) (*git.WorktreeInfo, error) {
			return &git.WorktreeInfo{Path: worktreeDir, Branch: testBranch123}, nil
		},
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

	hookMarker := filepath.Join(t.TempDir(), "hook-ran")
	hookCmd := fmt.Sprintf(`printf "ran:%%s:%%s:%%s:%%s\n" "$PWD" "$GW_WORKTREE_PATH" "$GW_BRANCH_NAME" "$GW_COMMAND" > %q`, hookMarker)

	cfg := &config.Config{PreEndHook: hookCmd}
	cmd := NewEndCommandWithConfig(deps, true, true, cfg)

	if err := cmd.Execute("123"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	content, err := os.ReadFile(hookMarker)
	if err != nil {
		t.Fatalf("Hook marker file not written: %v", err)
	}

	// On macOS /var is a symlink to /private/var, so $PWD and filepath.Abs can
	// differ; resolve both sides to compare the real paths.
	resolvedWorktreeDir, err := filepath.EvalSymlinks(worktreeDir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	parts := strings.SplitN(strings.TrimSuffix(string(content), "\n"), ":", 5)
	if len(parts) != 5 {
		t.Fatalf("Expected 5 fields in hook output, got: %q", string(content))
	}
	gotPWD, _ := filepath.EvalSymlinks(parts[1])
	gotWorktreePath, _ := filepath.EvalSymlinks(parts[2])
	if gotPWD != resolvedWorktreeDir {
		t.Errorf("Expected hook PWD %s, got %s", resolvedWorktreeDir, gotPWD)
	}
	if gotWorktreePath != resolvedWorktreeDir {
		t.Errorf("Expected GW_WORKTREE_PATH %s, got %s", resolvedWorktreeDir, gotWorktreePath)
	}
	if parts[3] != testBranch123 {
		t.Errorf("Expected GW_BRANCH_NAME %s, got %s", testBranch123, parts[3])
	}
	if parts[4] != "end" {
		t.Errorf("Expected GW_COMMAND end, got %s", parts[4])
	}
}

func TestEndCommand_Execute_PreEndHookRunsBeforeRemoval(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	testDir := t.TempDir()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	worktreeDir := t.TempDir()
	sentinel := filepath.Join(t.TempDir(), "hook-ran")

	var sentinelExistsAtRemove bool
	mockGitInstance := &mockGit{
		GetWorktreeForIssueFn: func(string) (*git.WorktreeInfo, error) {
			return &git.WorktreeInfo{Path: worktreeDir, Branch: testBranch123}, nil
		},
	}
	mockGitInstance.RemoveWorktreeByPathFn = func(string) error {
		_, err := os.Stat(sentinel)
		sentinelExistsAtRemove = err == nil
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{confirmResult: true},
		Detect: &mockDetect{},
		Stdout: stdout,
		Stderr: stderr,
	}

	cfg := &config.Config{PreEndHook: fmt.Sprintf("touch %q", sentinel)}

	// Empty issue number → interactive path → RemoveWorktreeByPath (which we observe).
	uiMock := deps.UI.(*mockUI)
	uiMock.SelectWorktreeFn = func() (*git.WorktreeInfo, error) {
		return &git.WorktreeInfo{Path: worktreeDir, Branch: testBranch123}, nil
	}

	cmd := NewEndCommandWithConfig(deps, true, true, cfg)
	if err := cmd.Execute(""); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !sentinelExistsAtRemove {
		t.Error("Expected pre_end_hook to run before RemoveWorktreeByPath, but sentinel did not exist yet")
	}
}

func TestEndCommand_Execute_PreEndHookFailure(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	testDir := t.TempDir()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	worktreeDir := t.TempDir()

	mockGitInstance := &mockGit{
		GetWorktreeForIssueFn: func(string) (*git.WorktreeInfo, error) {
			return &git.WorktreeInfo{Path: worktreeDir, Branch: testBranch123}, nil
		},
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

	cfg := &config.Config{PreEndHook: "exit 1"}
	cmd := NewEndCommandWithConfig(deps, true, true, cfg)

	if err := cmd.Execute("123"); err != nil {
		t.Fatalf("Expected no error even when hook fails, got: %v", err)
	}

	if !contains(stderr.String(), "Pre-end hook failed") {
		t.Errorf("Expected warning about hook failure in stderr, got:\n%s", stderr.String())
	}
	if !contains(stdout.String(), "Successfully removed worktree") {
		t.Errorf("Expected worktree to still be removed on hook failure, got:\n%s", stdout.String())
	}
}

func TestCleanCommand_Execute_PreEndHook(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	wt2 := filepath.Join(tmpDir, "wt2")
	if err := os.MkdirAll(wt1, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(wt2, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	markerDir := t.TempDir()

	removedPaths := []string{}
	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
				{Path: wt2, Branch: "456/impl"},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
		IsMergedToBaseBranchFn:  func(string) (bool, error) { return true, nil },
		RemoveWorktreeByPathFn: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGit,
		UI:     &mockUI{confirmResult: true},
		Stdout: stdout,
		Stderr: stderr,
	}

	hookCmd := fmt.Sprintf(`printf "%%s:%%s\n" "$PWD" "$GW_BRANCH_NAME" > %q/"$(basename "$PWD")".out`, markerDir)
	cfg := &config.Config{PreEndHook: hookCmd}

	cmd := NewCleanCommandWithConfig(deps, false, false, true, cfg)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(removedPaths) != 2 {
		t.Fatalf("Expected 2 worktrees removed, got %d", len(removedPaths))
	}

	for _, wt := range []struct{ path, branch string }{{wt1, testBranch123}, {wt2, "456/impl"}} {
		markerFile := filepath.Join(markerDir, filepath.Base(wt.path)+".out")
		data, err := os.ReadFile(markerFile)
		if err != nil {
			t.Errorf("Hook marker not found for %s: %v", wt.path, err)
			continue
		}
		parts := strings.SplitN(strings.TrimSuffix(string(data), "\n"), ":", 2)
		if len(parts) != 2 {
			t.Errorf("Hook output %q malformed", string(data))
			continue
		}
		resolvedExpected, _ := filepath.EvalSymlinks(wt.path)
		resolvedGot, _ := filepath.EvalSymlinks(parts[0])
		if resolvedGot != resolvedExpected {
			t.Errorf("Hook for %s: expected PWD %s, got %s", wt.path, resolvedExpected, resolvedGot)
		}
		if parts[1] != wt.branch {
			t.Errorf("Hook for %s: expected branch %s, got %s", wt.path, wt.branch, parts[1])
		}
	}
}

func TestCleanCommand_Execute_PreEndHookFailure(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	wt1 := filepath.Join(tmpDir, "wt1")
	if err := os.MkdirAll(wt1, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	removedPaths := []string{}
	mockGit := &mockGit{
		ListWorktreesFn: func() ([]git.WorktreeInfo, error) {
			return []git.WorktreeInfo{
				{Path: "/repo", Branch: "main"},
				{Path: wt1, Branch: testBranch123},
			}, nil
		},
		HasUncommittedChangesFn: func() (bool, error) { return false, nil },
		HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
		IsMergedToBaseBranchFn:  func(string) (bool, error) { return true, nil },
		RemoveWorktreeByPathFn: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGit,
		UI:     &mockUI{confirmResult: true},
		Stdout: stdout,
		Stderr: stderr,
	}

	cfg := &config.Config{PreEndHook: "exit 1"}
	cmd := NewCleanCommandWithConfig(deps, false, false, true, cfg)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !contains(stderr.String(), "Pre-end hook failed") {
		t.Errorf("Expected warning in stderr, got:\n%s", stderr.String())
	}
	if len(removedPaths) != 1 {
		t.Errorf("Expected worktree to still be removed despite hook failure, removedPaths=%v", removedPaths)
	}
}
