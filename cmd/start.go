package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/ui"

	"github.com/spf13/cobra"
)

var (
	startCopyEnvs bool
)

var startCmd = &cobra.Command{
	Use:   "start <issue-number> [base-branch]",
	Short: "Create a new worktree for the specified issue",
	Long: `Creates a new git worktree for the specified issue number.
The worktree will be created in a sibling directory named '{repository-name}-{issue-number}'.
A new branch '{issue-number}/impl' will be created based on the specified base branch.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runStart,
}

func init() {
	startCmd.Flags().BoolVar(&startCopyEnvs, "copy-envs", false, "Copy untracked .env files to the new worktree")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	issueNumber := args[0]
	baseBranch := "main"

	if len(args) > 1 {
		baseBranch = args[1]
	}

	// Check if we're in a git repository
	if !git.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Check if worktree already exists
	if wt, _ := git.GetWorktreeForIssue(issueNumber); wt != nil {
		return fmt.Errorf("worktree for issue %s already exists at %s", issueNumber, wt.Path)
	}

	fmt.Printf("Creating worktree for issue #%s based on %s...\n", issueNumber, baseBranch)

	// Create the worktree
	worktreePath, err := git.CreateWorktree(issueNumber, baseBranch)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Created worktree at %s\n", worktreePath)

	// Change to the new worktree directory
	if err := os.Chdir(worktreePath); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	fmt.Printf("✓ Changed to worktree directory\n")

	// Handle environment files
	sourceRoot := filepath.Dir(worktreePath)
	envFiles, err := git.FindUntrackedEnvFiles(sourceRoot)
	if err != nil {
		fmt.Printf("⚠ Failed to find env files: %v\n", err)
	} else if len(envFiles) > 0 {
		shouldCopy := startCopyEnvs

		// If flag not set, ask user
		if !startCopyEnvs {
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
			if err := git.CopyEnvFiles(envFiles, sourceRoot, worktreePath); err != nil {
				fmt.Printf("⚠ Failed to copy some env files: %v\n", err)
			}
		}
	}

	// Run setup if a package manager is detected
	if err := detect.RunSetup(worktreePath); err != nil {
		// Don't fail if setup fails, just warn
		fmt.Printf("⚠ Setup failed: %v\n", err)
	}

	fmt.Printf("\n✨ Worktree ready! You are now in:\n   %s\n", worktreePath)
	return nil
}
