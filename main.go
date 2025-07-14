package main

import (
	"gw/cmd"
	"os"
)

// These variables are set via ldflags by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Pass version info to cmd package
	cmd.SetVersionInfo(version, commit, date)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
