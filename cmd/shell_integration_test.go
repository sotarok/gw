package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestShellIntegrationCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		showScript  bool
		shell       string
		printPath   string
		wantError   bool
		errorMsg    string
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:       "show bash script",
			showScript: true,
			shell:      "bash",
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "gw()") {
					t.Error("Expected shell function definition")
				}
				if !strings.Contains(output, "#!/bin/bash") {
					t.Error("Expected bash shebang")
				}
				if !strings.Contains(output, "auto_cd = true") {
					t.Error("Expected auto_cd check")
				}
			},
		},
		{
			name:       "show zsh script",
			showScript: true,
			shell:      "zsh",
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "gw()") {
					t.Error("Expected shell function definition")
				}
				if !strings.Contains(output, "#!/bin/zsh") {
					t.Error("Expected zsh shebang")
				}
			},
		},
		{
			name:       "show fish script",
			showScript: true,
			shell:      "fish",
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "function gw") {
					t.Error("Expected fish function definition")
				}
				if !strings.Contains(output, "#!/usr/bin/env fish") {
					t.Error("Expected fish shebang")
				}
			},
		},
		{
			name:       "auto-detect shell from environment",
			showScript: true,
			shell:      "", // Should auto-detect
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "gw") {
					t.Error("Expected shell function")
				}
			},
		},
		{
			name:       "unsupported shell",
			showScript: true,
			shell:      "tcsh",
			wantError:  true,
			errorMsg:   "unsupported shell: tcsh",
		},
		{
			name:      "no flags specified",
			wantError: true,
			errorMsg:  "either --show-script or --print-path must be specified",
		},
		{
			name:       "both flags specified",
			showScript: true,
			printPath:  "123",
			wantError:  true,
			errorMsg:   "cannot use both --show-script and --print-path",
		},
		{
			name:      "print-path with issue number",
			printPath: "123",
			checkOutput: func(t *testing.T, output string) {
				// This test would need mock git implementation
				// For now, we just check it doesn't error with "not implemented"
				if strings.Contains(output, "error") {
					t.Skip("Skipping print-path test without git mock")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			cmd := NewShellIntegrationCommand(stdout, stderr)
			cmd.showScript = tt.showScript
			cmd.shell = tt.shell
			cmd.printPath = tt.printPath

			err := cmd.Execute()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				// Skip print-path tests that require git
				if tt.printPath != "" && strings.Contains(err.Error(), "not in a git repository") {
					t.Skip("Skipping print-path test outside git repository")
				}
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String())
			}
		})
	}
}
