package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const gitDir = ".git"

// GetRepositoryName returns the name of the current git repository
// Note: In a worktree, this returns the worktree directory name, not the original repository name.
// Use GetOriginalRepositoryName() if you need the original repository name.
func GetRepositoryName() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	repoPath := strings.TrimSpace(string(output))
	return filepath.Base(repoPath), nil
}

// GetOriginalRepositoryName returns the name of the original git repository.
// In a worktree, this returns the name of the main repository, not the worktree directory.
// This is useful for creating new worktrees with consistent naming.
func GetOriginalRepositoryName() (string, error) {
	// Get the git common directory (points to original .git in worktrees)
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	gitCommonDir := strings.TrimSpace(string(output))

	// If it's ".git", we're in the main repository - use show-toplevel
	if gitCommonDir == gitDir {
		return GetRepositoryName()
	}

	// In a worktree, gitCommonDir is an absolute path like /path/to/original-repo/.git
	// We need to get the parent directory's base name
	return filepath.Base(filepath.Dir(gitCommonDir)), nil
}

// IsGitRepository checks if the current directory is inside a git repository
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// GetCurrentBranch returns the name of the current branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ListAllBranches returns all local and remote branches
func ListAllBranches() ([]string, error) {
	// First, fetch to ensure we have latest remote branches
	fetchCmd := exec.Command("git", "fetch", "--prune")
	if err := fetchCmd.Run(); err != nil {
		// Continue even if fetch fails
		fmt.Printf("Warning: failed to fetch latest branches: %v\n", err)
	}

	// Get all branches (local and remote)
	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "HEAD") {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// LocalBranchExists checks if a local branch exists
func LocalBranchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

// RemoteBranchExists checks if a remote branch exists (origin/<branch>)
func RemoteBranchExists(branch string) bool {
	remoteRef := branch
	if !strings.HasPrefix(branch, "origin/") {
		remoteRef = "origin/" + branch
	}
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", remoteRef)
	return cmd.Run() == nil
}

// BranchExists checks if a branch exists (local or remote)
func BranchExists(branch string) (bool, error) {
	// Check if it's a local branch
	if LocalBranchExists(branch) {
		return true, nil
	}

	// Check if it's a remote branch
	if RemoteBranchExists(branch) {
		return true, nil
	}

	return false, nil
}

// DeleteBranch deletes a local git branch
func DeleteBranch(branch string) error {
	// Use -D flag to force delete even if not merged
	cmd := exec.Command("git", "branch", "-D", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %s", branch, string(output))
	}
	return nil
}
