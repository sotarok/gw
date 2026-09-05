package git

import (
	"os"
	"path/filepath"
	"testing"
)

// setupDirtyWorktree creates a repo with a linked worktree that holds an
// untracked file, which is exactly what makes `git worktree remove` refuse to
// delete it without --force. It returns the worktree's path.
func setupDirtyWorktree(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	mainRepo := filepath.Join(tempDir, "dirty-repo")
	if err := os.MkdirAll(mainRepo, 0755); err != nil {
		t.Fatalf("failed to create main repo dir: %v", err)
	}
	if err := os.Chdir(mainRepo); err != nil {
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

	worktreePath := filepath.Join(tempDir, "dirty-repo-123")
	if err := RunCommand("git worktree add " + worktreePath + " -b 123/impl"); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktreePath, "scratch.txt"), []byte("untracked"), 0644); err != nil {
		t.Fatalf("failed to create untracked file: %v", err)
	}

	return worktreePath
}

func TestRemoveWorktreeByPath_ForcesPastUntrackedFiles(t *testing.T) {
	t.Run("removes a worktree holding untracked files when forced", func(t *testing.T) {
		worktreePath := setupDirtyWorktree(t)

		if err := RemoveWorktreeByPath(worktreePath, true); err != nil {
			t.Fatalf("forced removal should succeed, got: %v", err)
		}
		if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
			t.Errorf("worktree directory should be gone, stat gave: %v", err)
		}
	})

	t.Run("refuses a worktree holding untracked files when not forced", func(t *testing.T) {
		worktreePath := setupDirtyWorktree(t)

		if err := RemoveWorktreeByPath(worktreePath, false); err == nil {
			t.Error("unforced removal should fail while untracked files remain")
		}
		if _, err := os.Stat(worktreePath); err != nil {
			t.Errorf("worktree directory should still exist, stat gave: %v", err)
		}
	})
}
