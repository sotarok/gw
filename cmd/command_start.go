package cmd

import (
	"fmt"
	"os"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/iterm2"
	"github.com/sotarok/gw/internal/spinner"
)

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

	// Create the worktree with spinner
	sp := spinner.New(fmt.Sprintf("Creating worktree for issue #%s based on %s...", issueNumber, baseBranch), c.deps.Stdout)
	sp.Start()
	worktreePath, err := c.deps.Git.CreateWorktree(issueNumber, baseBranch)
	sp.Stop()
	if err != nil {
		return err
	}

	if c.deps.Stdout != nil {
		fmt.Fprintf(c.deps.Stdout, "%s Created worktree at %s\n", coloredSuccess(), worktreePath)
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
	if err := c.handleEnvFiles(originalDir, worktreePath); err != nil {
		// Don't fail the command, just warn
		if c.deps.Stderr != nil {
			fmt.Fprintf(c.deps.Stderr, "%s Failed to handle env files: %v\n", coloredWarning(), err)
		}
	}

	// Run setup if a package manager is detected
	if err := c.deps.Detect.RunSetup(worktreePath); err != nil {
		// Don't fail if setup fails, just warn
		if c.deps.Stderr != nil {
			fmt.Fprintf(c.deps.Stderr, "%s Setup failed: %v\n", coloredWarning(), err)
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
