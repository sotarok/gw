package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// HasUncommittedChanges checks if there are uncommitted changes in the current worktree
func HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return strings.TrimSpace(string(output)) != "", nil
}

// HasUnpushedCommits checks if there are unpushed commits in the current branch
func HasUnpushedCommits() (bool, error) {
	branch, err := GetCurrentBranch()
	if err != nil {
		return false, err
	}

	// Check if the branch has an upstream
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err := cmd.Run(); err != nil {
		// No upstream branch configured
		return true, nil
	}

	// Check if there are commits ahead of upstream
	cmd = exec.Command("git", "rev-list", "--count", branch+"@{upstream}.."+branch)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check unpushed commits: %w", err)
	}

	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// IsMergedToOrigin checks if the current branch is merged to origin
func IsMergedToOrigin(targetBranch string) (bool, error) {
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return false, err
	}

	// Fetch the latest state from origin
	cmd := exec.Command("git", "fetch", "origin", targetBranch)
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to fetch origin: %w", err)
	}

	// Check if the current branch is merged into origin/targetBranch
	cmd = exec.Command("git", "branch", "-r", "--contains", currentBranch)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check merge status: %w", err)
	}

	branches := strings.Split(string(output), "\n")
	targetRef := fmt.Sprintf("origin/%s", targetBranch)

	for _, branch := range branches {
		if strings.TrimSpace(branch) == targetRef {
			return true, nil
		}
	}

	return false, nil
}
