package cmd

import (
	"github.com/spf13/cobra"
)

var (
	startCopyEnvs bool
)

var startCmd = &cobra.Command{
	Use:   "start <issue-number-or-branch> [base-branch]",
	Short: "Create a new worktree for the specified issue or branch",
	Long: `Creates a new git worktree for the specified issue number or branch name.

If only a number is provided (e.g., "123"), it creates:
  - Branch: {issue-number}/impl
  - Directory: ../{repository-name}-{issue-number}

If a branch name with "/" is provided (e.g., "476/impl-migration-script"), it creates:
  - Branch: Exactly as provided
  - Directory: ../{repository-name}-{sanitized-branch-name}

Examples:
  gw start 123              # Creates branch "123/impl"
  gw start 476/impl-migration-script  # Creates branch "476/impl-migration-script"
  gw start feature/new-feature        # Creates branch "feature/new-feature"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runStart,
}

func init() {
	startCmd.Flags().BoolVar(&startCopyEnvs, "copy-envs", false, "Copy untracked .env files to the new worktree")
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
	startCmd := NewStartCommand(deps, startCopyEnvs)
	return startCmd.Execute(issueNumber, baseBranch)
}
