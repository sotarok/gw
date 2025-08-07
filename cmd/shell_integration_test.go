package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunShellIntegration(t *testing.T) {
	// Save original values
	originalShowScript := shellIntegrationShowScript
	originalShell := shellIntegrationShell
	originalPrintPath := shellIntegrationPrintPath
	defer func() {
		shellIntegrationShowScript = originalShowScript
		shellIntegrationShell = originalShell
		shellIntegrationPrintPath = originalPrintPath
	}()

	tests := []struct {
		name       string
		showScript bool
		shell      string
		printPath  string
		wantError  bool
	}{
		{
			name:       "run with show-script flag",
			showScript: true,
			shell:      "bash",
			wantError:  false,
		},
		{
			name:      "run with print-path flag",
			printPath: "test-123",
			wantError: true, // Will error as we're not in a git repo
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global flags
			shellIntegrationShowScript = tt.showScript
			shellIntegrationShell = tt.shell
			shellIntegrationPrintPath = tt.printPath

			// Run the command
			err := runShellIntegration(nil, []string{})

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestShellIntegrationCommand_DetectShell(t *testing.T) {
	tests := []struct {
		name     string
		shellEnv string
		expected string
	}{
		{
			name:     "detects zsh",
			shellEnv: "/usr/local/bin/zsh",
			expected: "zsh",
		},
		{
			name:     "detects bash",
			shellEnv: "/bin/bash",
			expected: "bash",
		},
		{
			name:     "detects fish",
			shellEnv: "/opt/local/bin/fish",
			expected: "fish",
		},
		{
			name:     "defaults to bash for unknown",
			shellEnv: "/bin/sh",
			expected: "bash",
		},
		{
			name:     "handles empty SHELL env",
			shellEnv: "",
			expected: "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore SHELL env
			oldShell := os.Getenv("SHELL")
			os.Setenv("SHELL", tt.shellEnv)
			defer os.Setenv("SHELL", oldShell)

			cmd := &ShellIntegrationCommand{}
			result := cmd.detectShell()

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

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
			name:      "print-path with non-existent worktree",
			printPath: "99999", // Using a high number that's unlikely to exist
			wantError: true,
			// Don't check specific error message as it depends on whether we're in a git repo or not
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
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String())
			}
		})
	}
}
