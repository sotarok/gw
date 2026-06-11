package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/ui"
)

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
				Config: config.New(),
				Stdout: stdout,
				Stderr: stderr,
			}

			deps.Config = &config.Config{}
			cmd := NewCheckoutCommand(deps, tt.copyEnvs, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
	if err := cmd.Execute(testBranchFeature); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "repo-feature-test")
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
	if err := cmd.Execute(testBranchFeature); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, "repo-feature-test")
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	// noFetch=false with FetchBeforeCommand=true should call fetch
	deps.Config = &config.Config{FetchBeforeCommand: true}
	cmd := NewCheckoutCommand(deps, false, false)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{FetchBeforeCommand: true}
	cmd := NewCheckoutCommand(deps, false, false)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	// copyEnvs=false and CopyEnvs=nil should prompt user
	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCheckoutCommand(deps, true, true)
	if err := cmd.Execute(testBranchFeature); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedScanRoot != repoRoot {
		t.Errorf("FindUntrackedEnvFiles called with wrong root.\n  got:  %s\n  want: %s", capturedScanRoot, repoRoot)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{
		PostCheckoutHook: `echo "HOOK_OUTPUT:$GW_WORKTREE_PATH:$GW_BRANCH_NAME:$GW_COMMAND"`,
	}
	cmd := NewCheckoutCommand(deps, false, true)
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
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{
		PostCheckoutHook: "exit 1",
	}
	cmd := NewCheckoutCommand(deps, false, true)
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
