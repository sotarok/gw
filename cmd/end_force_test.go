package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
)

// runEndCapturingForce runs `gw end` in interactive mode against a stub worktree
// and reports the force flag that reached RemoveWorktreeByPath.
func runEndCapturingForce(t *testing.T, cmdForce bool, mg *mockGit, ui *mockUI) bool {
	t.Helper()

	originalDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	ui.SelectWorktreeFn = func() (*git.WorktreeInfo, error) {
		return &git.WorktreeInfo{Path: tempDir, Branch: testBranch123}, nil
	}

	var capturedForce bool
	mg.RemoveWorktreeByPathWithForceFn = func(_ string, force bool) error {
		capturedForce = force
		return nil
	}

	deps := &Dependencies{
		Config: config.New(),
		Git:    mg,
		UI:     ui,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	if err := NewEndCommand(deps, cmdForce, true, false).Execute(""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return capturedForce
}

func TestEndCommand_ForcePropagation(t *testing.T) {
	t.Run("--force forces the git removal", func(t *testing.T) {
		if force := runEndCapturingForce(t, true, &mockGit{}, &mockUI{}); !force {
			t.Error("--force must reach git worktree remove, otherwise untracked files still block it")
		}
	})

	t.Run("a clean worktree is removed without forcing", func(t *testing.T) {
		mg := &mockGit{
			HasUncommittedChangesFn: func() (bool, error) { return false, nil },
			HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
			IsMergedToBaseBranchFn:  func(string) (bool, error) { return true, nil },
		}
		if force := runEndCapturingForce(t, false, mg, &mockUI{}); force {
			t.Error("nothing warned, so the removal must not be forced")
		}
	})

	t.Run("confirming the safety warnings forces the git removal", func(t *testing.T) {
		mg := &mockGit{
			HasUncommittedChangesFn: func() (bool, error) { return true, nil },
			HasUnpushedCommitsFn:    func() (bool, error) { return false, nil },
			IsMergedToBaseBranchFn:  func(string) (bool, error) { return true, nil },
		}
		ui := &mockUI{confirmResult: true}
		if force := runEndCapturingForce(t, false, mg, ui); !force {
			t.Error("the user accepted the uncommitted-changes warning, so the removal must be forced")
		}
	})
}
