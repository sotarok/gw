package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/iterm2"
	"github.com/sotarok/gw/internal/ui"
)

// Symbol constants for consistent output formatting across commands
const (
	symbolSuccess = "✓"
	symbolError   = "✗"
	symbolWarning = "⚠"
	symbolArrow   = "→"
)

// Dependencies holds all the dependencies for commands
type Dependencies struct {
	Git    git.Interface
	UI     ui.Interface
	Detect detect.Interface
	Stdout io.Writer
	Stderr io.Writer
}

// DefaultDependencies returns the default dependencies
func DefaultDependencies() *Dependencies {
	return &Dependencies{
		Git:    git.NewDefaultClient(),
		UI:     ui.NewDefaultUI(),
		Detect: detect.NewDefaultDetector(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// StartCommand handles the start command logic
type StartCommand struct {
	deps     *Dependencies
	copyEnvs bool
	config   *config.Config
}

// NewStartCommand creates a new start command handler
func NewStartCommand(deps *Dependencies, copyEnvs bool) *StartCommand {
	// Load config
	cfg, _ := config.Load(config.GetConfigPath())
	return &StartCommand{
		deps:     deps,
		copyEnvs: copyEnvs,
		config:   cfg,
	}
}

// NewStartCommandWithConfig creates a new start command handler with explicit config
func NewStartCommandWithConfig(deps *Dependencies, copyEnvs bool, cfg *config.Config) *StartCommand {
	return &StartCommand{
		deps:     deps,
		copyEnvs: copyEnvs,
		config:   cfg,
	}
}

// Execute runs the start command
func (c *StartCommand) Execute(issueNumber, baseBranch string) error {
	// Check if we're in a git repository
	if !c.deps.Git.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Check if worktree already exists
	if wt, _ := c.deps.Git.GetWorktreeForIssue(issueNumber); wt != nil {
		return fmt.Errorf("worktree for issue %s already exists at %s", issueNumber, wt.Path)
	}

	// Get repository name for iTerm2 tab
	repoName, _ := c.deps.Git.GetRepositoryName()

	// Update iTerm2 tab if configured
	if c.config != nil && iterm2.ShouldUpdateTab(c.config.UpdateITerm2Tab) {
		_ = iterm2.UpdateTabName(c.deps.Stdout, repoName, issueNumber)
	}

	// Get the original repository root before creating worktree
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Print message if stdout is available
	if c.deps.Stdout != nil {
		fmt.Fprintf(c.deps.Stdout, "Creating worktree for issue #%s based on %s...\n", issueNumber, baseBranch)
	}

	// Create the worktree
	worktreePath, err := c.deps.Git.CreateWorktree(issueNumber, baseBranch)
	if err != nil {
		return err
	}

	if c.deps.Stdout != nil {
		fmt.Fprintf(c.deps.Stdout, "✓ Created worktree at %s\n", worktreePath)
	}

	// Change to the new worktree directory for setup operations
	// Note: This only affects the current process, not the parent shell
	if c.config != nil && c.config.AutoCD {
		if err := os.Chdir(worktreePath); err != nil {
			// Don't fail the command, just log the error
			if c.deps.Stderr != nil {
				fmt.Fprintf(c.deps.Stderr, "⚠ Could not change to worktree directory: %v\n", err)
			}
		}
	}

	// Handle environment files
	if err := c.handleEnvFiles(originalDir, worktreePath); err != nil {
		// Don't fail the command, just warn
		if c.deps.Stderr != nil {
			fmt.Fprintf(c.deps.Stderr, "⚠ Failed to handle env files: %v\n", err)
		}
	}

	// Run setup if a package manager is detected
	if err := c.deps.Detect.RunSetup(worktreePath); err != nil {
		// Don't fail if setup fails, just warn
		if c.deps.Stderr != nil {
			fmt.Fprintf(c.deps.Stderr, "⚠ Setup failed: %v\n", err)
		}
	}

	if c.deps.Stdout != nil {
		fmt.Fprintf(c.deps.Stdout, "\n✨ Worktree ready at:\n   %s\n", worktreePath)
		if c.config != nil && c.config.AutoCD {
			fmt.Fprintf(c.deps.Stdout, "\n💡 Shell integration will change to this directory after the command completes.\n")
		}
	}
	return nil
}

func (c *StartCommand) handleEnvFiles(originalDir, worktreePath string) error {
	return handleEnvFiles(c.deps, c.config, c.copyEnvs, originalDir, worktreePath)
}

// handleEnvFiles is a common function for handling environment files
// Priority order:
// 1. If --copy-envs flag is set, always copy
// 2. If config.CopyEnvs is set (true/false), use that value (unless flag overrides)
// 3. If neither is set, prompt user (interactive mode)
func handleEnvFiles(deps *Dependencies, cfg *config.Config, copyEnvsFlag bool, originalDir, worktreePath string) error {
	envFiles, err := deps.Git.FindUntrackedEnvFiles(originalDir)
	if err != nil {
		return fmt.Errorf("failed to find env files: %w", err)
	}

	if len(envFiles) == 0 {
		return nil
	}

	// Prepare file list
	filePaths := make([]string, len(envFiles))
	for i, f := range envFiles {
		filePaths[i] = f.Path
	}

	// Determine whether to copy based on priority
	var shouldCopy bool
	var needsPrompt bool

	if copyEnvsFlag {
		// Priority 1: Flag is set, always copy
		shouldCopy = true
		needsPrompt = false
	} else if cfg != nil && cfg.CopyEnvs != nil {
		// Priority 2: Config is set, use config value
		shouldCopy = *cfg.CopyEnvs
		needsPrompt = false
	} else {
		// Priority 3: Neither flag nor config is set, prompt user
		needsPrompt = true
	}

	if needsPrompt {
		fmt.Fprintf(deps.Stdout, "\nFound %d untracked environment file(s):\n", len(envFiles))
		deps.UI.ShowEnvFilesList(filePaths)

		fmt.Fprintf(deps.Stdout, "\nCopy them to the new worktree?")
		confirmed, err := deps.UI.ConfirmPrompt("")
		if err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		shouldCopy = confirmed
	} else if shouldCopy {
		// When copy decision is made without prompting, show the files being copied
		fmt.Fprintf(deps.Stdout, "\nCopying environment files:\n")
		deps.UI.ShowEnvFilesList(filePaths)
	}

	if shouldCopy {
		// Copy files
		if err := deps.Git.CopyEnvFiles(envFiles, originalDir, worktreePath); err != nil {
			return fmt.Errorf("failed to copy env files: %w", err)
		}
		fmt.Fprintf(deps.Stdout, "✓ Environment files copied successfully\n")
	}

	return nil
}

// CheckoutCommand handles the checkout command logic
type CheckoutCommand struct {
	deps     *Dependencies
	copyEnvs bool
	config   *config.Config
}

// NewCheckoutCommand creates a new checkout command handler
func NewCheckoutCommand(deps *Dependencies, copyEnvs bool) *CheckoutCommand {
	// Load config
	cfg, _ := config.Load(config.GetConfigPath())
	return &CheckoutCommand{
		deps:     deps,
		copyEnvs: copyEnvs,
		config:   cfg,
	}
}

// NewCheckoutCommandWithConfig creates a new checkout command handler with explicit config
func NewCheckoutCommandWithConfig(deps *Dependencies, copyEnvs bool, cfg *config.Config) *CheckoutCommand {
	return &CheckoutCommand{
		deps:     deps,
		copyEnvs: copyEnvs,
		config:   cfg,
	}
}

// Execute runs the checkout command
func (c *CheckoutCommand) Execute(branch string) error {
	// If no branch specified, use interactive mode
	if branch == "" {
		selectedBranch, err := c.selectBranch()
		if err != nil {
			return err
		}
		branch = selectedBranch
	}

	// Get repository name
	repoName, err := c.deps.Git.GetRepositoryName()
	if err != nil {
		return fmt.Errorf("failed to get repository name: %w", err)
	}

	// Update iTerm2 tab if configured
	if c.config != nil && iterm2.ShouldUpdateTab(c.config.UpdateITerm2Tab) {
		identifier := iterm2.GetIdentifierFromBranch(branch)
		_ = iterm2.UpdateTabName(c.deps.Stdout, repoName, identifier)
	}

	// Extract branch name without remote prefix
	branchName := branch
	if strings.HasPrefix(branch, "origin/") {
		branchName = strings.TrimPrefix(branch, "origin/")
	}

	// Create worktree directory name
	sanitizedBranchName := c.deps.Git.SanitizeBranchNameForDirectory(branchName)
	worktreeName := fmt.Sprintf("%s-%s", repoName, sanitizedBranchName)
	worktreePath := filepath.Join("..", worktreeName)

	// Check if branch exists
	exists, err := c.deps.Git.BranchExists(branch)
	if err != nil {
		return fmt.Errorf("failed to check branch existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("branch '%s' does not exist in the repository\nUse 'git branch -a' to see all available branches", branch)
	}

	// Get the original repository root before creating worktree
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create worktree
	fmt.Fprintf(c.deps.Stdout, "Creating worktree for branch '%s'...\n", branch)
	if err := c.deps.Git.CreateWorktreeFromBranch(worktreePath, branch, branchName); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Change to the new worktree directory if auto-cd is enabled
	absolutePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Change to the new worktree directory for setup operations
	// Note: This only affects the current process, not the parent shell
	if c.config != nil && c.config.AutoCD {
		if err := os.Chdir(worktreePath); err != nil {
			// Don't fail the command, just log the error
			if c.deps.Stderr != nil {
				fmt.Fprintf(c.deps.Stderr, "⚠ Could not change to worktree directory: %v\n", err)
			}
		}
	}

	// Handle environment files
	if err := c.handleEnvFiles(originalDir, absolutePath); err != nil {
		// Don't fail the command, just warn
		fmt.Fprintf(c.deps.Stderr, "⚠ Failed to handle env files: %v\n", err)
	}

	// Run package manager setup
	if err := c.deps.Detect.RunSetup(absolutePath); err != nil {
		// Don't fail if setup fails, just warn
		fmt.Fprintf(c.deps.Stderr, "⚠ Setup failed: %v\n", err)
	}

	// Show completion message
	if c.deps.Stdout != nil {
		fmt.Fprintf(c.deps.Stdout, "\n✨ Worktree ready at:\n   %s\n", absolutePath)
		if c.config != nil && c.config.AutoCD {
			fmt.Fprintf(c.deps.Stdout, "\n💡 Shell integration will change to this directory after the command completes.\n")
		}
	}

	return nil
}

func (c *CheckoutCommand) handleEnvFiles(originalDir, worktreePath string) error {
	return handleEnvFiles(c.deps, c.config, c.copyEnvs, originalDir, worktreePath)
}

func (c *CheckoutCommand) selectBranch() (string, error) {
	// Get all branches (local and remote)
	branches, err := c.deps.Git.ListAllBranches()
	if err != nil {
		return "", fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		return "", fmt.Errorf("no branches found")
	}

	// Filter out current branch and main/master
	currentBranch, err := c.deps.Git.GetCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	const (
		mainBranch   = "main"
		masterBranch = "master"
		originMain   = "origin/main"
		originMaster = "origin/master"
	)

	var filteredBranches []string
	for _, branch := range branches {
		// Skip current branch and main branches
		if branch != currentBranch && branch != mainBranch && branch != masterBranch &&
			branch != originMain && branch != originMaster {
			filteredBranches = append(filteredBranches, branch)
		}
	}

	if len(filteredBranches) == 0 {
		return "", fmt.Errorf("no branches available for checkout")
	}

	// Create items for selector
	items := make([]ui.SelectorItem, len(filteredBranches))
	for i, branch := range filteredBranches {
		items[i] = ui.SelectorItem{
			ID:   branch,
			Name: branch,
		}
	}

	// Show selector
	selected, err := c.deps.UI.ShowSelector("Select a branch to checkout:", items)
	if err != nil {
		return "", err
	}

	return selected.ID, nil
}

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
			fmt.Fprintf(c.deps.Stdout, "\n⚠ Safety check warnings:\n")
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

	fmt.Fprintf(c.deps.Stdout, "✓ Successfully removed worktree for issue #%s\n", issueNumber)

	// Delete the branch if auto-remove is enabled
	if c.config != nil && c.config.AutoRemoveBranch && branchName != "" {
		fmt.Fprintf(c.deps.Stdout, "Deleting branch %s...\n", branchName)
		if err := c.deps.Git.DeleteBranch(branchName); err != nil {
			// Don't fail the command, just warn
			fmt.Fprintf(c.deps.Stderr, "⚠ Failed to delete branch %s: %v\n", branchName, err)
		} else {
			fmt.Fprintf(c.deps.Stdout, "✓ Successfully deleted branch %s\n", branchName)
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
		fmt.Fprintf(c.deps.Stderr, "⚠ Warning: Could not check for uncommitted changes: %v\n", err)
	} else if hasChanges {
		warnings = append(warnings, "You have uncommitted changes")
	}

	// Check for unpushed commits
	hasUnpushed, err := c.deps.Git.HasUnpushedCommits()
	if err != nil {
		fmt.Fprintf(c.deps.Stderr, "⚠ Warning: Could not check for unpushed commits: %v\n", err)
	} else if hasUnpushed {
		warnings = append(warnings, "You have unpushed commits")
	}

	// Check if merged to origin
	isMerged, err := c.deps.Git.IsMergedToOrigin("main")
	if err != nil {
		fmt.Fprintf(c.deps.Stderr, "⚠ Warning: Could not check merge status: %v\n", err)
	} else if !isMerged {
		warnings = append(warnings, "Branch is not merged to origin/main")
	}

	return warnings
}

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
		fmt.Fprintf(c.deps.Stdout, "\n✓ Removable (%d)\n", len(removable))
		for _, status := range removable {
			dirName := filepath.Base(status.Info.Path)
			fmt.Fprintf(c.deps.Stdout, "  %s (%s)\n", dirName, status.Info.Branch)
		}
	}

	// Display non-removable worktrees
	if len(nonRemovable) > 0 {
		fmt.Fprintf(c.deps.Stdout, "\n✗ Non-removable (%d)\n", len(nonRemovable))
		for i, status := range nonRemovable {
			if i > 0 {
				fmt.Fprintf(c.deps.Stdout, "\n")
			}
			dirName := filepath.Base(status.Info.Path)
			fmt.Fprintf(c.deps.Stdout, "  %s (%s)\n", dirName, status.Info.Branch)
			if len(status.Warnings) > 0 {
				reasons := strings.Join(status.Warnings, ", ")
				fmt.Fprintf(c.deps.Stdout, "    → %s\n", reasons)
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
			fmt.Fprintf(c.deps.Stderr, "✗ Failed to remove %s: %v\n", dirName, err)
			failCount++
			continue
		}

		fmt.Fprintf(c.deps.Stdout, "✓ Removed %s\n", dirName)
		successCount++

		// Delete the branch if auto-remove is enabled
		if c.config != nil && c.config.AutoRemoveBranch && status.Info.Branch != "" {
			fmt.Fprintf(c.deps.Stdout, "Deleting branch %s...\n", status.Info.Branch)
			if err := c.deps.Git.DeleteBranch(status.Info.Branch); err != nil {
				// Don't fail the command, just warn
				fmt.Fprintf(c.deps.Stderr, "⚠ Failed to delete branch %s: %v\n", status.Info.Branch, err)
			} else {
				fmt.Fprintf(c.deps.Stdout, "✓ Deleted branch %s\n", status.Info.Branch)
			}
		}
	}

	// Summary
	fmt.Fprintf(c.deps.Stdout, "\n")
	if successCount > 0 {
		fmt.Fprintf(c.deps.Stdout, "✓ Successfully removed %d worktree(s)\n", successCount)
	}
	if failCount > 0 {
		fmt.Fprintf(c.deps.Stderr, "✗ Failed to remove %d worktree(s)\n", failCount)
		return fmt.Errorf("failed to remove %d worktree(s)", failCount)
	}

	return nil
}
