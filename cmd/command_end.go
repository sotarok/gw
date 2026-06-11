package cmd

import (
	"fmt"
	"strings"

	"github.com/sotarok/gw/internal/iterm2"
	"github.com/sotarok/gw/internal/spinner"
)

// EndCommand handles the end command logic
type EndCommand struct {
	deps    *Dependencies
	force   bool
	noFetch bool
}

// NewEndCommand creates a new end command handler
func NewEndCommand(deps *Dependencies, force, noFetch bool) *EndCommand {
	return &EndCommand{
		deps:    deps,
		force:   force,
		noFetch: noFetch,
	}
}

// Execute runs the end command
func (c *EndCommand) Execute(issueNumber string) error {
	var worktreePath string
	var branchName string

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
	} else {
		// Find the worktree for this issue
		wt, err := c.deps.Git.GetWorktreeForIssue(issueNumber)
		if err != nil {
			return err
		}
		worktreePath = wt.Path
		branchName = wt.Branch
	}

	if issueNumber == "" {
		return fmt.Errorf("could not determine issue number")
	}

	// Fetch from remotes if configured
	fetchIfConfigured(c.deps, c.noFetch)

	// Capture repo name from the current working directory, before any chdir
	// happens, since GetRepositoryName returns the worktree directory name when
	// called from inside a worktree (the hook expects the original repo name).
	var hookRepoName string
	if c.deps.Config.PreEndHook != "" {
		hookRepoName, _ = c.deps.Git.GetRepositoryName()
	}

	// Perform safety checks unless forced.
	if !c.force {
		sp := spinner.New(fmt.Sprintf("Checking worktree for issue #%s...", issueNumber), c.deps.Stdout)
		sp.Start()
		warnings := c.performSafetyChecks(worktreePath, branchName)
		sp.Stop()

		// If there are warnings, ask for confirmation
		if len(warnings) > 0 {
			fmt.Fprintf(c.deps.Stderr, "\n%s Safety check warnings:\n", coloredWarning())
			for _, warning := range warnings {
				fmt.Fprintf(c.deps.Stderr, "  • %s\n", warning)
			}

			fmt.Fprintf(c.deps.Stdout, "\nDo you want to continue?")
			confirmed, err := c.deps.UI.ConfirmPrompt(" (y/N): ")
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			if !confirmed {
				fmt.Fprintf(c.deps.Stdout, "Aborted.\n")
				return nil
			}
		}
	}

	// Execute pre-end hook with cwd set to the worktree so the hook can operate
	// on files that are about to disappear (e.g. docker compose).
	if c.deps.Config.PreEndHook != "" {
		runPreEndHook(c.deps, c.deps.Config.PreEndHook, worktreePath, branchName, hookRepoName, "end")
	}

	// Remove the worktree with spinner
	sp := spinner.New(fmt.Sprintf("Removing worktree for issue #%s...", issueNumber), c.deps.Stdout)
	sp.Start()
	// Remove by the resolved worktree path. Whether selected interactively or
	// looked up from the issue number / branch name, worktreePath already points
	// at the actual worktree, so this works regardless of how the input maps to a
	// directory suffix (e.g. "527" vs "527/impl").
	removeErr := c.deps.Git.RemoveWorktreeByPath(worktreePath)
	sp.Stop()
	if removeErr != nil {
		return removeErr
	}

	fmt.Fprintf(c.deps.Stdout, "%s Successfully removed worktree for issue #%s\n", coloredSuccess(), issueNumber)

	// Delete the branch if auto-remove is enabled
	if c.deps.Config.AutoRemoveBranch && branchName != "" {
		fmt.Fprintf(c.deps.Stdout, "Deleting branch %s...\n", branchName)
		if err := c.deps.Git.DeleteBranch(branchName); err != nil {
			// Don't fail the command, just warn
			fmt.Fprintf(c.deps.Stderr, "%s Failed to delete branch %s: %v\n", coloredWarning(), branchName, err)
		} else {
			fmt.Fprintf(c.deps.Stdout, "%s Successfully deleted branch %s\n", coloredSuccess(), branchName)
		}
	}

	// Reset iTerm2 tab if configured
	if iterm2.ShouldUpdateTab(c.deps.Config.UpdateITerm2Tab) {
		_ = iterm2.ResetTabName(c.deps.Stdout)
	}

	return nil
}

// performSafetyChecks runs the three safety checks for the worktree at
// worktreePath in parallel and formats them into end's warning wording.
// Check failures are reported on stderr; only tripped checks become warnings.
func (c *EndCommand) performSafetyChecks(worktreePath, branchName string) []string {
	res := runSafetyChecks(c.deps.Git, worktreePath, branchName, defaultBaseBranch)

	checks := []struct {
		check    safetyCheck
		warning  string
		errLabel string
	}{
		{res.Uncommitted, "You have uncommitted changes", "Could not check for uncommitted changes"},
		{res.Unpushed, "You have unpushed commits", "Could not check for unpushed commits"},
		{res.Merged, "Branch is not merged to main", "Could not check merge status"},
	}

	var warnings []string
	for _, c2 := range checks {
		if c2.check.Err != nil {
			fmt.Fprintf(c.deps.Stderr, "%s Warning: %s: %v\n", coloredWarning(), c2.errLabel, c2.check.Err)
			continue
		}
		if c2.check.Tripped {
			warnings = append(warnings, c2.warning)
		}
	}
	return warnings
}
