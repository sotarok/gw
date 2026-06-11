package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
)

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
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)
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
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, true, true)

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
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, true, false, true) // force = true

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
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	cfg := &config.Config{
		AutoRemoveBranch: true,
	}

	deps.Config = cfg
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
			return false, &git.GitError{
				Args:     []string{"status", "--porcelain"},
				ExitCode: 128,
				Stderr:   "fatal: not a git repository",
			}
		},
	}

	mockUI := &mockUI{}

	deps := &Dependencies{
		Config: config.New(),
		Git:    mockGit,
		UI:     mockUI,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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

// TestCleanCommand_checkWorktree_DeletedWorktreeDir_Integration drives the
// real git.Client (not a mock) against a worktree whose directory has been
// removed from disk, reproducing the user scenario behind the runner fix.
// git itself must report exit 128 ("cannot change to '<dir>'") so the
// broken-worktree short-circuit fires and the user sees exactly one
// "invalid git repository" warning instead of three meaningless ones.
func TestCleanCommand_checkWorktree_DeletedWorktreeDir_Integration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repo := t.TempDir()
	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGit(repo, "init")
	runGit(repo, "config", "user.email", "test@example.com")
	runGit(repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(repo, "add", "f.txt")
	runGit(repo, "commit", "-m", "initial")

	// Create a real worktree on a feature branch, then delete its directory.
	wtPath := filepath.Join(t.TempDir(), "wt-feature")
	runGit(repo, "worktree", "add", wtPath, "-b", "feature/impl")
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("remove worktree dir: %v", err)
	}

	deps := &Dependencies{
		Config: config.New(),
		Git:    git.NewClient(),
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, true, true)

	status := cmd.checkWorktree(&git.WorktreeInfo{Path: wtPath, Branch: "feature/impl"})

	if status.CanRemove {
		t.Errorf("expected CanRemove=false for deleted worktree dir")
	}
	if len(status.Warnings) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d: %v", len(status.Warnings), status.Warnings)
	}
	if status.Warnings[0] != "invalid git repository" {
		t.Errorf("expected single 'invalid git repository' warning, got: %q", status.Warnings[0])
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
		Config: config.New(),
		Git:    mg,
		UI:     &mockUI{},
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)
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
		Config: config.New(),
		Git:    mg,
		UI:     ui,
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)
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
		Config: config.New(),
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mg,
		UI:     &mockUI{},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)

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
		Config: config.New(),
		Git:    mg,
		UI:     ui,
		Stdout: stdout,
		Stderr: stderr,
	}

	cfg := &config.Config{AutoRemoveBranch: true}
	deps.Config = cfg
	cmd := NewCleanCommand(deps, false, false, true)
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
		Config: config.New(),
		Git:    mg,
		UI:     &mockUI{},
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewCleanCommand(deps, false, false, true)
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
		Config: config.New(),
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
		Config: config.New(),
		Git:    mockGit,
		UI:     &mockUI{confirmResult: true},
		Stdout: stdout,
		Stderr: stderr,
	}

	hookCmd := fmt.Sprintf(`printf "%%s:%%s\n" "$PWD" "$GW_BRANCH_NAME" > %q/"$(basename "$PWD")".out`, markerDir)
	cfg := &config.Config{PreEndHook: hookCmd}

	deps.Config = cfg
	cmd := NewCleanCommand(deps, false, false, true)
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
		Config: config.New(),
		Git:    mockGit,
		UI:     &mockUI{confirmResult: true},
		Stdout: stdout,
		Stderr: stderr,
	}

	cfg := &config.Config{PreEndHook: "exit 1"}
	deps.Config = cfg
	cmd := NewCleanCommand(deps, false, false, true)

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
