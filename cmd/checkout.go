package cmd

import (
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
	}

	// Use the new command structure
	deps := DefaultDependencies()
	checkoutCmd := NewCheckoutCommand(deps, checkoutCopyEnvs)
	return checkoutCmd.Execute(branch)
}
