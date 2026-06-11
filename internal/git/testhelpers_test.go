package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

const (
	defaultMainBranch   = "main"
	defaultMasterBranch = "master"
)

// Helper function to run git commands in tests
func runGitCommand(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run git %v: %v", args, err)
	}
}

// Helper function to get the default branch name (main or master)
func getDefaultBranchName(_ *testing.T, dir string) string {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// If we can't get the branch name, assume main
		return defaultMainBranch
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return defaultMainBranch
	}
	return branch
}

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
