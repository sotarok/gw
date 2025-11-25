package cmd

import (
	"github.com/spf13/cobra"
)

var (
	forceClean  bool
	dryRunClean bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all safely deletable worktrees",
	Long: `Removes all git worktrees that are safe to delete.
A worktree is considered safe to delete if it meets all of the following conditions:
  1. No uncommitted changes
  2. No unpushed commits
  3. Merged to origin/main

The command will show which worktrees can be removed and which cannot (with reasons),
then ask for confirmation before removing them.`,
	Args: cobra.NoArgs,
	RunE: runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVarP(&forceClean, "force", "f", false, "Force removal without confirmation prompt")
	cleanCmd.Flags().BoolVar(&dryRunClean, "dry-run", false, "Show what would be removed without actually removing")
}

func runClean(cmd *cobra.Command, args []string) error {
	deps := DefaultDependencies()
	cleanCmd := NewCleanCommand(deps, forceClean, dryRunClean)
	return cleanCmd.Execute()
}
