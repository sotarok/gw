package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/hook"
	"github.com/sotarok/gw/internal/iterm2"
	"github.com/sotarok/gw/internal/spinner"
)

// EndCommand handles the end command logic
type EndCommand struct {
	deps    *Dependencies
	force   bool
	noFetch bool
	config  *config.Config
}

// NewEndCommand creates a new end command handler
func NewEndCommand(deps *Dependencies, force, noFetch bool) *EndCommand {
	// Load config
	cfg, _ := config.Load(config.GetConfigPath())
	return &EndCommand{
		deps:    deps,
		force:   force,
		noFetch: noFetch,
		config:  cfg,
	}
}

// NewEndCommandWithConfig creates a new end command handler with explicit config
func NewEndCommandWithConfig(deps *Dependencies, force, noFetch bool, cfg *config.Config) *EndCommand {
	return &EndCommand{
		deps:    deps,
		force:   force,
		noFetch: noFetch,
		config:  cfg,
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

	// Fetch from remotes if configured
	fetchIfConfigured(c.deps, c.config, c.noFetch)

	// Capture repo name from the current working directory, before any chdir
	// happens, since GetRepositoryName returns the worktree directory name when
	// called from inside a worktree (the hook expects the original repo name).
	var hookRepoName string
	if c.config != nil && c.config.PreEndHook != "" {
		hookRepoName, _ = c.deps.Git.GetRepositoryName()
	}

	// Perform safety checks unless forced. Checks operate via `git -C <path>`
	// so no chdir is needed and the three checks can run in parallel.
	if !c.force {
		sp := spinner.New(fmt.Sprintf("Checking worktree for issue #%s...", issueNumber), c.deps.Stdout)
		sp.Start()
		warnings := c.performSafetyChecks(worktreePath, branchName)
		sp.Stop()

		// If there are warnings, ask for confirmation
		if len(warnings) > 0 {
			fmt.Fprintf(c.deps.Stdout, "\n%s Safety check warnings:\n", coloredWarning())
			for _, warning := range warnings {
				fmt.Fprintf(c.deps.Stdout, "  • %s\n", warning)
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
	if c.config != nil && c.config.PreEndHook != "" {
		c.runPreEndHook(worktreePath, branchName, hookRepoName)
	}

	// Remove the worktree with spinner
	sp := spinner.New(fmt.Sprintf("Removing worktree for issue #%s...", issueNumber), c.deps.Stdout)
	sp.Start()
	var removeErr error
	if isInteractiveMode {
		// Use the actual path when selected from interactive mode
		removeErr = c.deps.Git.RemoveWorktreeByPath(worktreePath)
	} else {
		// Use issue number template when specified directly
		removeErr = c.deps.Git.RemoveWorktree(issueNumber)
	}
	sp.Stop()
	if removeErr != nil {
		return removeErr
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

// runPreEndHook runs pre_end_hook with the worktree as cwd, then restores the
// original directory regardless of hook outcome. Hook failures are warnings.
func (c *EndCommand) runPreEndHook(worktreePath, branchName, repoName string) {
	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(c.deps.Stderr, "%s Could not capture cwd for pre-end hook: %v\n", coloredWarning(), err)
		return
	}
	if err := os.Chdir(worktreePath); err != nil {
		fmt.Fprintf(c.deps.Stderr, "%s Could not enter %s to run pre-end hook: %v\n", coloredWarning(), worktreePath, err)
		return
	}
	defer func() { _ = os.Chdir(originalDir) }()

	absWorktreePath, _ := filepath.Abs(worktreePath)
	hookEnv := hook.Env{
		WorktreePath: absWorktreePath,
		BranchName:   branchName,
		RepoName:     repoName,
		Command:      "end",
	}
	if err := hook.Execute(c.config.PreEndHook, hookEnv, c.deps.Stdout, c.deps.Stderr); err != nil {
		fmt.Fprintf(c.deps.Stderr, "%s Pre-end hook failed: %v\n", coloredWarning(), err)
	}
}

// performSafetyChecks runs the three safety checks (uncommitted changes,
// unpushed commits, merge status) for the worktree at worktreePath in
// parallel. The checks are independent and now use `git -C <path>` instead of
// relying on cwd, so they can safely overlap.
func (c *EndCommand) performSafetyChecks(worktreePath, branchName string) []string {
	type result struct {
		warning string
		errMsg  string // empty when no stderr message to emit
	}

	var wg sync.WaitGroup
	results := make([]result, 3)

	wg.Add(3)
	go func() {
		defer wg.Done()
		hasChanges, err := c.deps.Git.HasUncommittedChanges(worktreePath)
		if err != nil {
			results[0].errMsg = fmt.Sprintf("Could not check for uncommitted changes: %v", err)
		} else if hasChanges {
			results[0].warning = "You have uncommitted changes"
		}
	}()
	go func() {
		defer wg.Done()
		hasUnpushed, err := c.deps.Git.HasUnpushedCommits(worktreePath, branchName)
		if err != nil {
			results[1].errMsg = fmt.Sprintf("Could not check for unpushed commits: %v", err)
		} else if hasUnpushed {
			results[1].warning = "You have unpushed commits"
		}
	}()
	go func() {
		defer wg.Done()
		isMerged, err := c.deps.Git.IsMergedToOrigin(worktreePath, branchName, "main")
		if err != nil {
			results[2].errMsg = fmt.Sprintf("Could not check merge status: %v", err)
		} else if !isMerged {
			results[2].warning = "Branch is not merged to origin/main"
		}
	}()
	wg.Wait()

	var warnings []string
	for _, r := range results {
		if r.errMsg != "" {
			fmt.Fprintf(c.deps.Stderr, "%s Warning: %s\n", coloredWarning(), r.errMsg)
		}
		if r.warning != "" {
			warnings = append(warnings, r.warning)
		}
	}
	return warnings
}
