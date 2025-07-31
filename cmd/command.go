package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/ui"
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
}

// NewStartCommand creates a new start command handler
func NewStartCommand(deps *Dependencies, copyEnvs bool) *StartCommand {
	return &StartCommand{
		deps:     deps,
		copyEnvs: copyEnvs,
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

	// Get the original repository root before creating worktree
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Fprintf(c.deps.Stdout, "Creating worktree for issue #%s based on %s...\n", issueNumber, baseBranch)

	// Create the worktree
	worktreePath, err := c.deps.Git.CreateWorktree(issueNumber, baseBranch)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.deps.Stdout, "✓ Created worktree at %s\n", worktreePath)

	// Change to the new worktree directory
	if err := os.Chdir(worktreePath); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	fmt.Fprintf(c.deps.Stdout, "✓ Changed to worktree directory\n")

	// Handle environment files
	if err := c.handleEnvFiles(originalDir, worktreePath); err != nil {
		// Don't fail the command, just warn
		fmt.Fprintf(c.deps.Stderr, "⚠ Failed to handle env files: %v\n", err)
	}

	// Run setup if a package manager is detected
	if err := c.deps.Detect.RunSetup(worktreePath); err != nil {
		// Don't fail if setup fails, just warn
		fmt.Fprintf(c.deps.Stderr, "⚠ Setup failed: %v\n", err)
	}

	fmt.Fprintf(c.deps.Stdout, "\n✨ Worktree ready! You are now in:\n   %s\n", worktreePath)
	return nil
}

func (c *StartCommand) handleEnvFiles(originalDir, worktreePath string) error {
	envFiles, err := c.deps.Git.FindUntrackedEnvFiles(originalDir)
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

	shouldCopy := c.copyEnvs

	// If flag not set, ask user
	if !c.copyEnvs {
		fmt.Fprintf(c.deps.Stdout, "\nFound %d untracked environment file(s):\n", len(envFiles))
		c.deps.UI.ShowEnvFilesList(filePaths)

		fmt.Fprintf(c.deps.Stdout, "\nCopy them to the new worktree?")
		confirmed, err := c.deps.UI.ConfirmPrompt("")
		if err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		shouldCopy = confirmed
	} else {
		// When flag is set, also show the files being copied
		fmt.Fprintf(c.deps.Stdout, "\nCopying environment files:\n")
		c.deps.UI.ShowEnvFilesList(filePaths)
	}

	if shouldCopy {
		// Copy files
		if err := c.deps.Git.CopyEnvFiles(envFiles, originalDir, worktreePath); err != nil {
			return fmt.Errorf("failed to copy env files: %w", err)
		}
		fmt.Fprintf(c.deps.Stdout, "✓ Environment files copied successfully\n")
	}

	return nil
}
