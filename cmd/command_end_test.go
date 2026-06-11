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
)

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
				if !contains(stderr, "Safety check warnings:") {
					t.Error("Expected warnings in stderr")
				}
				if !contains(stderr, "You have uncommitted changes") {
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
				if !contains(stderr, "Safety check warnings:") {
					t.Error("Expected warnings in stderr")
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
				if contains(stdout, "Safety check warnings:") || contains(stderr, "Safety check warnings:") {
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
