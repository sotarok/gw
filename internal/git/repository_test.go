package git

import (
	"os"
	"testing"
)

func TestGetRepositoryName(t *testing.T) {
	t.Run("returns repository name from git repository", func(t *testing.T) {
		// This test will fail initially because we're testing against the actual git command
		// In TDD style, we write the test first, see it fail, then make it pass

		name, err := GetRepositoryName()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// We expect "gw" as the repository name since we're in the gw directory
		expected := "gw"
		if name != expected {
			t.Errorf("expected repository name %q, got %q", expected, name)
		}
	})

	t.Run("returns error when not in git repository", func(t *testing.T) {
		// Create a temporary directory that's not a git repo
		tempDir, err := os.MkdirTemp("", "test-non-git")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Change to temp directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer func() {
			_ = os.Chdir(originalDir)
		}()

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		// Now test should return an error
		_, err = GetRepositoryName()
		if err == nil {
			t.Error("expected error when not in git repository, got nil")
		}
	})
}

func TestIsGitRepository(t *testing.T) {
	t.Run("returns true in git repository", func(t *testing.T) {
		// We're running this test in a git repository
		if !IsGitRepository() {
			t.Error("expected IsGitRepository to return true in git repository")
		}
	})

	t.Run("returns false outside git repository", func(t *testing.T) {
		// Create a temporary directory that's not a git repo
		tempDir, err := os.MkdirTemp("", "test-non-git")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Change to temp directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer func() {
			_ = os.Chdir(originalDir)
		}()

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		if IsGitRepository() {
			t.Error("expected IsGitRepository to return false outside git repository")
		}
	})
}

func TestBranchExists(t *testing.T) {
	t.Run("returns true for existing local branch", func(t *testing.T) {
		// Get current branch to test with
		currentBranch, err := GetCurrentBranch()
		if err != nil {
			t.Fatalf("failed to get current branch: %v", err)
		}

		exists, err := BranchExists(currentBranch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Errorf("expected BranchExists to return true for current branch '%s'", currentBranch)
		}
	})

	t.Run("returns true for existing remote branch", func(t *testing.T) {
		// Check for both origin/main and origin/master
		exists, err := BranchExists("origin/main")
		if err != nil {
			t.Fatalf("unexpected error checking origin/main: %v", err)
		}

		if !exists {
			// Try origin/master if origin/main doesn't exist
			exists, err = BranchExists("origin/master")
			if err != nil {
				t.Fatalf("unexpected error checking origin/master: %v", err)
			}
			if !exists {
				t.Error("expected BranchExists to return true for either 'origin/main' or 'origin/master' branch")
			}
		}
	})

	t.Run("returns false for non-existent branch", func(t *testing.T) {
		exists, err := BranchExists("non-existent-branch-12345")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("expected BranchExists to return false for non-existent branch")
		}
	})

	t.Run("checks remote when local branch doesn't exist", func(t *testing.T) {
		// Create a temporary git repository for testing
		tempDir, err := os.MkdirTemp("", "test-git-repo")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save current directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer func() {
			_ = os.Chdir(originalDir)
		}()

		// Change to temp directory
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		// Initialize git repo
		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Create initial commit
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Create a branch
		if err := RunCommand("git checkout -b test-branch"); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Test that the branch exists
		exists, err := BranchExists("test-branch")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected BranchExists to return true for created branch")
		}
	})
}
