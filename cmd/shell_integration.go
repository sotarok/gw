package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sotarok/gw/internal/git"
	"github.com/spf13/cobra"
)

var (
	shellIntegrationPrintPath string
)

var shellIntegrationCmd = &cobra.Command{
	Use:   "shell-integration",
	Short: "Shell integration utilities",
	Long:  `Utilities for shell integration to enable features like automatic directory changing.`,
	RunE:  runShellIntegration,
}

func init() {
	shellIntegrationCmd.Flags().StringVar(&shellIntegrationPrintPath, "print-path", "",
		"Print the worktree path for the specified issue or branch")
	rootCmd.AddCommand(shellIntegrationCmd)
}

func runShellIntegration(cmd *cobra.Command, args []string) error {
	if shellIntegrationPrintPath == "" {
		return fmt.Errorf("no action specified. Use --print-path to get worktree path")
	}

	// Create git client
	gitClient := git.NewDefaultClient()

	// Try to find the worktree path
	worktreePath, err := findWorktreePath(gitClient, shellIntegrationPrintPath)
	if err != nil {
		// Don't print error message, just return non-zero exit code
		// Shell function will check exit code
		return err
	}

	// Print only the path
	fmt.Println(worktreePath)
	return nil
}

func findWorktreePath(gitClient git.Interface, identifier string) (string, error) {
	// Get repository name
	repoName, err := gitClient.GetRepositoryName()
	if err != nil {
		return "", err
	}

	// First, check if the expected directory exists (most common case after 'gw start')
	// This works even if git worktree list hasn't updated yet
	expectedPath := filepath.Join("..", fmt.Sprintf("%s-%s", repoName, identifier))
	absPath, err := filepath.Abs(expectedPath)
	if err == nil {
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath, nil
		}
	}

	// Next, try to find via git worktree list
	worktrees, err := gitClient.ListWorktrees()
	if err == nil {
		for _, wt := range worktrees {
			// Check for exact branch match
			if wt.Branch == identifier {
				return wt.Path, nil
			}
			// Check if branch matches issue pattern (e.g., "123/impl")
			if strings.HasPrefix(wt.Branch, identifier+"/") {
				return wt.Path, nil
			}
		}
	}

	// If not found as issue, try as branch name (for checkout command)
	sanitizedBranchName := gitClient.SanitizeBranchNameForDirectory(identifier)
	if sanitizedBranchName != identifier {
		expectedPath = filepath.Join("..", fmt.Sprintf("%s-%s", repoName, sanitizedBranchName))
		absPath, err = filepath.Abs(expectedPath)
		if err == nil {
			if info, err := os.Stat(absPath); err == nil && info.IsDir() {
				return absPath, nil
			}
		}
	}

	return "", fmt.Errorf("worktree not found for: %s", identifier)
}
