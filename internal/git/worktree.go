package git

import (
	"fmt"
	"os"
	"os/exec"
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

// CreateWorktree creates a new git worktree
func CreateWorktree(issueNumberOrBranch, baseBranch string) (string, error) {
	if !IsGitRepository() {
		return "", fmt.Errorf("not in a git repository")
	}

	repoName, err := GetRepositoryName()
	if err != nil {
		return "", err
	}

	// Determine branch name and directory suffix
	branchName, dirSuffix := DetermineWorktreeNames(issueNumberOrBranch)

	// Create worktree directory name
	worktreeDir := fmt.Sprintf("../%s-%s", repoName, dirSuffix)

	// Create the worktree
	cmd := exec.Command("git", "worktree", "add", worktreeDir, "-b", branchName, baseBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
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
func RemoveWorktree(issueNumberOrBranch string) error {
	if !IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	repoName, err := GetRepositoryName()
	if err != nil {
		return err
	}

	// Determine directory suffix
	_, dirSuffix := DetermineWorktreeNames(issueNumberOrBranch)

	worktreeDir := fmt.Sprintf("../%s-%s", repoName, dirSuffix)
	return RemoveWorktreeByPath(worktreeDir)
}

// RemoveWorktreeByPath removes a git worktree by its path
func RemoveWorktreeByPath(worktreePath string) error {
	if !IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// ListWorktrees returns a list of all worktrees
func ListWorktrees() ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	lines := strings.Split(string(output), "\n")
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
func GetWorktreeForIssue(issueNumberOrBranch string) (*WorktreeInfo, error) {
	repoName, err := GetRepositoryName()
	if err != nil {
		return nil, err
	}

	// Determine directory suffix
	_, dirSuffix := DetermineWorktreeNames(issueNumberOrBranch)

	targetPath := fmt.Sprintf("%s-%s", repoName, dirSuffix)

	worktrees, err := ListWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if strings.Contains(wt.Path, targetPath) {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("worktree for %s not found", issueNumberOrBranch)
}

// CreateWorktreeFromBranch creates a new git worktree from an existing branch
func CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error {
	if !IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Check if source branch starts with origin/
	isRemoteBranch := strings.HasPrefix(sourceBranch, "origin/")

	var cmd *exec.Cmd
	if isRemoteBranch {
		// For remote branches, create a new local branch tracking the remote
		cmd = exec.Command("git", "worktree", "add", worktreePath, "-b", targetBranch, sourceBranch)
	} else {
		// For local branches, just check it out
		cmd = exec.Command("git", "worktree", "add", worktreePath, sourceBranch)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// RunCommand executes a command in the current directory
func RunCommand(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
