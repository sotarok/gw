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
