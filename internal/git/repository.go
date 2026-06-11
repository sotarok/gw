package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

const gitDir = ".git"

// showTopLevel returns the absolute path of the current git repository root by
// running `git rev-parse --show-toplevel`. It is the shared implementation
// behind GetRepositoryName and GetRepositoryRoot.
func (c *Client) showTopLevel() (string, error) {
	out, err := c.r.run("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	return out, nil
}

// GetRepositoryName returns the name of the current git repository
// Note: In a worktree, this returns the worktree directory name, not the original repository name.
// Use GetOriginalRepositoryName() if you need the original repository name.
func (c *Client) GetRepositoryName() (string, error) {
	repoPath, err := c.showTopLevel()
	if err != nil {
		return "", err
	}
	return filepath.Base(repoPath), nil
}

// GetRepositoryRoot returns the absolute path of the current git repository root.
// When invoked from a sub directory, this still returns the repository root (not cwd).
// In a worktree, this returns the worktree's root directory, not the main repo's root.
func (c *Client) GetRepositoryRoot() (string, error) {
	return c.showTopLevel()
}

// GetOriginalRepositoryName returns the name of the original git repository.
// In a worktree, this returns the name of the main repository, not the worktree directory.
// This is useful for creating new worktrees with consistent naming.
func (c *Client) GetOriginalRepositoryName() (string, error) {
	// Get the git common directory (points to original .git in worktrees)
	out, err := c.r.run("", "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	// The output may be cwd-relative (".git", "../.git", "../../.git", ...)
	// when called from the main repo or its sub directories, or absolute when
	// called from inside a worktree. Resolve to absolute before extracting the
	// repository directory name — otherwise sub-directory invocations yield
	// "..", "..-{branch}", etc.
	absGitCommonDir, err := filepath.Abs(out)
	if err != nil {
		return "", fmt.Errorf("failed to resolve git common dir: %w", err)
	}

	return filepath.Base(filepath.Dir(absGitCommonDir)), nil
}

// IsGitRepository checks if the current directory is inside a git repository
func (c *Client) IsGitRepository() bool {
	_, err := c.r.run("", "rev-parse", "--git-dir")
	return err == nil
}

// FetchAll fetches from all remotes and prunes deleted remote-tracking branches
func (c *Client) FetchAll() error {
	if _, err := c.r.runCombined("", "fetch", "--all", "--prune"); err != nil {
		return fmt.Errorf("failed to fetch from remotes: %w", err)
	}
	return nil
}

// GetCurrentBranch returns the name of the current branch
func (c *Client) GetCurrentBranch() (string, error) {
	out, err := c.r.run("", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return out, nil
}

// ListAllBranches returns all local and remote branches
func (c *Client) ListAllBranches() ([]string, error) {
	// First, fetch to ensure we have latest remote branches
	if _, err := c.r.run("", "fetch", "--prune"); err != nil {
		// Continue even if fetch fails
		fmt.Printf("Warning: failed to fetch latest branches: %v\n", err)
	}

	// Get all branches (local and remote)
	out, err := c.r.run("", "branch", "-a", "--format=%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(out, "\n")
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
func (c *Client) localBranchExists(branch string) bool {
	_, err := c.r.run("", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// remoteBranchExists checks if a remote branch exists (origin/<branch>)
func (c *Client) remoteBranchExists(branch string) bool {
	remoteRef := branch
	if !strings.HasPrefix(branch, "origin/") {
		remoteRef = "origin/" + branch
	}
	_, err := c.r.run("", "rev-parse", "--verify", "--quiet", remoteRef)
	return err == nil
}

// BranchExists checks if a branch exists (local or remote)
func (c *Client) BranchExists(branch string) (bool, error) {
	// Check if it's a local branch
	if c.localBranchExists(branch) {
		return true, nil
	}

	// Check if it's a remote branch
	if c.remoteBranchExists(branch) {
		return true, nil
	}

	return false, nil
}

// DeleteBranch deletes a local git branch
func (c *Client) DeleteBranch(branch string) error {
	// Use -D flag to force delete even if not merged
	if _, err := c.r.runCombined("", "branch", "-D", branch); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branch, err)
	}
	return nil
}
