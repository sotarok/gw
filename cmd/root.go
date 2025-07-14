package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gw",
	Short: "Git worktree CLI tool to manage worktrees easily",
	Long: `gw is a CLI tool that makes working with Git worktrees more convenient.
It provides simple commands to create and remove worktrees with automatic
setup for various package managers.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}