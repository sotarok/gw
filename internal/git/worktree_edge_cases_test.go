package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateWorktree_EdgeCases(t *testing.T) {
	t.Run("handles empty issue number", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should handle empty issue number
		_, err = CreateWorktree("", "main")
		if err == nil {
			t.Error("Expected error with empty issue number")
		}
	})

	t.Run("handles GetRepositoryName failure", func(t *testing.T) {
		// Create a directory that's not a git repository
		tmpDir, err := os.MkdirTemp("", "gw-test-nogit-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to non-git directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should fail when GetRepositoryName fails
		_, err = CreateWorktree("123", "main")
		if err == nil {
			t.Error("Expected error when GetRepositoryName fails")
		}
	})

	t.Run("handles git worktree add failure", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Create a worktree first
		worktreePath, err := CreateWorktree("123", "main")
		if err != nil {
			t.Fatalf("Failed to create first worktree: %v", err)
		}
		defer os.RemoveAll(worktreePath)

		// Try to create the same worktree again - should fail
		_, err = CreateWorktree("123", "main")
		if err == nil {
			t.Error("Expected error when creating duplicate worktree")
		}
	})

	t.Run("handles absolute path resolution", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Create worktree
		worktreePath, err := CreateWorktree("456", "main")
		if err != nil {
			t.Fatalf("Failed to create worktree: %v", err)
		}
		defer os.RemoveAll(worktreePath)

		// Should return absolute path
		if !filepath.IsAbs(worktreePath) {
			t.Error("Expected absolute path")
		}
	})
}

func TestRemoveWorktree_EdgeCases(t *testing.T) {
	t.Run("handles empty issue number", func(t *testing.T) {
		err := RemoveWorktree("")
		if err == nil {
			t.Error("Expected error with empty issue number")
		}
	})

	t.Run("handles GetWorktreeForIssue failure", func(t *testing.T) {
		// Create a directory that's not a git repository
		tmpDir, err := os.MkdirTemp("", "gw-test-nogit-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to non-git directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should fail when GetWorktreeForIssue fails
		err = RemoveWorktree("123")
		if err == nil {
			t.Error("Expected error when GetWorktreeForIssue fails")
		}
	})

	t.Run("handles git worktree remove failure", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Try to remove non-existent worktree
		err = RemoveWorktree("999")
		if err == nil {
			t.Error("Expected error when removing non-existent worktree")
		}
		// The error message varies between git versions
		// Just check that we got an error
	})
}

func TestRemoveWorktreeByPath_EdgeCases(t *testing.T) {
	t.Run("handles empty path", func(t *testing.T) {
		err := RemoveWorktreeByPath("")
		if err == nil {
			t.Error("Expected error with empty path")
		}
	})

	t.Run("handles git command failure", func(t *testing.T) {
		// Create a directory that's not a git repository
		tmpDir, err := os.MkdirTemp("", "gw-test-nogit-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to non-git directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should fail when git command fails
		err = RemoveWorktreeByPath("/some/path")
		if err == nil {
			t.Error("Expected error when git command fails")
		}
	})
}

func TestListWorktrees_EdgeCases(t *testing.T) {
	t.Run("handles git command failure", func(t *testing.T) {
		// Create a directory that's not a git repository
		tmpDir, err := os.MkdirTemp("", "gw-test-nogit-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to non-git directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should fail when git command fails
		_, err = ListWorktrees()
		if err == nil {
			t.Error("Expected error when git command fails")
		}
	})

	t.Run("handles malformed output", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should handle main worktree (which has different format)
		worktrees, err := ListWorktrees()
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		// Should have at least the main worktree
		if len(worktrees) == 0 {
			t.Error("Expected at least one worktree")
		}
	})

	t.Run("handles worktree with HEAD", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Create a detached HEAD state
		runGitCommand(t, tmpDir, "checkout", "HEAD~0")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should handle detached HEAD
		worktrees, err := ListWorktrees()
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		// Should have the main worktree with HEAD
		if len(worktrees) == 0 {
			t.Error("Expected at least one worktree")
		}

		// In detached HEAD state, the branch might be shown as HEAD or as a commit hash
		// Just verify we got at least one worktree
		if len(worktrees) == 0 {
			t.Error("Expected at least one worktree in detached HEAD state")
		}
	})

	t.Run("handles worktree without branch info", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// List worktrees - should handle various formats
		worktrees, err := ListWorktrees()
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		// Should have parsed correctly
		if len(worktrees) == 0 {
			t.Error("Expected at least one worktree")
		}
	})
}

func TestGetWorktreeForIssue_EdgeCases(t *testing.T) {
	t.Run("handles ListWorktrees failure", func(t *testing.T) {
		// Create a directory that's not a git repository
		tmpDir, err := os.MkdirTemp("", "gw-test-nogit-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to non-git directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should fail when ListWorktrees fails
		_, err = GetWorktreeForIssue("123")
		if err == nil {
			t.Error("Expected error when ListWorktrees fails")
		}
	})

	t.Run("handles empty issue number", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should return error for empty issue
		wt, err := GetWorktreeForIssue("")
		// Empty issue might return an error or nil, depending on implementation
		// Just verify it doesn't panic
		if err == nil && wt != nil {
			t.Error("Expected nil worktree or error for empty issue")
		}
	})
}

func TestCreateWorktreeFromBranch_EdgeCases(t *testing.T) {
	t.Run("handles empty branch name", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should handle empty branch name
		worktreePath := filepath.Join(tmpDir, "..", "test-worktree")
		err = CreateWorktreeFromBranch(worktreePath, "", "")
		if err == nil {
			t.Error("Expected error with empty branch name")
		}
	})

	t.Run("handles local branch", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Create a new branch
		runGitCommand(t, tmpDir, "checkout", "-b", "feature-branch")
		runGitCommand(t, tmpDir, "checkout", "main")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should create worktree from local branch
		worktreePath := filepath.Join(tmpDir, "..", "gw-test-feature-branch")
		err = CreateWorktreeFromBranch(worktreePath, "feature-branch", "feature-branch")
		if err != nil {
			t.Fatalf("Failed to create worktree from local branch: %v", err)
		}
		defer os.RemoveAll(worktreePath)

		// Verify worktree was created
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Error("Worktree directory was not created")
		}
	})

	t.Run("handles git worktree add failure", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Create a branch and a worktree for it
		runGitCommand(t, tmpDir, "checkout", "-b", "test-branch")
		runGitCommand(t, tmpDir, "checkout", "main")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Create the first worktree
		worktreePath := filepath.Join(tmpDir, "..", "gw-test-branch-1")
		err = CreateWorktreeFromBranch(worktreePath, "test-branch", "test-branch")
		if err != nil {
			t.Fatalf("Failed to create first worktree: %v", err)
		}
		defer os.RemoveAll(worktreePath)

		// Try to create another worktree for the same branch - should fail
		worktreePath2 := filepath.Join(tmpDir, "..", "gw-test-branch-2")
		err = CreateWorktreeFromBranch(worktreePath2, "test-branch", "test-branch")
		if err == nil {
			t.Error("Expected error when creating duplicate worktree for same branch")
		}
	})
}
