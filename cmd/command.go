package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	return handleEnvFiles(c.deps, c.copyEnvs, originalDir, worktreePath)
}

// handleEnvFiles is a common function for handling environment files
func handleEnvFiles(deps *Dependencies, copyEnvs bool, originalDir, worktreePath string) error {
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

	shouldCopy := copyEnvs

	// If flag not set, ask user
	if !copyEnvs {
		fmt.Fprintf(deps.Stdout, "\nFound %d untracked environment file(s):\n", len(envFiles))
		deps.UI.ShowEnvFilesList(filePaths)

		fmt.Fprintf(deps.Stdout, "\nCopy them to the new worktree?")
		confirmed, err := deps.UI.ConfirmPrompt("")
		if err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		shouldCopy = confirmed
	} else {
		// When flag is set, also show the files being copied
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
}

// NewCheckoutCommand creates a new checkout command handler
func NewCheckoutCommand(deps *Dependencies, copyEnvs bool) *CheckoutCommand {
	return &CheckoutCommand{
		deps:     deps,
		copyEnvs: copyEnvs,
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

	// Change to the new worktree directory
	absolutePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := os.Chdir(absolutePath); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	fmt.Fprintf(c.deps.Stdout, "Changed directory to: %s\n", absolutePath)

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

	return nil
}

func (c *CheckoutCommand) handleEnvFiles(originalDir, worktreePath string) error {
	return handleEnvFiles(c.deps, c.copyEnvs, originalDir, worktreePath)
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
	deps  *Dependencies
	force bool
}

// NewEndCommand creates a new end command handler
func NewEndCommand(deps *Dependencies, force bool) *EndCommand {
	return &EndCommand{
		deps:  deps,
		force: force,
	}
}

// Execute runs the end command
func (c *EndCommand) Execute(issueNumber string) error {
	var worktreePath string
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
		isInteractiveMode = true
	} else {
		// Find the worktree for this issue
		wt, err := c.deps.Git.GetWorktreeForIssue(issueNumber)
		if err != nil {
			return err
		}
		worktreePath = wt.Path
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
