package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
)

// WorktreeStatus holds the status of a worktree for the clean command
type WorktreeStatus struct {
	Info      *git.WorktreeInfo
	CanRemove bool
	Warnings  []string
}

// CleanCommand handles the clean command logic
type CleanCommand struct {
	deps   *Dependencies
	force  bool
	dryRun bool
	config *config.Config
}

// NewCleanCommand creates a new clean command handler
func NewCleanCommand(deps *Dependencies, force, dryRun bool) *CleanCommand {
	// Load config
	cfg, _ := config.Load(config.GetConfigPath())
	return &CleanCommand{
		deps:   deps,
		force:  force,
		dryRun: dryRun,
		config: cfg,
	}
}

// NewCleanCommandWithConfig creates a new clean command handler with explicit config
func NewCleanCommandWithConfig(deps *Dependencies, force, dryRun bool, cfg *config.Config) *CleanCommand {
	return &CleanCommand{
		deps:   deps,
		force:  force,
		dryRun: dryRun,
		config: cfg,
	}
}

// Execute runs the clean command
func (c *CleanCommand) Execute() error {
	fmt.Fprintf(c.deps.Stdout, "Checking worktrees...\n")

	// Get all worktrees
	worktrees, err := c.deps.Git.ListWorktrees()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Filter out the main worktree and check each worktree
	statuses := make([]*WorktreeStatus, 0, len(worktrees))
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	for _, wt := range worktrees {
		// Skip the main worktree (no branch or main/master branch)
		if wt.Branch == "" || wt.Branch == defaultBaseBranch || wt.Branch == "master" {
			continue
		}

		status := c.checkWorktree(&wt, originalDir)
		statuses = append(statuses, status)
	}

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

// checkWorktree checks if a worktree can be safely removed
func (c *CleanCommand) checkWorktree(info *git.WorktreeInfo, originalDir string) *WorktreeStatus {
	status := &WorktreeStatus{
		Info:      info,
		CanRemove: true,
		Warnings:  []string{},
	}

	// Change to the worktree directory to check status
	if err := os.Chdir(info.Path); err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Could not access directory: %v", err))
		status.CanRemove = false
		return status
	}

	// Check 1: Uncommitted changes
	hasChanges, err := c.deps.Git.HasUncommittedChanges()
	if err != nil {
		// Check if this is a broken worktree (exit status 128 typically means git repository is invalid)
		errMsg := err.Error()
		if strings.Contains(errMsg, "exit status 128") || strings.Contains(errMsg, "not a git repository") {
			status.Warnings = append(status.Warnings, "invalid git repository")
			status.CanRemove = false
			// Don't run further checks if the worktree is fundamentally broken
			_ = os.Chdir(originalDir)
			return status
		}
		status.Warnings = append(status.Warnings, fmt.Sprintf("Could not check uncommitted changes: %v", err))
		status.CanRemove = false
	} else if hasChanges {
		status.Warnings = append(status.Warnings, "uncommitted changes")
		status.CanRemove = false
	}

	// Check 2: Unpushed commits
	hasUnpushed, err := c.deps.Git.HasUnpushedCommits()
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Could not check unpushed commits: %v", err))
		status.CanRemove = false
	} else if hasUnpushed {
		status.Warnings = append(status.Warnings, "unpushed commits")
		status.CanRemove = false
	}

	// Check 3: Merge status with origin/main
	isMerged, err := c.deps.Git.IsMergedToOrigin("main")
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Could not check merge status: %v", err))
		status.CanRemove = false
	} else if !isMerged {
		status.Warnings = append(status.Warnings, "not merged")
		status.CanRemove = false
	}

	// Change back to original directory
	_ = os.Chdir(originalDir)

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

	for _, status := range statuses {
		if !status.CanRemove {
			continue
		}

		dirName := filepath.Base(status.Info.Path)
		fmt.Fprintf(c.deps.Stdout, "Removing %s...\n", dirName)

		// Remove the worktree
		if err := c.deps.Git.RemoveWorktreeByPath(status.Info.Path); err != nil {
			fmt.Fprintf(c.deps.Stderr, "%s Failed to remove %s: %v\n", coloredError(), dirName, err)
			failCount++
			continue
		}

		fmt.Fprintf(c.deps.Stdout, "%s Removed %s\n", coloredSuccess(), dirName)
		successCount++

		// Delete the branch if auto-remove is enabled
		if c.config != nil && c.config.AutoRemoveBranch && status.Info.Branch != "" {
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
