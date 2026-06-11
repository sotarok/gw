package git

import (
	"fmt"
	"strings"
)

// HasUncommittedChanges checks if the worktree at worktreePath has any
// uncommitted changes.
func (c *Client) HasUncommittedChanges(worktreePath string) (bool, error) {
	out, err := c.r.run(worktreePath, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return out != "", nil
}

// HasUnpushedCommits checks whether currentBranch in the worktree at
// worktreePath has commits that haven't been pushed to its upstream. When the
// branch has no upstream configured, the function falls back to checking
// whether it is already merged into the base branch (local main or
// origin/main), covering both the "PR merged then remote branch auto-deleted"
// case and the "merged into local main before pushing" case.
func (c *Client) HasUnpushedCommits(worktreePath, currentBranch string) (bool, error) {
	// Check if the branch has an upstream
	if _, err := c.r.run(worktreePath, "rev-parse", "--abbrev-ref", currentBranch+"@{upstream}"); err != nil {
		// No upstream branch configured.
		// Check if the branch is already merged to main/master. This handles
		// the case where the branch was merged and the remote was deleted.
		merged, mergeErr := c.IsMergedToBaseBranch(worktreePath, currentBranch, "main")
		if mergeErr == nil && merged {
			// Branch is merged, so no unpushed commits
			return false, nil
		}

		// If we can't determine merge status or branch is not merged,
		// assume there are unpushed commits for safety
		return true, nil
	}

	// Check if there are commits ahead of upstream
	out, err := c.r.run(worktreePath, "rev-list", "--count", currentBranch+"@{upstream}.."+currentBranch)
	if err != nil {
		return false, fmt.Errorf("failed to check unpushed commits: %w", err)
	}

	return out != "0", nil
}

// IsMergedToBaseBranch reports whether currentBranch in the worktree at
// worktreePath is already merged into the base branch, considering both the
// local <targetBranch> and origin/<targetBranch>. A branch merged into the
// local base branch is treated as merged even when that merge hasn't been
// pushed yet, since the work is preserved in the local base branch's history
// and is therefore safe to remove. Callers must refresh remote-tracking refs
// themselves (e.g. via fetchIfConfigured) — this function does not fetch.
func (c *Client) IsMergedToBaseBranch(worktreePath, currentBranch, targetBranch string) (bool, error) {
	// Merged into the remote base branch (origin/<targetBranch>).
	remoteMerged, err := c.branchListContains(worktreePath, currentBranch, true, "origin/"+targetBranch)
	if err != nil {
		return false, err
	}
	if remoteMerged {
		return true, nil
	}

	// Merged into the local base branch. This covers the common case of
	// merging the branch into local main before pushing main.
	return c.branchListContains(worktreePath, currentBranch, false, targetBranch)
}

// branchListContains reports whether wantRef appears in the output of
// `git branch [-r] --contains currentBranch`, i.e. whether wantRef's history
// contains the tip of currentBranch. When remote is true it inspects
// remote-tracking branches, otherwise local branches.
func (c *Client) branchListContains(worktreePath, currentBranch string, remote bool, wantRef string) (bool, error) {
	args := []string{"branch"}
	if remote {
		args = append(args, "-r")
	}
	args = append(args, "--contains", currentBranch)

	output, err := c.r.run(worktreePath, args...)
	if err != nil {
		return false, fmt.Errorf("failed to check merge status: %w", err)
	}

	for _, line := range strings.Split(output, "\n") {
		// Strip the leading markers git prints: "* " for the current branch and
		// "+ " for a branch checked out in another worktree (e.g. local main).
		name := strings.TrimSpace(line)
		name = strings.TrimPrefix(name, "* ")
		name = strings.TrimPrefix(name, "+ ")
		if strings.TrimSpace(name) == wantRef {
			return true, nil
		}
	}

	return false, nil
}
