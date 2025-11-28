package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/iterm2"
	"github.com/sotarok/gw/internal/spinner"
	"github.com/sotarok/gw/internal/ui"
)

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

	// Create worktree with spinner
	sp := spinner.New(fmt.Sprintf("Creating worktree for branch '%s'...", branch), c.deps.Stdout)
	sp.Start()
	createErr := c.deps.Git.CreateWorktreeFromBranch(worktreePath, branch, branchName)
	sp.Stop()
	if createErr != nil {
		return fmt.Errorf("failed to create worktree: %w", createErr)
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
				fmt.Fprintf(c.deps.Stderr, "%s Could not change to worktree directory: %v\n", coloredWarning(), err)
			}
		}
	}

	// Handle environment files
	if err := c.handleEnvFiles(originalDir, absolutePath); err != nil {
		// Don't fail the command, just warn
		fmt.Fprintf(c.deps.Stderr, "%s Failed to handle env files: %v\n", coloredWarning(), err)
	}

	// Run package manager setup
	if err := c.deps.Detect.RunSetup(absolutePath); err != nil {
		// Don't fail if setup fails, just warn
		fmt.Fprintf(c.deps.Stderr, "%s Setup failed: %v\n", coloredWarning(), err)
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
	// Get all branches (local and remote) with spinner
	sp := spinner.New("Fetching branches...", c.deps.Stdout)
	sp.Start()
	branches, err := c.deps.Git.ListAllBranches()
	sp.Stop()
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
