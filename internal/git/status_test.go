package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	mainBranch   = "main"
	masterBranch = "master"
)

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
		hasChanges, err := HasUncommittedChanges(tempDir)
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
		hasChanges, err := HasUncommittedChanges(tempDir)
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
		hasChanges, err := HasUncommittedChanges(tempDir)
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
		if branch != masterBranch && branch != mainBranch {
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
		branch, _ := GetCurrentBranch()
		hasUnpushed, err := HasUnpushedCommits(tempDir, branch)
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
		branch, _ := GetCurrentBranch()
		hasUnpushed, err := HasUnpushedCommits(tempDir, branch)
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

		// Create a new branch without upstream, with its own commit so it has
		// work that is neither pushed nor merged into the base branch.
		cmd = exec.Command("git", "checkout", "-b", "no-upstream-branch")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		if err := os.WriteFile(testFile, []byte("no upstream change"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}
		if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}
		if err := exec.Command("git", "commit", "-m", "no upstream commit").Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Check for unpushed commits - should return true for branch with no upstream
		branch, _ := GetCurrentBranch()
		hasUnpushed, err := HasUnpushedCommits(tempDir, branch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !hasUnpushed {
			t.Error("expected unpushed commits for branch with no upstream")
		}
	})

	t.Run("returns false when branch is merged and remote deleted", func(t *testing.T) {
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
		if err := os.WriteFile(testFile, []byte("feature content"), 0644); err != nil {
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

		// Push feature branch
		cmd = exec.Command("git", "push", "-u", "origin", "feature")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to push feature: %v", err)
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

		// Delete remote branch (simulate PR merge with branch deletion)
		cmd = exec.Command("git", "push", "origin", "--delete", "feature")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to delete remote branch: %v", err)
		}

		// Unset the upstream to simulate the "[gone]" state
		cmd = exec.Command("git", "branch", "--unset-upstream")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to unset upstream: %v", err)
		}

		// Now check for unpushed commits
		// Should return false because the branch is merged even though upstream is gone
		branch, _ := GetCurrentBranch()
		hasUnpushed, err := HasUnpushedCommits(tempDir, branch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if hasUnpushed {
			t.Error("expected no unpushed commits for merged branch with deleted remote")
		}
	})
}

func TestIsMergedToBaseBranch(t *testing.T) {
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
		branch, _ := GetCurrentBranch()
		isMerged, err := IsMergedToBaseBranch(tempDir, branch, "main")
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
		branch, _ := GetCurrentBranch()
		isMerged, err := IsMergedToBaseBranch(tempDir, branch, "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if isMerged {
			t.Error("expected branch to not be merged to origin/main")
		}
	})

	t.Run("returns true when merged to local main but not pushed", func(t *testing.T) {
		tempDir, cleanup := createTestRepo(t)
		defer cleanup()

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

		// Initial commit on main (no remote configured at all).
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}
		if err := exec.Command("git", "commit", "-m", "initial").Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}
		cmd := exec.Command("git", "branch", "-m", "main")
		_ = cmd.Run()

		// Feature branch with a commit.
		if err := exec.Command("git", "checkout", "-b", "feature").Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}
		if err := os.WriteFile(testFile, []byte("feature"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}
		if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}
		if err := exec.Command("git", "commit", "-m", "feature commit").Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Merge into local main, but never push (there is no remote).
		if err := exec.Command("git", "checkout", "main").Run(); err != nil {
			t.Fatalf("failed to checkout main: %v", err)
		}
		if err := exec.Command("git", "merge", "feature").Run(); err != nil {
			t.Fatalf("failed to merge feature: %v", err)
		}
		if err := exec.Command("git", "checkout", "feature").Run(); err != nil {
			t.Fatalf("failed to checkout feature: %v", err)
		}

		isMerged, err := IsMergedToBaseBranch(tempDir, "feature", "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isMerged {
			t.Error("expected branch merged into local main to be considered merged")
		}
	})
}
