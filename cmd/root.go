package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version   string
	commit    string
	buildDate string
)

var rootCmd = &cobra.Command{
	Use:   "gw",
	Short: "Git worktree CLI tool to manage worktrees easily",
	Long: `gw is a CLI tool that makes working with Git worktrees more convenient.
It provides simple commands to create and remove worktrees with automatic
setup for various package managers.`,
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}

func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	buildDate = d
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate)
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
`)
}
