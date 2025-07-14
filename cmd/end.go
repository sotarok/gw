package cmd

import (
	"bufio"
	"fmt"
	"gw/internal/git"
	"gw/internal/ui"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	forceEnd bool
)

var endCmd = &cobra.Command{
	Use:   "end [issue-number]",
	Short: "Remove a worktree for the specified issue",
	Long: `Removes a git worktree for the specified issue number.
If no issue number is provided, an interactive selector will be shown.
The command will check for uncommitted changes and unpushed commits before removing.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEnd,
}

func init() {
	rootCmd.AddCommand(endCmd)
	endCmd.Flags().BoolVarP(&forceEnd, "force", "f", false, "Force removal without safety checks")
}

func runEnd(cmd *cobra.Command, args []string) error {
	var issueNumber string
	var worktreePath string

	if len(args) == 0 {
		// Interactive mode
		fmt.Println("No issue number provided, entering interactive mode...")

		selected, err := ui.SelectWorktree()
		if err != nil {
			return err
		}

		// Extract issue number from the path or branch
		parts := strings.Split(selected.Branch, "/")
		if len(parts) > 0 {
			issueNumber = parts[0]
		}
		worktreePath = selected.Path
	} else {
		issueNumber = args[0]

		// Find the worktree for this issue
		wt, err := git.GetWorktreeForIssue(issueNumber)
		if err != nil {
			return err
		}
		worktreePath = wt.Path
	}

	if issueNumber == "" {
		return fmt.Errorf("could not determine issue number")
	}

	fmt.Printf("Checking worktree for issue #%s...\n", issueNumber)

	// Change to the worktree directory to check status
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(worktreePath); err != nil {
		return fmt.Errorf("failed to change to worktree directory: %w", err)
	}

	// Perform safety checks unless forced
	if !forceEnd {
		var warnings []string

		// Check for uncommitted changes
		hasChanges, err := git.HasUncommittedChanges()
		if err != nil {
			fmt.Printf("⚠ Warning: Could not check for uncommitted changes: %v\n", err)
		} else if hasChanges {
			warnings = append(warnings, "You have uncommitted changes")
		}

		// Check for unpushed commits
		hasUnpushed, err := git.HasUnpushedCommits()
		if err != nil {
			fmt.Printf("⚠ Warning: Could not check for unpushed commits: %v\n", err)
		} else if hasUnpushed {
			warnings = append(warnings, "You have unpushed commits")
		}

		// Check if merged to origin
		isMerged, err := git.IsMergedToOrigin("main")
		if err != nil {
			fmt.Printf("⚠ Warning: Could not check merge status: %v\n", err)
		} else if !isMerged {
			warnings = append(warnings, "Branch is not merged to origin/main")
		}

		// If there are warnings, ask for confirmation
		if len(warnings) > 0 {
			fmt.Println("\n⚠ Safety check warnings:")
			for _, warning := range warnings {
				fmt.Printf("  • %s\n", warning)
			}

			fmt.Print("\nDo you want to continue? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// Change back to original directory before removing
	if err := os.Chdir(originalDir); err != nil {
		return fmt.Errorf("failed to change back to original directory: %w", err)
	}

	fmt.Printf("Removing worktree for issue #%s...\n", issueNumber)

	// Remove the worktree
	if err := git.RemoveWorktree(issueNumber); err != nil {
		return err
	}

	fmt.Printf("✓ Successfully removed worktree for issue #%s\n", issueNumber)
	return nil
}
