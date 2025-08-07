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
			rcPath := filepath.Join(tempDir, ".bashrc")
			cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, rcPath)
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
	rcPath := filepath.Join(tempDir, ".bashrc")
	cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, rcPath)
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

func TestNewInitCommand(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	configPath := "/tmp/test.gwrc"

	cmd := NewInitCommand(stdin, stdout, stderr, configPath)

	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}
	if cmd.stdin != stdin {
		t.Error("Expected stdin to be set correctly")
	}
	if cmd.stdout != stdout {
		t.Error("Expected stdout to be set correctly")
	}
	if cmd.stderr != stderr {
		t.Error("Expected stderr to be set correctly")
	}
	if cmd.configPath != configPath {
		t.Error("Expected configPath to be set correctly")
	}
	if cmd.rcPath != "" {
		t.Error("Expected rcPath to be empty by default")
	}
}

func TestInitCommand_DetectShellType(t *testing.T) {
	tests := []struct {
		name     string
		shellEnv string
		expected string
	}{
		{
			name:     "detects zsh from SHELL env",
			shellEnv: "/bin/zsh",
			expected: "zsh",
		},
		{
			name:     "detects bash from SHELL env",
			shellEnv: "/usr/local/bin/bash",
			expected: "bash",
		},
		{
			name:     "detects fish from SHELL env",
			shellEnv: "/usr/bin/fish",
			expected: "fish",
		},
		{
			name:     "unknown shell defaults to unknown",
			shellEnv: "/bin/sh",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp home dir to avoid checking real rc files
			tempHome := t.TempDir()
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", tempHome)
			defer os.Setenv("HOME", oldHome)

			// Save and restore SHELL env
			oldShell := os.Getenv("SHELL")
			os.Setenv("SHELL", tt.shellEnv)
			defer os.Setenv("SHELL", oldShell)

			cmd := &InitCommand{}
			result := cmd.detectShellType()

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInitCommand_DetectRCPath(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		expected string
	}{
		{
			name:     "zsh rc path",
			shell:    "zsh",
			expected: ".zshrc",
		},
		{
			name:     "bash rc path",
			shell:    "bash",
			expected: ".bashrc",
		},
		{
			name:     "fish rc path",
			shell:    "fish",
			expected: ".config/fish/config.fish",
		},
		{
			name:     "unknown shell returns empty",
			shell:    "unknown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &InitCommand{}
			result := cmd.detectRCPath(tt.shell)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty string, got %s", result)
				}
			} else {
				// Check if the result ends with expected path
				if !strings.HasSuffix(result, tt.expected) {
					t.Errorf("Expected path to end with %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestInitCommand_ShowManualInstructions(t *testing.T) {
	tests := []struct {
		name          string
		shell         string
		expectStrings []string
	}{
		{
			name:  "bash instructions",
			shell: "bash",
			expectStrings: []string{
				"Detected shell: bash",
				"Add to ~/.bashrc",
				"eval \"$(gw shell-integration --show-script --shell=bash)\"",
			},
		},
		{
			name:  "zsh instructions",
			shell: "zsh",
			expectStrings: []string{
				"Detected shell: zsh",
				"Add to ~/.zshrc",
				"eval \"$(gw shell-integration --show-script --shell=zsh)\"",
			},
		},
		{
			name:  "fish instructions",
			shell: "fish",
			expectStrings: []string{
				"Detected shell: fish",
				"Add to ~/.config/fish/config.fish",
				"gw shell-integration --show-script --shell=fish | source",
			},
		},
		{
			name:  "unknown shell shows all instructions",
			shell: "unknown",
			expectStrings: []string{
				"# For bash",
				"# For zsh",
				"# For fish",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			cmd := &InitCommand{
				stdout: stdout,
				stderr: stderr,
			}

			err := cmd.showManualInstructions(tt.shell)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			output := stdout.String()
			for _, expected := range tt.expectStrings {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q", expected)
				}
			}
		})
	}
}

func TestInitCommand_AddShellIntegration_Errors(t *testing.T) {
	tests := []struct {
		name          string
		setupRcFile   func(string) error
		expectedError string
	}{
		{
			name: "handles file read error",
			setupRcFile: func(path string) error {
				// Create a directory instead of a file to cause read error
				return os.Mkdir(path, 0755)
			},
			expectedError: "failed to read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			rcPath := filepath.Join(tempDir, ".bashrc")

			if tt.setupRcFile != nil {
				if err := tt.setupRcFile(rcPath); err != nil {
					t.Fatalf("Failed to setup rc file: %v", err)
				}
			}

			cmd := &InitCommand{}
			err := cmd.addShellIntegration(rcPath, "bash")

			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %v", tt.expectedError, err)
				}
			}
		})
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
			name:           "user enables auto-cd and shell integration added automatically",
			userInput:      "y\ny\n", // Enable auto-cd, enable shell integration
			shellPath:      "/bin/bash",
			expectRcUpdate: true, // Now writes to rc file
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Shell Integration") {
					t.Error("Expected shell integration prompt")
				}
				if !strings.Contains(output, "✓ Shell integration added to") {
					t.Error("Expected success message")
				}
			},
		},
		{
			name:           "user enables auto-cd but declines shell integration instructions",
			userInput:      "y\nn\n", // Enable auto-cd, decline shell integration instructions
			shellPath:      "/bin/bash",
			expectRcUpdate: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Shell Integration") {
					t.Error("Expected shell integration prompt")
				}
				if !strings.Contains(output, "Shell integration setup skipped") {
					t.Error("Expected skip message")
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
			name:           "shell integration already exists - shows update instructions",
			userInput:      "y\ny\n", // Enable auto-cd, enable shell integration
			shellPath:      "/bin/bash",
			expectRcUpdate: false, // Should not update because it already exists
			existingRc:     true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "⚠️  Shell integration already exists") {
					t.Error("Expected warning about existing shell integration")
				}
				if !strings.Contains(output, "The shell integration is already set up using the eval method") {
					t.Error("Expected message about eval method")
				}
				if !strings.Contains(output, "If you need to update or modify the integration") {
					t.Error("Expected update instructions")
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
eval "$(gw shell-integration --show-script --shell=bash)"
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
				} else if !strings.Contains(string(content), "gw shell-integration --show-script") {
					t.Errorf("Expected shell integration command in rc file")
				}
			}
		})
	}
}
