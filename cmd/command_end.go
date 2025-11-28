package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/iterm2"
)

// EndCommand handles the end command logic
type EndCommand struct {
	deps   *Dependencies
	force  bool
	config *config.Config
}

// NewEndCommand creates a new end command handler
func NewEndCommand(deps *Dependencies, force bool) *EndCommand {
	// Load config
	cfg, _ := config.Load(config.GetConfigPath())
	return &EndCommand{
		deps:   deps,
		force:  force,
		config: cfg,
	}
}

// NewEndCommandWithConfig creates a new end command handler with explicit config
func NewEndCommandWithConfig(deps *Dependencies, force bool, cfg *config.Config) *EndCommand {
	return &EndCommand{
		deps:   deps,
		force:  force,
		config: cfg,
	}
}

// Execute runs the end command
func (c *EndCommand) Execute(issueNumber string) error {
	var worktreePath string
	var branchName string
	var isInteractiveMode bool

	if issueNumber == "" {
		// Interactive mode
		fmt.Fprintf(c.deps.Stdout, "No issue number provided, entering interactive mode...\n")

		selected, err := c.deps.UI.SelectWorktree()
		if err != nil {
			return err
		}

		// Extract issue number from the path or branch
		parts := strings.Split(selected.Branch, "/")
		if len(parts) > 0 {
			issueNumber = parts[0]
		}
		worktreePath = selected.Path
		branchName = selected.Branch
		isInteractiveMode = true
	} else {
		// Find the worktree for this issue
		wt, err := c.deps.Git.GetWorktreeForIssue(issueNumber)
		if err != nil {
			return err
		}
		worktreePath = wt.Path
		branchName = wt.Branch
		isInteractiveMode = false
	}

	if issueNumber == "" {
		return fmt.Errorf("could not determine issue number")
	}

	fmt.Fprintf(c.deps.Stdout, "Checking worktree for issue #%s...\n", issueNumber)

	// Change to the worktree directory to check status
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(worktreePath); err != nil {
		return fmt.Errorf("failed to change to worktree directory: %w", err)
	}

	// Perform safety checks unless forced
	if !c.force {
		warnings := c.performSafetyChecks()

		// If there are warnings, ask for confirmation
		if len(warnings) > 0 {
			fmt.Fprintf(c.deps.Stdout, "\n%s Safety check warnings:\n", coloredWarning())
			for _, warning := range warnings {
				fmt.Fprintf(c.deps.Stdout, "  • %s\n", warning)
			}

			fmt.Fprintf(c.deps.Stdout, "\nDo you want to continue?")
			confirmed, err := c.deps.UI.ConfirmPrompt(" (y/N): ")
			if err != nil {
				_ = os.Chdir(originalDir)
				return fmt.Errorf("failed to read response: %w", err)
			}

			if !confirmed {
				fmt.Fprintf(c.deps.Stdout, "Aborted.\n")
				_ = os.Chdir(originalDir)
				return nil
			}
		}
	}

	// Change back to original directory before removing
	if err := os.Chdir(originalDir); err != nil {
		return fmt.Errorf("failed to change back to original directory: %w", err)
	}

	fmt.Fprintf(c.deps.Stdout, "Removing worktree for issue #%s...\n", issueNumber)

	// Remove the worktree
	if isInteractiveMode {
		// Use the actual path when selected from interactive mode
		if err := c.deps.Git.RemoveWorktreeByPath(worktreePath); err != nil {
			return err
		}
	} else {
		// Use issue number template when specified directly
		if err := c.deps.Git.RemoveWorktree(issueNumber); err != nil {
			return err
		}
	}

	fmt.Fprintf(c.deps.Stdout, "%s Successfully removed worktree for issue #%s\n", coloredSuccess(), issueNumber)

	// Delete the branch if auto-remove is enabled
	if c.config != nil && c.config.AutoRemoveBranch && branchName != "" {
		fmt.Fprintf(c.deps.Stdout, "Deleting branch %s...\n", branchName)
		if err := c.deps.Git.DeleteBranch(branchName); err != nil {
			// Don't fail the command, just warn
			fmt.Fprintf(c.deps.Stderr, "%s Failed to delete branch %s: %v\n", coloredWarning(), branchName, err)
		} else {
			fmt.Fprintf(c.deps.Stdout, "%s Successfully deleted branch %s\n", coloredSuccess(), branchName)
		}
	}

	// Reset iTerm2 tab if configured
	if c.config != nil && iterm2.ShouldUpdateTab(c.config.UpdateITerm2Tab) {
		_ = iterm2.ResetTabName(c.deps.Stdout)
	}

	return nil
}

func (c *EndCommand) performSafetyChecks() []string {
	var warnings []string

	// Check for uncommitted changes
	hasChanges, err := c.deps.Git.HasUncommittedChanges()
	if err != nil {
		fmt.Fprintf(c.deps.Stderr, "%s Warning: Could not check for uncommitted changes: %v\n", coloredWarning(), err)
	} else if hasChanges {
		warnings = append(warnings, "You have uncommitted changes")
	}

	// Check for unpushed commits
	hasUnpushed, err := c.deps.Git.HasUnpushedCommits()
	if err != nil {
		fmt.Fprintf(c.deps.Stderr, "%s Warning: Could not check for unpushed commits: %v\n", coloredWarning(), err)
	} else if hasUnpushed {
		warnings = append(warnings, "You have unpushed commits")
	}

	// Check if merged to origin
	isMerged, err := c.deps.Git.IsMergedToOrigin("main")
	if err != nil {
		fmt.Fprintf(c.deps.Stderr, "%s Warning: Could not check merge status: %v\n", coloredWarning(), err)
	} else if !isMerged {
		warnings = append(warnings, "Branch is not merged to origin/main")
	}

	return warnings
}
