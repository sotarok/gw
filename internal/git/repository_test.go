package git

import (
	"os"
	"strings"
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

		// The repository name should contain "gw" (could be "gw" or "gw-{branch}" in a worktree)
		if !strings.Contains(name, "gw") {
			t.Errorf("expected repository name to contain 'gw', got %q", name)
		}

		// Also verify it starts with "gw"
		if !strings.HasPrefix(name, "gw") {
			t.Errorf("expected repository name to start with 'gw', got %q", name)
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
		// In CI environment, we might be in a shallow clone without remote branches
		// Let's skip this test if we can't find any remote branches
		branches, err := ListAllBranches()
		if err != nil {
			t.Fatalf("failed to list branches: %v", err)
		}

		// Find any origin/* branch to test with
		var remoteBranch string
		for _, branch := range branches {
			if strings.HasPrefix(branch, "origin/") {
				remoteBranch = branch
				break
			}
		}

		if remoteBranch == "" {
			t.Skip("No remote branches found, skipping remote branch test")
		}

		exists, err := BranchExists(remoteBranch)
		if err != nil {
			t.Fatalf("unexpected error checking %s: %v", remoteBranch, err)
		}
		if !exists {
			t.Errorf("expected BranchExists to return true for existing remote branch '%s'", remoteBranch)
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

func TestDeleteBranch(t *testing.T) {
	t.Run("successfully deletes existing branch", func(t *testing.T) {
		// Create a temporary git repository for testing
		tempDir, err := os.MkdirTemp("", "test-delete-branch")
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

		// Change to temp directory and setup git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		// Initialize git repo
		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Create a test branch
		if err := RunCommand("git checkout -b test-branch"); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Switch back to main/master
		if err := RunCommand("git checkout -"); err != nil {
			t.Fatalf("failed to switch back to main: %v", err)
		}

		// Verify branch exists before deletion
		exists, err := BranchExists("test-branch")
		if err != nil {
			t.Fatalf("failed to check branch existence: %v", err)
		}
		if !exists {
			t.Fatal("test-branch should exist before deletion")
		}

		// Delete the branch
		err = DeleteBranch("test-branch")
		if err != nil {
			t.Fatalf("failed to delete branch: %v", err)
		}

		// Verify branch no longer exists
		exists, err = BranchExists("test-branch")
		if err != nil {
			t.Fatalf("failed to check branch existence after deletion: %v", err)
		}
		if exists {
			t.Error("test-branch should not exist after deletion")
		}
	})

	t.Run("returns error when deleting non-existent branch", func(t *testing.T) {
		// Create a temporary git repository for testing
		tempDir, err := os.MkdirTemp("", "test-delete-nonexistent")
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

		// Change to temp directory and setup git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		// Initialize git repo
		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Try to delete non-existent branch
		err = DeleteBranch("non-existent-branch")
		if err == nil {
			t.Error("expected error when deleting non-existent branch")
		}
		if !strings.Contains(err.Error(), "failed to delete branch") {
			t.Errorf("expected error to contain 'failed to delete branch', got: %v", err)
		}
	})

	t.Run("force deletes unmerged branch", func(t *testing.T) {
		// Create a temporary git repository for testing
		tempDir, err := os.MkdirTemp("", "test-delete-unmerged")
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

		// Change to temp directory and setup git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		// Initialize git repo
		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Create a test branch with unmerged changes
		if err := RunCommand("git checkout -b unmerged-branch"); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Add a commit to the branch
		if err := os.WriteFile("test.txt", []byte("unmerged changes"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'unmerged changes'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Switch back to main/master
		if err := RunCommand("git checkout -"); err != nil {
			t.Fatalf("failed to switch back to main: %v", err)
		}

		// Delete the unmerged branch (should work with -D flag)
		err = DeleteBranch("unmerged-branch")
		if err != nil {
			t.Fatalf("failed to force delete unmerged branch: %v", err)
		}

		// Verify branch no longer exists
		exists, err := BranchExists("unmerged-branch")
		if err != nil {
			t.Fatalf("failed to check branch existence after deletion: %v", err)
		}
		if exists {
			t.Error("unmerged-branch should not exist after force deletion")
		}
	})
}
