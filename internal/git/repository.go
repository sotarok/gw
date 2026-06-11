package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const gitDir = ".git"

// showTopLevel returns the absolute path of the current git repository root by
// running `git rev-parse --show-toplevel`. It is the shared implementation
// behind GetRepositoryName and GetRepositoryRoot.
func showTopLevel() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRepositoryName returns the name of the current git repository
// Note: In a worktree, this returns the worktree directory name, not the original repository name.
// Use GetOriginalRepositoryName() if you need the original repository name.
func GetRepositoryName() (string, error) {
	repoPath, err := showTopLevel()
	if err != nil {
		return "", err
	}
	return filepath.Base(repoPath), nil
}

// GetRepositoryRoot returns the absolute path of the current git repository root.
// When invoked from a sub directory, this still returns the repository root (not cwd).
// In a worktree, this returns the worktree's root directory, not the main repo's root.
func GetRepositoryRoot() (string, error) {
	return showTopLevel()
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

	// The output may be cwd-relative (".git", "../.git", "../../.git", ...)
	// when called from the main repo or its sub directories, or absolute when
	// called from inside a worktree. Resolve to absolute before extracting the
	// repository directory name — otherwise sub-directory invocations yield
	// "..", "..-{branch}", etc.
	absGitCommonDir, err := filepath.Abs(strings.TrimSpace(string(output)))
	if err != nil {
		return "", fmt.Errorf("failed to resolve git common dir: %w", err)
	}

	return filepath.Base(filepath.Dir(absGitCommonDir)), nil
}

// IsGitRepository checks if the current directory is inside a git repository
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// FetchAll fetches from all remotes and prunes deleted remote-tracking branches
func FetchAll() error {
	cmd := exec.Command("git", "fetch", "--all", "--prune")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch from remotes: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
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

// localBranchExists checks if a local branch exists
func localBranchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

// remoteBranchExists checks if a remote branch exists (origin/<branch>)
func remoteBranchExists(branch string) bool {
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
	if localBranchExists(branch) {
		return true, nil
	}

	// Check if it's a remote branch
	if remoteBranchExists(branch) {
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
		return fmt.Errorf("failed to delete branch %s: %s: %w", branch, string(output), err)
	}
	return nil
}
