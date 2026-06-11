package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WorktreeInfo represents information about a git worktree
type WorktreeInfo struct {
	Path       string
	Branch     string
	Commit     string
	IsDetached bool
	IsCurrent  bool
}

// DetermineWorktreeNames determines the branch name and directory suffix based on input
// If input contains a slash, it's treated as a full branch name
// Otherwise, "/impl" is appended to create the branch name
func DetermineWorktreeNames(input string) (branchName, dirSuffix string) {
	if strings.Contains(input, "/") {
		// Input is a full branch name
		branchName = input
		// Sanitize for directory name
		dirSuffix = SanitizeBranchNameForDirectory(input)
	} else {
		// Input is an issue number or simple identifier
		branchName = fmt.Sprintf("%s/impl", input)
		dirSuffix = input
	}
	return branchName, dirSuffix
}

// ResolveWorktreePath derives the worktree directory path for a given
// repository name and suffix, anchored as a sibling of the repository root:
// `<repoRoot>/../<repoName>-<suffix>`. This is the single source of truth for
// the worktree naming convention shared by creation, checkout and lookup.
func ResolveWorktreePath(repoRoot, repoName, suffix string) string {
	return filepath.Join(repoRoot, "..", fmt.Sprintf("%s-%s", repoName, suffix))
}

// ResolveBaseBranch resolves the base branch, checking local first, then remote
// Returns the resolved branch reference and whether it's a remote branch
func (c *Client) ResolveBaseBranch(baseBranch string) (string, bool) {
	// If the branch exists locally, use it as-is
	if c.localBranchExists(baseBranch) {
		return baseBranch, false
	}

	// If it already starts with origin/, check if it exists
	if strings.HasPrefix(baseBranch, "origin/") {
		if c.remoteBranchExists(baseBranch) {
			return baseBranch, true
		}
		return baseBranch, false
	}

	// Check if it exists as a remote branch
	remoteBranch := "origin/" + baseBranch
	if c.remoteBranchExists(remoteBranch) {
		return remoteBranch, true
	}

	// Return original if nothing found (let git handle the error)
	return baseBranch, false
}

// CreateWorktree creates a new git worktree
func (c *Client) CreateWorktree(issueNumberOrBranch, baseBranch string) (string, error) {
	if !c.IsGitRepository() {
		return "", fmt.Errorf("not in a git repository")
	}

	repoName, err := c.GetOriginalRepositoryName()
	if err != nil {
		return "", err
	}

	// Get repository root directory
	repoRoot, err := c.r.run("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Determine branch name and directory suffix
	branchName, dirSuffix := DetermineWorktreeNames(issueNumberOrBranch)

	// Create worktree directory path relative to repository root
	worktreeDir := ResolveWorktreePath(repoRoot, repoName, dirSuffix)

	// Resolve base branch (check local first, then remote)
	resolvedBaseBranch, _ := c.ResolveBaseBranch(baseBranch)

	// Create the worktree
	if err := c.r.runStreaming("", "worktree", "add", worktreeDir, "-b", branchName, resolvedBaseBranch); err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(worktreeDir)
	if err != nil {
		return worktreeDir, nil
	}

	return absPath, nil
}

// RemoveWorktree removes a git worktree by issue number or branch name
func (c *Client) RemoveWorktree(issueNumberOrBranch string) error {
	if !c.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	repoName, err := c.GetOriginalRepositoryName()
	if err != nil {
		return err
	}

	// Get repository root directory
	repoRoot, err := c.r.run("", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	// Determine directory suffix
	_, dirSuffix := DetermineWorktreeNames(issueNumberOrBranch)

	// Create worktree directory path relative to repository root
	worktreeDir := ResolveWorktreePath(repoRoot, repoName, dirSuffix)
	return c.RemoveWorktreeByPath(worktreeDir)
}

// RemoveWorktreeByPath removes a git worktree by its path
func (c *Client) RemoveWorktreeByPath(worktreePath string) error {
	if !c.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Remove the worktree
	if err := c.r.runStreaming("", "worktree", "remove", worktreePath); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// ListWorktrees returns a list of all worktrees
func (c *Client) ListWorktrees() ([]WorktreeInfo, error) {
	output, err := c.r.run("", "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	lines := strings.Split(output, "\n")
	var current WorktreeInfo

	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "HEAD ") {
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			branch := strings.TrimPrefix(line, "branch ")
			// Remove refs/heads/ prefix if present
			branch = strings.TrimPrefix(branch, "refs/heads/")
			current.Branch = branch
		} else if line == "detached" {
			current.IsDetached = true
		} else if line == "" && current.Path != "" {
			worktrees = append(worktrees, current)
			current = WorktreeInfo{}
		}
	}

	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	// Mark current worktree
	cwd, err := os.Getwd()
	if err == nil {
		for i := range worktrees {
			if absPath, err := filepath.Abs(worktrees[i].Path); err == nil {
				if strings.HasPrefix(cwd, absPath) {
					worktrees[i].IsCurrent = true
					break
				}
			}
		}
	}

	return worktrees, nil
}

// GetWorktreeForIssue finds a worktree for a specific issue number or branch name
func (c *Client) GetWorktreeForIssue(issueNumberOrBranch string) (*WorktreeInfo, error) {
	repoName, err := c.GetOriginalRepositoryName()
	if err != nil {
		return nil, err
	}

	// Determine the branch name and directory suffix
	branchName, dirSuffix := DetermineWorktreeNames(issueNumberOrBranch)

	targetPath := fmt.Sprintf("%s-%s", repoName, dirSuffix)

	worktrees, err := c.ListWorktrees()
	if err != nil {
		return nil, err
	}

	// Match by branch name first, then fall back to the computed directory path.
	// Matching by branch name lets the same worktree be found whether the user
	// passes the issue number ("527") or the full branch name ("527/impl"),
	// which is what shell completion suggests.
	for _, wt := range worktrees {
		if wt.Branch == branchName || strings.Contains(wt.Path, targetPath) {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("worktree for %s not found", issueNumberOrBranch)
}

// CreateWorktreeFromBranch creates a new git worktree from an existing branch
func (c *Client) CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error {
	if !c.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Check if source branch starts with origin/
	isRemoteBranch := strings.HasPrefix(sourceBranch, "origin/")

	var err error
	if isRemoteBranch {
		// For remote branches, create a new local branch tracking the remote
		err = c.r.runStreaming("", "worktree", "add", worktreePath, "-b", targetBranch, sourceBranch)
	} else {
		// For local branches, just check it out
		err = c.r.runStreaming("", "worktree", "add", worktreePath, sourceBranch)
	}

	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// RunCommand executes a command in the current directory
func (c *Client) RunCommand(command string) error {
	return c.r.runShell(command)
}
