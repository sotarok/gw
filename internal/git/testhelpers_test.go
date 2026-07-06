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

// testClient is the shared git.Client used by the package-level test helpers
// below. The helpers exist so the large body of existing integration tests can
// keep calling git operations by short names while every call routes through a
// real *Client method (the production entry point after the runner/Client
// refactor).
var testClient = NewClient()

func RunCommand(command string) error            { return testClient.RunCommand(command) }
func IsGitRepository() bool                      { return testClient.IsGitRepository() }
func GetRepositoryName() (string, error)         { return testClient.GetRepositoryName() }
func GetOriginalRepositoryName() (string, error) { return testClient.GetOriginalRepositoryName() }
func GetRepositoryRoot() (string, error)         { return testClient.GetRepositoryRoot() }
func GetMainRepositoryRoot() (string, error)     { return testClient.GetMainRepositoryRoot() }
func GetCurrentBranch() (string, error)          { return testClient.GetCurrentBranch() }
func FetchAll() error                            { return testClient.FetchAll() }
func ListAllBranches() ([]string, error)         { return testClient.ListAllBranches() }
func BranchExists(branch string) (bool, error)   { return testClient.BranchExists(branch) }
func DeleteBranch(branch string) error           { return testClient.DeleteBranch(branch) }
func ListWorktrees() ([]WorktreeInfo, error)     { return testClient.ListWorktrees() }
func RemoveWorktree(issueNumber string) error    { return testClient.RemoveWorktree(issueNumber) }
func RemoveWorktreeByPath(worktreePath string) error {
	return testClient.RemoveWorktreeByPath(worktreePath)
}
func CreateWorktree(issueNumberOrBranch, baseBranch string) (string, error) {
	return testClient.CreateWorktree(issueNumberOrBranch, baseBranch)
}
func CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error {
	return testClient.CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch)
}
func GetWorktreeForIssue(issueNumberOrBranch string) (*WorktreeInfo, error) {
	return testClient.GetWorktreeForIssue(issueNumberOrBranch)
}
func ResolveBaseBranch(baseBranch string) (string, bool) {
	return testClient.ResolveBaseBranch(baseBranch)
}
func HasUncommittedChanges(worktreePath string) (bool, error) {
	return testClient.HasUncommittedChanges(worktreePath)
}
func HasUnpushedCommits(worktreePath, currentBranch string) (bool, error) {
	return testClient.HasUnpushedCommits(worktreePath, currentBranch)
}
func IsMergedToBaseBranch(worktreePath, currentBranch, targetBranch string) (bool, error) {
	return testClient.IsMergedToBaseBranch(worktreePath, currentBranch, targetBranch)
}
func FindUntrackedEnvFiles(repoPath string) ([]EnvFile, error) {
	return testClient.FindUntrackedEnvFiles(repoPath)
}
func CopyEnvFiles(envFiles []EnvFile, sourceRoot, destRoot string) error {
	return testClient.CopyEnvFiles(envFiles, sourceRoot, destRoot)
}

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
