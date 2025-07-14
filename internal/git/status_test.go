package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Helper function to create a temporary git repository for testing
func createTestRepo(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "test-git-repo")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	_ = cmd.Run()

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestHasUncommittedChanges(t *testing.T) {
	t.Run("returns false for clean repository", func(t *testing.T) {
		tempDir, cleanup := createTestRepo(t)
		defer cleanup()

		// Change to test repo
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

		// Create and commit a file
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial commit")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Now check for uncommitted changes
		hasChanges, err := HasUncommittedChanges()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if hasChanges {
			t.Error("expected no uncommitted changes in clean repository")
		}
	})

	t.Run("returns true for modified files", func(t *testing.T) {
		tempDir, cleanup := createTestRepo(t)
		defer cleanup()

		// Change to test repo
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

		// Create and commit a file
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial commit")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Modify the file
		if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		// Check for uncommitted changes
		hasChanges, err := HasUncommittedChanges()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !hasChanges {
			t.Error("expected uncommitted changes for modified file")
		}
	})

	t.Run("returns true for staged files", func(t *testing.T) {
		tempDir, cleanup := createTestRepo(t)
		defer cleanup()

		// Change to test repo
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

		// Create and add a file without committing
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		// Check for uncommitted changes
		hasChanges, err := HasUncommittedChanges()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !hasChanges {
			t.Error("expected uncommitted changes for staged file")
		}
	})
}

func TestGetCurrentBranch(t *testing.T) {
	t.Run("returns current branch name", func(t *testing.T) {
		tempDir, cleanup := createTestRepo(t)
		defer cleanup()

		// Change to test repo
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

		// Create initial commit
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Get current branch
		branch, err := GetCurrentBranch()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// In a new repo, the default branch might be "master" or "main"
		if branch != "master" && branch != "main" {
			t.Errorf("expected 'master' or 'main', got %q", branch)
		}
	})

	t.Run("returns correct branch after checkout", func(t *testing.T) {
		tempDir, cleanup := createTestRepo(t)
		defer cleanup()

		// Change to test repo
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

		// Create initial commit
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Create and checkout new branch
		cmd = exec.Command("git", "checkout", "-b", "test-branch")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Get current branch
		branch, err := GetCurrentBranch()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if branch != "test-branch" {
			t.Errorf("expected 'test-branch', got %q", branch)
		}
	})
}
