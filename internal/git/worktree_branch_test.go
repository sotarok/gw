package git

import (
	"os"
	"path/filepath"
	"testing"
)

// setupWorktreeBranchRepo creates a repo with an initial commit on the default
// branch plus a linked worktree on "feature/a". It returns the resolved main
// repo root and the resolved linked worktree root.
func setupWorktreeBranchRepo(t *testing.T) (mainRoot, featureRoot string) {
	t.Helper()

	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	mainRepoPath := filepath.Join(tempDir, "wt-branch-repo")
	if err := os.MkdirAll(mainRepoPath, 0755); err != nil {
		t.Fatalf("failed to create main repo dir: %v", err)
	}
	if err := os.Chdir(mainRepoPath); err != nil {
		t.Fatalf("failed to change to main repo dir: %v", err)
	}
	if err := RunCommand("git init -b main"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
	if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
		t.Fatalf("failed to configure git: %v", err)
	}
	if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	featurePath := filepath.Join(tempDir, "wt-branch-repo-a")
	if err := RunCommand("git worktree add " + featurePath + " -b feature/a"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(mainRepoPath)
		_ = RunCommand("git worktree remove --force " + featurePath)
	})

	return resolvePath(t, mainRepoPath), resolvePath(t, featurePath)
}

func resolvePath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for %q: %v", path, err)
	}
	return resolved
}

func TestGetWorktreeRootForBranch(t *testing.T) {
	t.Run("returns the main repo root for the branch checked out there", func(t *testing.T) {
		mainRoot, _ := setupWorktreeBranchRepo(t)

		root, err := GetWorktreeRootForBranch("main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := resolvePath(t, root); got != mainRoot {
			t.Errorf("expected %q, got %q", mainRoot, got)
		}
	})

	t.Run("returns the linked worktree root for a branch checked out there", func(t *testing.T) {
		_, featureRoot := setupWorktreeBranchRepo(t)

		root, err := GetWorktreeRootForBranch("feature/a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := resolvePath(t, root); got != featureRoot {
			t.Errorf("expected %q, got %q", featureRoot, got)
		}
	})

	t.Run("resolves the same root when invoked from inside a linked worktree", func(t *testing.T) {
		mainRoot, featureRoot := setupWorktreeBranchRepo(t)

		if err := os.Chdir(featureRoot); err != nil {
			t.Fatalf("failed to change to worktree: %v", err)
		}

		root, err := GetWorktreeRootForBranch("main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := resolvePath(t, root); got != mainRoot {
			t.Errorf("expected main repo root %q, got %q", mainRoot, got)
		}
	})

	t.Run("returns an error when no worktree has the branch checked out", func(t *testing.T) {
		setupWorktreeBranchRepo(t)

		if err := RunCommand("git branch feature/unchecked main"); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		if _, err := GetWorktreeRootForBranch("feature/unchecked"); err == nil {
			t.Error("expected an error for a branch with no worktree")
		}
	})

	t.Run("returns an error for a remote-tracking ref", func(t *testing.T) {
		setupWorktreeBranchRepo(t)

		if _, err := GetWorktreeRootForBranch("origin/main"); err == nil {
			t.Error("expected an error for a remote-tracking ref")
		}
	})

	t.Run("ignores detached worktrees when the branch name is empty", func(t *testing.T) {
		mainRoot, _ := setupWorktreeBranchRepo(t)

		detachedPath := filepath.Join(filepath.Dir(mainRoot), "wt-branch-repo-detached")
		if err := RunCommand("git worktree add --detach " + detachedPath); err != nil {
			t.Fatalf("failed to create detached worktree: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chdir(mainRoot)
			_ = RunCommand("git worktree remove --force " + detachedPath)
		})

		if _, err := GetWorktreeRootForBranch(""); err == nil {
			t.Error("expected an error for an empty branch name, not a match against a detached worktree")
		}
	})
}
