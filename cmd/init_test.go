package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sotarok/gw/internal/config"
)

func TestInitCommand_Execute(t *testing.T) {
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tests := []struct {
		name          string
		userInput     string // Simulated user input
		expectedError string
		checkConfig   func(t *testing.T, cfg *config.Config)
	}{
		{
			name:      "user selects auto-cd true",
			userInput: "y\ny\n", // Enable auto-cd, enable shell integration
			checkConfig: func(t *testing.T, cfg *config.Config) {
				if !cfg.AutoCD {
					t.Error("Expected AutoCD to be true")
				}
			},
		},
		{
			name:      "user selects auto-cd false",
			userInput: "n\n",
			checkConfig: func(t *testing.T, cfg *config.Config) {
				if cfg.AutoCD {
					t.Error("Expected AutoCD to be false")
				}
			},
		},
		{
			name:      "user uses default (press enter)",
			userInput: "\ny\n", // Use default (true), enable shell integration
			checkConfig: func(t *testing.T, cfg *config.Config) {
				if !cfg.AutoCD {
					t.Error("Expected AutoCD to be true (default)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for config
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, ".gwrc")

			// Setup mock stdin
			stdin := strings.NewReader(tt.userInput)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// Create init command with test rc path
			testRcPath := filepath.Join(tempDir, ".bashrc")
			cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, testRcPath)
			err := cmd.Execute()

			// Check error
			if tt.expectedError != "" {
				if err == nil || err.Error() != tt.expectedError {
					t.Errorf("Expected error %q, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Load and check config
			if tt.checkConfig != nil {
				cfg, err := config.Load(configPath)
				if err != nil {
					t.Fatalf("Failed to load config: %v", err)
				}
				tt.checkConfig(t, cfg)
			}

			// Check output contains expected messages
			output := stdout.String()
			if !strings.Contains(output, "Welcome to gw configuration") {
				t.Error("Expected welcome message in output")
			}
			if !strings.Contains(output, "Configuration saved to") {
				t.Error("Expected save confirmation in output")
			}
		})
	}
}

func TestInitCommand_ExistingConfig(t *testing.T) {
	// Create temp directory and existing config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Create existing config
	existingConfig := &config.Config{
		AutoCD: false,
	}
	if err := existingConfig.Save(configPath); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// Setup mock stdin/stdout
	stdin := strings.NewReader("y\n\ny\n") // Confirm overwrite, use default (true), enable shell integration
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Create init command with test rc path
	testRcPath := filepath.Join(tempDir, ".bashrc")
	cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, testRcPath)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that config was overwritten
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.AutoCD {
		t.Error("Expected AutoCD to be true after overwrite")
	}

	// Check output contains overwrite warning
	output := stdout.String()
	if !strings.Contains(output, "already exists") {
		t.Error("Expected overwrite warning in output")
	}
}

func TestInitCommand_ShellIntegration(t *testing.T) {
	tests := []struct {
		name           string
		userInput      string
		shellPath      string
		expectRcUpdate bool
		existingRc     bool // Whether to create existing shell integration
		checkOutput    func(t *testing.T, output string)
	}{
		{
			name:           "user enables auto-cd and shell integration",
			userInput:      "y\ny\n", // Enable auto-cd, enable shell integration
			shellPath:      "/bin/bash",
			expectRcUpdate: true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Shell Integration") {
					t.Error("Expected shell integration prompt")
				}
			},
		},
		{
			name:           "user enables auto-cd but declines shell integration",
			userInput:      "y\nn\n", // Enable auto-cd, decline shell integration
			shellPath:      "/bin/bash",
			expectRcUpdate: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Shell Integration") {
					t.Error("Expected shell integration prompt")
				}
			},
		},
		{
			name:           "user disables auto-cd, no shell integration prompt",
			userInput:      "n\n", // Disable auto-cd
			shellPath:      "/bin/bash",
			expectRcUpdate: false,
			checkOutput: func(t *testing.T, output string) {
				if strings.Contains(output, "Shell Integration") {
					t.Error("Should not show shell integration prompt when auto-cd is disabled")
				}
			},
		},
		{
			name:           "shell integration already exists - shows warning and instructions",
			userInput:      "y\ny\n", // Enable auto-cd, enable shell integration
			shellPath:      "/bin/bash",
			expectRcUpdate: false, // Should not update because it already exists
			existingRc:     true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "⚠️  Shell integration already exists") {
					t.Error("Expected warning about existing shell integration")
				}
				if !strings.Contains(output, "To update the shell function manually") {
					t.Error("Expected manual update instructions")
				}
				if !strings.Contains(output, "gw()") {
					t.Error("Expected shell function code to be displayed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, ".gwrc")
			rcPath := filepath.Join(tempDir, ".bashrc")

			// If test requires existing shell integration, create it
			if tt.existingRc {
				existingContent := `# Existing content
# gw shell integration
gw() {
    # Old version of the function
    command gw "$@"
}
`
				if err := os.WriteFile(rcPath, []byte(existingContent), 0644); err != nil {
					t.Fatalf("Failed to create existing rc file: %v", err)
				}
			}

			// Setup environment
			os.Setenv("SHELL", tt.shellPath)
			defer os.Unsetenv("SHELL")

			// Setup mock stdin/stdout
			stdin := strings.NewReader(tt.userInput)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// Create init command with test rc path
			cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, rcPath)
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String())
			}

			// Check if rc file was updated
			if tt.expectRcUpdate {
				content, err := os.ReadFile(rcPath)
				if err != nil {
					t.Errorf("Expected rc file to be created/updated")
				} else if !strings.Contains(string(content), "gw()") {
					t.Errorf("Expected shell function in rc file")
				}
			}
		})
	}
}
