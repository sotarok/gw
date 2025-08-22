package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListAllBranches_EdgeCases(t *testing.T) {
	t.Run("handles remote branches", func(t *testing.T) {
		// Create temporary directory for main repo
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a bare repository to act as remote
		remoteDir, err := os.MkdirTemp("", "gw-remote-*")
		if err != nil {
			t.Fatalf("Failed to create remote dir: %v", err)
		}
		defer os.RemoveAll(remoteDir)

		// Initialize bare repository
		runGitCommand(t, remoteDir, "init", "--bare")

		// Initialize main repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Get the default branch name
		defaultBranch := getDefaultBranchName(t, tmpDir)

		// Add remote and push
		runGitCommand(t, tmpDir, "remote", "add", "origin", remoteDir)
		runGitCommand(t, tmpDir, "push", "-u", "origin", defaultBranch)

		// Create and push additional branches
		runGitCommand(t, tmpDir, "checkout", "-b", "feature-1")
		os.WriteFile(testFile, []byte("feature 1"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Feature 1")
		runGitCommand(t, tmpDir, "push", "-u", "origin", "feature-1")

		runGitCommand(t, tmpDir, "checkout", "-b", "feature-2")
		os.WriteFile(testFile, []byte("feature 2"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Feature 2")
		runGitCommand(t, tmpDir, "push", "-u", "origin", "feature-2")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// List all branches including remotes
		branches, err := ListAllBranches()
		if err != nil {
			t.Fatalf("ListAllBranches failed: %v", err)
		}

		// Should have both local and remote branches
		const testMainBranch = "main"
		expectedBranches := map[string]bool{
			testMainBranch:             false,
			"feature-1":                false,
			"feature-2":                false,
			"origin/" + testMainBranch: false,
			"origin/feature-1":         false,
			"origin/feature-2":         false,
		}

		for _, branch := range branches {
			if _, ok := expectedBranches[branch]; ok {
				expectedBranches[branch] = true
			}
		}

		// Check all expected branches were found
		for branch, found := range expectedBranches {
			if !found {
				t.Errorf("Expected branch %s not found", branch)
			}
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
		_, err = ListAllBranches()
		if err == nil {
			t.Error("Expected error when git command fails")
		}
	})

	t.Run("handles empty output with remote", func(t *testing.T) {
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

		// Add a fake remote (doesn't actually exist)
		runGitCommand(t, tmpDir, "remote", "add", "origin", "https://example.com/repo.git")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Should handle case where remote exists but no remote branches
		branches, err := ListAllBranches()
		if err != nil {
			t.Fatalf("ListAllBranches failed: %v", err)
		}

		// Should at least have main branch
		const testMainBranch = "main"
		found := false
		for _, branch := range branches {
			if branch == testMainBranch {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find main branch")
		}
	})
}

func TestBranchExists_EdgeCases(t *testing.T) {
	t.Run("handles remote branch notation", func(t *testing.T) {
		// Create temporary directory for main repo
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a bare repository to act as remote
		remoteDir, err := os.MkdirTemp("", "gw-remote-*")
		if err != nil {
			t.Fatalf("Failed to create remote dir: %v", err)
		}
		defer os.RemoveAll(remoteDir)

		// Initialize bare repository
		runGitCommand(t, remoteDir, "init", "--bare")

		// Initialize main repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create initial commit
		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

		// Get the default branch name
		defaultBranch := getDefaultBranchName(t, tmpDir)

		// Add remote and push
		runGitCommand(t, tmpDir, "remote", "add", "origin", remoteDir)
		runGitCommand(t, tmpDir, "push", "-u", "origin", defaultBranch)

		// Create remote-only branch
		runGitCommand(t, tmpDir, "checkout", "-b", "remote-only")
		os.WriteFile(testFile, []byte("remote"), 0644)
		runGitCommand(t, tmpDir, "add", "test.txt")
		runGitCommand(t, tmpDir, "commit", "-m", "Remote branch")
		runGitCommand(t, tmpDir, "push", "-u", "origin", "remote-only")
		runGitCommand(t, tmpDir, "checkout", defaultBranch)
		runGitCommand(t, tmpDir, "branch", "-D", "remote-only") // Delete local branch

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Check local branch exists
		exists, err := BranchExists(defaultBranch)
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if !exists {
			t.Errorf("Expected %s branch to exist", defaultBranch)
		}

		// Check remote-only branch exists
		exists, err = BranchExists("origin/remote-only")
		if err != nil {
			t.Fatalf("BranchExists failed for remote branch: %v", err)
		}
		if !exists {
			t.Error("Expected origin/remote-only branch to exist")
		}

		// Check non-existent branch
		exists, err = BranchExists("non-existent")
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if exists {
			t.Error("Expected non-existent branch to not exist")
		}
	})

	t.Run("handles ListAllBranches failure", func(t *testing.T) {
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

		// BranchExists might not return an error for non-git directories
		// It might just return false
		exists, err := BranchExists("main")
		// Either it returns an error or false
		if err == nil && exists {
			t.Error("Expected false or error when not in git repository")
		}
	})

	t.Run("handles special characters in branch name", func(t *testing.T) {
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

		// Create branch with special characters (valid in git)
		specialBranch := "feature/test-123_impl"
		runGitCommand(t, tmpDir, "checkout", "-b", specialBranch)

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Check special branch exists
		exists, err := BranchExists(specialBranch)
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if !exists {
			t.Errorf("Expected branch %s to exist", specialBranch)
		}
	})

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

		// Empty branch name should not exist
		exists, err := BranchExists("")
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if exists {
			t.Error("Expected empty branch name to not exist")
		}
	})

	t.Run("handles branch with asterisk", func(t *testing.T) {
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

		// Create current branch
		runGitCommand(t, tmpDir, "checkout", "-b", "current-branch")

		// Change to the repository directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// List branches to see format with asterisk
		branches, err := ListAllBranches()
		if err != nil {
			t.Fatalf("ListAllBranches failed: %v", err)
		}

		// Check that current branch is properly handled (with or without asterisk)
		hasCurrentBranch := false
		for _, branch := range branches {
			// Branch might have asterisk prefix from git branch output
			cleanBranch := strings.TrimPrefix(branch, "* ")
			if cleanBranch == "current-branch" {
				hasCurrentBranch = true
				break
			}
		}

		if !hasCurrentBranch {
			t.Error("Expected to find current-branch in list")
		}

		// BranchExists should work correctly
		exists, err := BranchExists("current-branch")
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if !exists {
			t.Error("Expected current-branch to exist")
		}
	})
}
