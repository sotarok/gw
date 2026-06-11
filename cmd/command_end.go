package cmd

import (
	"fmt"
	"strings"

	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/iterm2"
	"github.com/sotarok/gw/internal/spinner"
)

// endGit is the subset of git operations EndCommand actually uses.
type endGit interface {
	git.RepositoryReader // GetRepositoryName, FetchAll
	git.WorktreeManager  // GetWorktreeForIssue, RemoveWorktreeByPath
	git.BranchManager    // DeleteBranch
	git.StatusChecker
}

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

// git returns the command's git dependency narrowed to the operations it uses.
func (c *EndCommand) git() endGit { return c.deps.Git }

// Execute runs the end command
func (c *EndCommand) Execute(issueNumber string) error {
	issueNumber, worktreePath, branchName, err := c.resolveWorktree(issueNumber)
	if err != nil {
		return err
	}

	// Fetch from remotes if configured
	fetchIfConfigured(c.deps, c.noFetch)

	// Capture repo name from the current working directory, before any chdir
	// happens, since GetRepositoryName returns the worktree directory name when
	// called from inside a worktree (the hook expects the original repo name).
	var hookRepoName string
	if c.deps.Config.PreEndHook != "" {
		hookRepoName, _ = c.git().GetRepositoryName()
	}

	proceed, err := c.confirmRemoval(issueNumber, worktreePath, branchName)
	if err != nil {
		return err
	}
	if !proceed {
		return nil
	}

	return c.remove(issueNumber, worktreePath, branchName, hookRepoName)
}

// resolveWorktree determines the worktree to remove, either via interactive
// selection (when issueNumber is empty) or by looking it up from the issue
// number. It returns the resolved issue number, worktree path, and branch name.
func (c *EndCommand) resolveWorktree(issueNumber string) (resolvedIssue, worktreePath, branchName string, err error) {
	if issueNumber == "" {
		// Interactive mode
		fmt.Fprintf(c.deps.Stdout, "No issue number provided, entering interactive mode...\n")

		selected, selErr := c.deps.UI.SelectWorktree()
		if selErr != nil {
			return "", "", "", selErr
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
		wt, lookupErr := c.git().GetWorktreeForIssue(issueNumber)
		if lookupErr != nil {
			return "", "", "", lookupErr
		}
		worktreePath = wt.Path
		branchName = wt.Branch
	}

	if issueNumber == "" {
		return "", "", "", fmt.Errorf("could not determine issue number")
	}

	return issueNumber, worktreePath, branchName, nil
}

// confirmRemoval runs the safety checks (unless forced) and, when they raise
// warnings, prompts the user to continue. It returns whether the removal should
// proceed.
func (c *EndCommand) confirmRemoval(issueNumber, worktreePath, branchName string) (bool, error) {
	if c.force {
		return true, nil
	}

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
			return false, fmt.Errorf("failed to read response: %w", err)
		}

		if !confirmed {
			fmt.Fprintf(c.deps.Stdout, "Aborted.\n")
			return false, nil
		}
	}

	return true, nil
}

// remove runs the pre-end hook, removes the worktree, optionally deletes the
// branch, and resets the iTerm2 tab.
func (c *EndCommand) remove(issueNumber, worktreePath, branchName, hookRepoName string) error {
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
	removeErr := c.git().RemoveWorktreeByPath(worktreePath)
	sp.Stop()
	if removeErr != nil {
		return removeErr
	}

	fmt.Fprintf(c.deps.Stdout, "%s Successfully removed worktree for issue #%s\n", coloredSuccess(), issueNumber)

	// Delete the branch if auto-remove is enabled
	if c.deps.Config.AutoRemoveBranch && branchName != "" {
		fmt.Fprintf(c.deps.Stdout, "Deleting branch %s...\n", branchName)
		if err := c.git().DeleteBranch(branchName); err != nil {
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
	res := runSafetyChecks(c.git(), worktreePath, branchName, defaultBaseBranch)

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
