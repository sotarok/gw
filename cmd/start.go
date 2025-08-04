package cmd

import (
	"github.com/spf13/cobra"
)

var (
	startCopyEnvs  bool
	startPrintPath bool
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
	startCmd.Flags().BoolVar(&startPrintPath, "print-path", false, "Print only the worktree path (for shell integration)")
	rootCmd.AddCommand(startCmd)
}

const defaultBaseBranch = "main"

func runStart(cmd *cobra.Command, args []string) error {
	issueNumber := args[0]
	baseBranch := defaultBaseBranch

	if len(args) > 1 {
		baseBranch = args[1]
	}

	// Use the new command structure
	deps := DefaultDependencies()
	if startPrintPath {
		// For print-path mode, suppress regular output
		deps.Stdout = nil
		deps.Stderr = nil
	}
	startCmd := NewStartCommandWithOptions(deps, startCopyEnvs, startPrintPath)
	return startCmd.Execute(issueNumber, baseBranch)
}
