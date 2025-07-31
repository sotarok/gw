package cmd

import (
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
	if len(args) > 0 {
		issueNumber = args[0]
	}

	// Use the new command structure
	deps := DefaultDependencies()
	endCmd := NewEndCommand(deps, forceEnd)
	return endCmd.Execute(issueNumber)
}
