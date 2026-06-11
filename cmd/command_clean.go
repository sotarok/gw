package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/spinner"
)

// cleanCheckConcurrency caps the number of worktrees whose safety checks may
// run in parallel during `gw clean`. Each check forks three `git` subprocesses,
// so the effective fd ceiling is ~3× this value.
const cleanCheckConcurrency = 8

// WorktreeStatus holds the status of a worktree for the clean command
type WorktreeStatus struct {
	Info      *git.WorktreeInfo
	CanRemove bool
	Warnings  []string
}

// CleanCommand handles the clean command logic
type CleanCommand struct {
	deps    *Dependencies
	force   bool
	dryRun  bool
	noFetch bool
}

// NewCleanCommand creates a new clean command handler
func NewCleanCommand(deps *Dependencies, force, dryRun, noFetch bool) *CleanCommand {
	return &CleanCommand{
		deps:    deps,
		force:   force,
		dryRun:  dryRun,
		noFetch: noFetch,
	}
}

// Execute runs the clean command
func (c *CleanCommand) Execute() error {
	// Fetch from remotes if configured
	fetchIfConfigured(c.deps, c.noFetch)

	// Get all worktrees
	worktrees, err := c.deps.Git.ListWorktrees()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	candidates := make([]git.WorktreeInfo, 0, len(worktrees))
	for _, wt := range worktrees {
		if wt.Branch == "" || wt.Branch == defaultBaseBranch || wt.Branch == "master" {
			continue
		}
		candidates = append(candidates, wt)
	}

	statuses := make([]*WorktreeStatus, len(candidates))
	sp := spinner.New("Checking worktrees...", c.deps.Stdout)
	sp.Start()
	// Bound concurrency: each check forks three `git` subprocesses, so
	// unbounded fan-out over a large worktree count could exhaust file
	// descriptors and saturate the disk.
	sem := make(chan struct{}, cleanCheckConcurrency)
	var wg sync.WaitGroup
	wg.Add(len(candidates))
	for i := range candidates {
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			statuses[idx] = c.checkWorktree(&candidates[idx])
		}(i)
	}
	wg.Wait()
	sp.Stop()

	// Display results
	c.displayResults(statuses)

	// Count removable worktrees
	removableCount := 0
	for _, status := range statuses {
		if status.CanRemove {
			removableCount++
		}
	}

	if removableCount == 0 {
		fmt.Fprintf(c.deps.Stdout, "\nNo worktrees to remove.\n")
		return nil
	}

	// If dry-run, stop here
	if c.dryRun {
		fmt.Fprintf(c.deps.Stdout, "\nDry-run mode: no changes made.\n")
		return nil
	}

	// Ask for confirmation unless forced
	if !c.force {
		var prompt string
		if removableCount == 1 {
			prompt = "\nRemove 1 worktree? (y/N): "
		} else {
			prompt = fmt.Sprintf("\nRemove %d worktrees? (y/N): ", removableCount)
		}

		confirmed, err := c.deps.UI.ConfirmPrompt(prompt)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if !confirmed {
			fmt.Fprintf(c.deps.Stdout, "Aborted.\n")
			return nil
		}
	}

	// Remove worktrees
	return c.removeWorktrees(statuses)
}

// checkWorktree checks if a worktree can be safely removed.
func (c *CleanCommand) checkWorktree(info *git.WorktreeInfo) *WorktreeStatus {
	status := &WorktreeStatus{
		Info:      info,
		CanRemove: true,
		Warnings:  []string{},
	}

	res := runSafetyChecks(c.deps.Git, info.Path, info.Branch, defaultBaseBranch)

	// A broken or missing worktree (git exit 128) — surface a single clear
	// reason instead of three meaningless ones.
	if res.InvalidRepo {
		status.Warnings = append(status.Warnings, "invalid git repository")
		status.CanRemove = false
		return status
	}

	if res.Uncommitted.Err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Could not check uncommitted changes: %v", res.Uncommitted.Err))
		status.CanRemove = false
	} else if res.Uncommitted.Tripped {
		status.Warnings = append(status.Warnings, "uncommitted changes")
		status.CanRemove = false
	}

	if res.Unpushed.Err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Could not check unpushed commits: %v", res.Unpushed.Err))
		status.CanRemove = false
	} else if res.Unpushed.Tripped {
		status.Warnings = append(status.Warnings, "unpushed commits")
		status.CanRemove = false
	}

	if res.Merged.Err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Could not check merge status: %v", res.Merged.Err))
		status.CanRemove = false
	} else if res.Merged.Tripped {
		status.Warnings = append(status.Warnings, "not merged")
		status.CanRemove = false
	}

	return status
}

// displayResults displays the status of all worktrees
func (c *CleanCommand) displayResults(statuses []*WorktreeStatus) {
	removable := []*WorktreeStatus{}
	nonRemovable := []*WorktreeStatus{}

	for _, status := range statuses {
		if status.CanRemove {
			removable = append(removable, status)
		} else {
			nonRemovable = append(nonRemovable, status)
		}
	}

	// Display removable worktrees
	if len(removable) > 0 {
		fmt.Fprintf(c.deps.Stdout, "\n%s Removable (%d)\n", coloredSuccess(), len(removable))
		for _, status := range removable {
			dirName := filepath.Base(status.Info.Path)
			fmt.Fprintf(c.deps.Stdout, "  %s (%s)\n", dirName, status.Info.Branch)
		}
	}

	// Display non-removable worktrees
	if len(nonRemovable) > 0 {
		fmt.Fprintf(c.deps.Stdout, "\n%s Non-removable (%d)\n", coloredError(), len(nonRemovable))
		for i, status := range nonRemovable {
			if i > 0 {
				fmt.Fprintf(c.deps.Stdout, "\n")
			}
			dirName := filepath.Base(status.Info.Path)
			fmt.Fprintf(c.deps.Stdout, "  %s (%s)\n", dirName, status.Info.Branch)
			if len(status.Warnings) > 0 {
				reasons := strings.Join(status.Warnings, ", ")
				fmt.Fprintf(c.deps.Stdout, "    %s %s\n", coloredArrow(), reasons)
			}
		}
	}
}

// removeWorktrees removes all removable worktrees
func (c *CleanCommand) removeWorktrees(statuses []*WorktreeStatus) error {
	successCount := 0
	failCount := 0

	var repoName string
	if c.deps.Config.PreEndHook != "" {
		repoName, _ = c.deps.Git.GetRepositoryName()
	}

	for _, status := range statuses {
		if !status.CanRemove {
			continue
		}

		dirName := filepath.Base(status.Info.Path)

		// Run pre-end hook from inside the worktree before it gets removed.
		if c.deps.Config.PreEndHook != "" {
			runPreEndHook(c.deps, c.deps.Config.PreEndHook, status.Info.Path, status.Info.Branch, repoName, "clean")
		}

		// Remove the worktree with spinner
		sp := spinner.New(fmt.Sprintf("Removing %s...", dirName), c.deps.Stdout)
		sp.Start()
		removeErr := c.deps.Git.RemoveWorktreeByPath(status.Info.Path)
		sp.Stop()
		if removeErr != nil {
			fmt.Fprintf(c.deps.Stderr, "%s Failed to remove %s: %v\n", coloredError(), dirName, removeErr)
			failCount++
			continue
		}

		fmt.Fprintf(c.deps.Stdout, "%s Removed %s\n", coloredSuccess(), dirName)
		successCount++

		// Delete the branch if auto-remove is enabled
		if c.deps.Config.AutoRemoveBranch && status.Info.Branch != "" {
			fmt.Fprintf(c.deps.Stdout, "Deleting branch %s...\n", status.Info.Branch)
			if err := c.deps.Git.DeleteBranch(status.Info.Branch); err != nil {
				// Don't fail the command, just warn
				fmt.Fprintf(c.deps.Stderr, "%s Failed to delete branch %s: %v\n", coloredWarning(), status.Info.Branch, err)
			} else {
				fmt.Fprintf(c.deps.Stdout, "%s Deleted branch %s\n", coloredSuccess(), status.Info.Branch)
			}
		}
	}

	// Summary
	fmt.Fprintf(c.deps.Stdout, "\n")
	if successCount > 0 {
		fmt.Fprintf(c.deps.Stdout, "%s Successfully removed %d worktree(s)\n", coloredSuccess(), successCount)
	}
	if failCount > 0 {
		fmt.Fprintf(c.deps.Stderr, "%s Failed to remove %d worktree(s)\n", coloredError(), failCount)
		return fmt.Errorf("failed to remove %d worktree(s)", failCount)
	}

	return nil
}
