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

// CreateWorktree creates a new git worktree
func CreateWorktree(issueNumber, baseBranch string) (string, error) {
	if !IsGitRepository() {
		return "", fmt.Errorf("not in a git repository")
	}

	repoName, err := GetRepositoryName()
	if err != nil {
		return "", err
	}

	// Create worktree directory name
	worktreeDir := fmt.Sprintf("../%s-%s", repoName, issueNumber)
	branchName := fmt.Sprintf("%s/impl", issueNumber)

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

// RemoveWorktree removes a git worktree
func RemoveWorktree(issueNumber string) error {
	if !IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	repoName, err := GetRepositoryName()
	if err != nil {
		return err
	}

	worktreeDir := fmt.Sprintf("../%s-%s", repoName, issueNumber)

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", worktreeDir)
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
			current.Branch = strings.TrimPrefix(line, "branch ")
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

// GetWorktreeForIssue finds a worktree for a specific issue number
func GetWorktreeForIssue(issueNumber string) (*WorktreeInfo, error) {
	repoName, err := GetRepositoryName()
	if err != nil {
		return nil, err
	}

	targetPath := fmt.Sprintf("%s-%s", repoName, issueNumber)
	
	worktrees, err := ListWorktrees()
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if strings.Contains(wt.Path, targetPath) {
			return &wt, nil
		}
	}

	return nil, fmt.Errorf("worktree for issue %s not found", issueNumber)
}