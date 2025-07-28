package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/ui"
	"github.com/spf13/cobra"
)

var (
	checkoutCopyEnvs bool
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout [branch]",
	Short: "Checkout an existing branch as a new worktree",
	Long: `Checkout an existing branch as a new worktree.
If no branch is specified, an interactive selector will be shown.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCheckout,
}

func init() {
	checkoutCmd.Flags().BoolVar(&checkoutCopyEnvs, "copy-envs", false, "Copy untracked .env files to the new worktree")
	rootCmd.AddCommand(checkoutCmd)
}

func runCheckout(cmd *cobra.Command, args []string) error {
	var branch string
	if len(args) > 0 {
		branch = args[0]
	} else {
		// Interactive mode
		selectedBranch, err := selectBranch()
		if err != nil {
			return err
		}
		branch = selectedBranch
	}

	// Get repository name
	repoName, err := git.GetRepositoryName()
	if err != nil {
		return fmt.Errorf("failed to get repository name: %w", err)
	}

	// Extract branch name without remote prefix
	branchName := branch
	if strings.HasPrefix(branch, "origin/") {
		branchName = strings.TrimPrefix(branch, "origin/")
	}

	// Create worktree directory name
	sanitizedBranchName := git.SanitizeBranchNameForDirectory(branchName)
	worktreeName := fmt.Sprintf("%s-%s", repoName, sanitizedBranchName)
	worktreePath := filepath.Join("..", worktreeName)

	// Check if branch exists
	exists, err := git.BranchExists(branch)
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
	fmt.Printf("Creating worktree for branch '%s'...\n", branch)
	if err := git.CreateWorktreeFromBranch(worktreePath, branch, branchName); err != nil {
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

	fmt.Printf("Changed directory to: %s\n", absolutePath)

	// Handle environment files
	// Use the original repository root to find env files
	envFiles, err := git.FindUntrackedEnvFiles(originalDir)
	if err != nil {
		fmt.Printf("⚠ Failed to find env files: %v\n", err)
	} else if len(envFiles) > 0 {
		shouldCopy := checkoutCopyEnvs

		// If flag not set, ask user
		if !checkoutCopyEnvs {
			fmt.Printf("\nFound %d untracked environment file(s). Copy them to the new worktree?", len(envFiles))
			confirmed, err := ui.ConfirmPrompt("")
			if err != nil {
				fmt.Printf("⚠ Failed to get user input: %v\n", err)
			} else {
				shouldCopy = confirmed
			}
		}

		if shouldCopy {
			// Show files to be copied
			filePaths := make([]string, len(envFiles))
			for i, f := range envFiles {
				filePaths[i] = f.Path
			}
			ui.ShowEnvFilesList(filePaths)

			// Copy files
			if err := git.CopyEnvFiles(envFiles, originalDir, absolutePath); err != nil {
				fmt.Printf("⚠ Failed to copy some env files: %v\n", err)
			}
		}
	}

	// Run package manager setup
	pm, err := detect.DetectPackageManager(absolutePath)
	if err != nil {
		fmt.Printf("No package manager detected, skipping setup: %v\n", err)
		return nil
	}

	if len(pm.InstallCmd) > 0 {
		fmt.Printf("Running %s install...\n", pm.Name)
		installCmd := strings.Join(pm.InstallCmd, " ")
		if err := git.RunCommand(installCmd); err != nil {
			fmt.Printf("Warning: %s install failed: %v\n", pm.Name, err)
		}
	}

	return nil
}

func selectBranch() (string, error) {
	// Get all branches (local and remote)
	branches, err := git.ListAllBranches()
	if err != nil {
		return "", fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		return "", fmt.Errorf("no branches found")
	}

	// Filter out current branch and main/master
	currentBranch, err := git.GetCurrentBranch()
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
	selected, err := ui.ShowSelector("Select a branch to checkout:", items)
	if err != nil {
		return "", err
	}

	return selected.ID, nil
}
