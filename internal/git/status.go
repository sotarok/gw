package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// HasUncommittedChanges checks if the worktree at worktreePath has any
// uncommitted changes. The function uses `git -C` instead of relying on the
// process cwd so callers can run multiple checks against different worktrees
// concurrently without serializing on os.Chdir.
func HasUncommittedChanges(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return strings.TrimSpace(string(output)) != "", nil
}

// HasUnpushedCommits checks whether currentBranch in the worktree at
// worktreePath has commits that haven't been pushed to its upstream. When the
// branch has no upstream configured, the function falls back to checking
// whether it is already merged to origin/main (covering the common
// "PR merged then remote branch auto-deleted" case).
func HasUnpushedCommits(worktreePath, currentBranch string) (bool, error) {
	// Check if the branch has an upstream
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", currentBranch+"@{upstream}")
	if err := cmd.Run(); err != nil {
		// No upstream branch configured.
		// Check if the branch is already merged to main/master. This handles
		// the case where the branch was merged and the remote was deleted.
		merged, mergeErr := IsMergedToOrigin(worktreePath, currentBranch, "main")
		if mergeErr == nil && merged {
			// Branch is merged, so no unpushed commits
			return false, nil
		}

		// If we can't determine merge status or branch is not merged,
		// assume there are unpushed commits for safety
		return true, nil
	}

	// Check if there are commits ahead of upstream
	cmd = exec.Command("git", "-C", worktreePath, "rev-list", "--count", currentBranch+"@{upstream}.."+currentBranch)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check unpushed commits: %w", err)
	}

	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// IsMergedToOrigin checks if currentBranch in the worktree at worktreePath is
// merged into origin/<targetBranch>.
//
// This function does NOT fetch from origin — callers are expected to have
// already updated remote-tracking refs (e.g. via `fetchIfConfigured`) when a
// fresh view is required. Avoiding the internal fetch keeps `gw clean` from
// hitting the network once per worktree.
func IsMergedToOrigin(worktreePath, currentBranch, targetBranch string) (bool, error) {
	// Check if the current branch is merged into origin/targetBranch
	cmd := exec.Command("git", "-C", worktreePath, "branch", "-r", "--contains", currentBranch)
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
