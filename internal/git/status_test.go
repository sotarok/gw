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

func TestHasUnpushedCommits(t *testing.T) {
	t.Run("returns false when all commits are pushed", func(t *testing.T) {
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

		// Create a bare remote repository
		remoteDir, err := os.MkdirTemp("", "test-remote")
		if err != nil {
			t.Fatalf("failed to create remote dir: %v", err)
		}
		defer os.RemoveAll(remoteDir)

		cmd = exec.Command("git", "init", "--bare")
		cmd.Dir = remoteDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init bare repo: %v", err)
		}

		// Add remote
		cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		// Push to remote
		cmd = exec.Command("git", "push", "-u", "origin", "HEAD")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to push: %v", err)
		}

		// Check for unpushed commits
		hasUnpushed, err := HasUnpushedCommits()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if hasUnpushed {
			t.Error("expected no unpushed commits when all commits are pushed")
		}
	})

	t.Run("returns true when there are unpushed commits", func(t *testing.T) {
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

		// Create a bare remote repository
		remoteDir, err := os.MkdirTemp("", "test-remote")
		if err != nil {
			t.Fatalf("failed to create remote dir: %v", err)
		}
		defer os.RemoveAll(remoteDir)

		cmd = exec.Command("git", "init", "--bare")
		cmd.Dir = remoteDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init bare repo: %v", err)
		}

		// Add remote
		cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		// Push to remote
		cmd = exec.Command("git", "push", "-u", "origin", "HEAD")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to push: %v", err)
		}

		// Create another commit (unpushed)
		if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		cmd = exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "unpushed commit")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Check for unpushed commits
		hasUnpushed, err := HasUnpushedCommits()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !hasUnpushed {
			t.Error("expected unpushed commits")
		}
	})

	t.Run("returns true when branch has no upstream", func(t *testing.T) {
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

		// Create a new branch without upstream
		cmd = exec.Command("git", "checkout", "-b", "no-upstream-branch")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Check for unpushed commits - should return true for branch with no upstream
		hasUnpushed, err := HasUnpushedCommits()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !hasUnpushed {
			t.Error("expected unpushed commits for branch with no upstream")
		}
	})
}

func TestIsMergedToOrigin(t *testing.T) {
	t.Run("returns true when branch is merged", func(t *testing.T) {
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

		// Create initial commit on main
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

		// Rename branch to main
		cmd = exec.Command("git", "branch", "-m", "main")
		_ = cmd.Run()

		// Create a bare remote repository
		remoteDir, err := os.MkdirTemp("", "test-remote")
		if err != nil {
			t.Fatalf("failed to create remote dir: %v", err)
		}
		defer os.RemoveAll(remoteDir)

		cmd = exec.Command("git", "init", "--bare")
		cmd.Dir = remoteDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init bare repo: %v", err)
		}

		// Add remote and push main
		cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		cmd = exec.Command("git", "push", "-u", "origin", "main")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to push main: %v", err)
		}

		// Create feature branch
		cmd = exec.Command("git", "checkout", "-b", "feature")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		// Add commit to feature branch
		if err := os.WriteFile(testFile, []byte("feature"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		cmd = exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "feature commit")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Merge feature to main
		cmd = exec.Command("git", "checkout", "main")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout main: %v", err)
		}

		cmd = exec.Command("git", "merge", "feature")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to merge feature: %v", err)
		}

		// Push main with merged changes
		cmd = exec.Command("git", "push", "origin", "main")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to push: %v", err)
		}

		// Switch back to feature branch
		cmd = exec.Command("git", "checkout", "feature")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout feature: %v", err)
		}

		// Check if merged
		isMerged, err := IsMergedToOrigin("main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !isMerged {
			t.Error("expected branch to be merged to origin/main")
		}
	})

	t.Run("returns false when branch is not merged", func(t *testing.T) {
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

		// Create initial commit on main
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

		// Rename branch to main
		cmd = exec.Command("git", "branch", "-m", "main")
		_ = cmd.Run()

		// Create a bare remote repository
		remoteDir, err := os.MkdirTemp("", "test-remote")
		if err != nil {
			t.Fatalf("failed to create remote dir: %v", err)
		}
		defer os.RemoveAll(remoteDir)

		cmd = exec.Command("git", "init", "--bare")
		cmd.Dir = remoteDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init bare repo: %v", err)
		}

		// Add remote and push main
		cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		cmd = exec.Command("git", "push", "-u", "origin", "main")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to push main: %v", err)
		}

		// Create feature branch
		cmd = exec.Command("git", "checkout", "-b", "feature")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		// Add commit to feature branch (not merged)
		if err := os.WriteFile(testFile, []byte("feature"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		cmd = exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "feature commit")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Check if merged (should be false)
		isMerged, err := IsMergedToOrigin("main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if isMerged {
			t.Error("expected branch to not be merged to origin/main")
		}
	})
}
