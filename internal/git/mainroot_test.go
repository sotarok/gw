package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetMainRepositoryRoot(t *testing.T) {
	const testRepoName = "main-root-repo"

	t.Run("returns the main worktree root from the main repository", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-main-root")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer func() { _ = os.Chdir(originalDir) }()

		mainRepoPath := filepath.Join(tempDir, testRepoName)
		if err := os.MkdirAll(mainRepoPath, 0755); err != nil {
			t.Fatalf("failed to create main repo dir: %v", err)
		}

		if err := os.Chdir(mainRepoPath); err != nil {
			t.Fatalf("failed to change to main repo dir: %v", err)
		}
		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		root, err := GetMainRepositoryRoot()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resolvedMainRepoPath, err := filepath.EvalSymlinks(mainRepoPath)
		if err != nil {
			t.Fatalf("failed to resolve symlinks for comparison: %v", err)
		}
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatalf("failed to resolve symlinks for root: %v", err)
		}
		if resolvedRoot != resolvedMainRepoPath {
			t.Errorf("expected root %q, got %q", resolvedMainRepoPath, resolvedRoot)
		}
		if filepath.Base(root) == ".git" {
			t.Errorf("expected root to not contain a .git component, got %q", root)
		}
	})

	t.Run("returns the same main root from a linked worktree", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-main-root-worktree")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer func() { _ = os.Chdir(originalDir) }()

		mainRepoPath := filepath.Join(tempDir, testRepoName)
		if err := os.MkdirAll(mainRepoPath, 0755); err != nil {
			t.Fatalf("failed to create main repo dir: %v", err)
		}
		if err := os.Chdir(mainRepoPath); err != nil {
			t.Fatalf("failed to change to main repo dir: %v", err)
		}
		if err := RunCommand("git init"); err != nil {
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

		worktreePath := filepath.Join(tempDir, testRepoName+"-123")
		if err := RunCommand("git worktree add " + worktreePath + " -b 123/impl"); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}
		defer func() {
			_ = os.Chdir(mainRepoPath)
			_ = RunCommand("git worktree remove " + worktreePath)
		}()

		if err := os.Chdir(worktreePath); err != nil {
			t.Fatalf("failed to change to worktree: %v", err)
		}

		root, err := GetMainRepositoryRoot()
		if err != nil {
			t.Fatalf("unexpected error from worktree: %v", err)
		}

		resolvedMainRepoPath, err := filepath.EvalSymlinks(mainRepoPath)
		if err != nil {
			t.Fatalf("failed to resolve symlinks for comparison: %v", err)
		}
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatalf("failed to resolve symlinks for root: %v", err)
		}
		if resolvedRoot != resolvedMainRepoPath {
			t.Errorf("expected root from worktree to be the main repo path %q, got %q", resolvedMainRepoPath, resolvedRoot)
		}
	})

	t.Run("returns error when not in a git repository", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-main-root-non-git")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer func() { _ = os.Chdir(originalDir) }()

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		if _, err := GetMainRepositoryRoot(); err == nil {
			t.Error("expected error when not in a git repository")
		}
	})
}

func TestGetMainRepositoryRoot_SeparateGitDir(t *testing.T) {
	// Regression: for a --separate-git-dir checkout, --git-common-dir's
	// parent is the external git-dir's own parent directory, not the
	// worktree root. GetMainRepositoryRoot must resolve to the actual
	// checkout root (via --show-toplevel) instead of silently reading a
	// .gwrc from that unrelated external location.
	tempDir, err := os.MkdirTemp("", "test-separate-git-dir")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	externalGitDir := filepath.Join(tempDir, "gitstore", "repo.git")
	worktreePath := filepath.Join(tempDir, "wt")
	if err := os.MkdirAll(filepath.Dir(externalGitDir), 0755); err != nil {
		t.Fatalf("failed to create external git-dir parent: %v", err)
	}
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}

	if err := RunCommand("git init --separate-git-dir=" + externalGitDir + " " + worktreePath); err != nil {
		t.Fatalf("failed to init separate-git-dir repo: %v", err)
	}

	if err := os.Chdir(worktreePath); err != nil {
		t.Fatalf("failed to change to worktree dir: %v", err)
	}

	root, err := GetMainRepositoryRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resolvedWorktreePath, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for comparison: %v", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for root: %v", err)
	}
	if resolvedRoot != resolvedWorktreePath {
		t.Errorf("expected root to be the actual checkout root %q, got %q (external git-dir parent would be %q)",
			resolvedWorktreePath, resolvedRoot, filepath.Dir(externalGitDir))
	}
}
