package cmd

import (
	"bufio"
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
			name: "user selects all true",
			// Enable all options + shell integration
			userInput: "y\ny\ny\ny\ny\ny\n",
			checkConfig: func(t *testing.T, cfg *config.Config) {
				if !cfg.AutoCD {
					t.Error("Expected AutoCD to be true")
				}
				if !cfg.UpdateITerm2Tab {
					t.Error("Expected UpdateITerm2Tab to be true")
				}
				if !cfg.AutoRemoveBranch {
					t.Error("Expected AutoRemoveBranch to be true")
				}
				if cfg.CopyEnvs == nil || !*cfg.CopyEnvs {
					t.Error("Expected CopyEnvs to be true")
				}
			},
		},
		{
			name:      "user selects all false",
			userInput: "n\nn\nn\nn\nn\n", // Disable all (auto-cd, iterm2, auto-remove, copy-envs, fetch-before-command)
			checkConfig: func(t *testing.T, cfg *config.Config) {
				if cfg.AutoCD {
					t.Error("Expected AutoCD to be false")
				}
				if cfg.UpdateITerm2Tab {
					t.Error("Expected UpdateITerm2Tab to be false")
				}
				if cfg.AutoRemoveBranch {
					t.Error("Expected AutoRemoveBranch to be false")
				}
				if cfg.CopyEnvs != nil && *cfg.CopyEnvs {
					t.Error("Expected CopyEnvs to be false")
				}
			},
		},
		{
			name:      "user uses defaults (press enter)",
			userInput: "\n\n\n\n\ny\n", // Use defaults (true, false, false, false, true), enable shell integration
			checkConfig: func(t *testing.T, cfg *config.Config) {
				if !cfg.AutoCD {
					t.Error("Expected AutoCD to be true (default)")
				}
				if cfg.UpdateITerm2Tab {
					t.Error("Expected UpdateITerm2Tab to be false (default)")
				}
				if cfg.AutoRemoveBranch {
					t.Error("Expected AutoRemoveBranch to be false (default)")
				}
				// copy_envs default is nil (not configured), which is fine
			},
		},
		{
			name:      "mixed selections",
			userInput: "n\ny\ny\nn\nn\n", // Disable auto-cd, enable iterm2, enable auto-remove, disable copy-envs, disable fetch-before-command
			checkConfig: func(t *testing.T, cfg *config.Config) {
				if cfg.AutoCD {
					t.Error("Expected AutoCD to be false")
				}
				if !cfg.UpdateITerm2Tab {
					t.Error("Expected UpdateITerm2Tab to be true")
				}
				if !cfg.AutoRemoveBranch {
					t.Error("Expected AutoRemoveBranch to be true")
				}
				if cfg.CopyEnvs != nil && *cfg.CopyEnvs {
					t.Error("Expected CopyEnvs to be false")
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
	stdin := strings.NewReader("y\n\n\n\n\ny\n") // Confirm overwrite, use defaults (true, false, false, false), enable shell integration
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
			name: "user enables auto-cd and shell integration added automatically",
			// auto-cd=y, iterm2=n, auto-remove=n, copy-envs=n, fetch=default, shell-int=y
			userInput:      "y\nn\nn\nn\n\ny\n",
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
			name: "user enables auto-cd but declines shell integration instructions",
			// auto-cd=y, iterm2=n, auto-remove=n, copy-envs=n, fetch=default, shell-int=n
			userInput:      "y\nn\nn\nn\n\nn\n",
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
			name: "user disables auto-cd, no shell integration prompt",
			// Disable all options
			userInput:      "n\nn\nn\nn\nn\n",
			shellPath:      "/bin/bash",
			expectRcUpdate: false,
			checkOutput: func(t *testing.T, output string) {
				if strings.Contains(output, "Shell Integration") {
					t.Error("Should not show shell integration prompt when auto-cd is disabled")
				}
			},
		},
		{
			name: "shell integration already exists - shows update instructions",
			// auto-cd=y, iterm2=n, auto-remove=n, copy-envs=n, fetch=default, shell-int=y
			userInput:      "y\nn\nn\nn\n\ny\n",
			shellPath:      "/bin/bash",
			expectRcUpdate: false, // Should not update because it already exists
			existingRc:     true,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "⚠ Shell integration already exists") {
					t.Error("Expected warning about existing shell integration")
				}
				if !strings.Contains(output, "To update or reinstall") {
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

// Additional init tests for uncovered paths

func TestInitCommand_ExistingConfig_UserDeclinesOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Create existing config
	existingConfig := &config.Config{AutoCD: true}
	if err := existingConfig.Save(configPath); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// User declines overwrite
	stdin := strings.NewReader("n\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rcPath := filepath.Join(tempDir, ".bashrc")
	cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, rcPath)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Configuration initialization canceled") {
		t.Error("Expected cancel message")
	}
}

func TestInitCommand_PromptForConfigItem_InvalidInput(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Provide invalid input for first config item, then defaults for the rest
	stdin := strings.NewReader("invalid\nn\nn\nn\nn\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rcPath := filepath.Join(tempDir, ".bashrc")
	cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, rcPath)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Invalid input, using default") {
		t.Error("Expected 'Invalid input' message for invalid response")
	}
}

func TestInitCommand_GetEvalCommand_Fish(t *testing.T) {
	cmd := &InitCommand{}
	result := cmd.getEvalCommand("fish")
	expected := "gw shell-integration --show-script"
	if !strings.Contains(result, expected) {
		t.Errorf("Expected fish eval command to contain %q, got %q", expected, result)
	}
	// Fish command should NOT contain 'eval "$(...)"'
	if strings.Contains(result, "eval") {
		t.Errorf("Fish eval command should not contain 'eval', got %q", result)
	}
}

func TestInitCommand_GetEvalCommand_Bash(t *testing.T) {
	cmd := &InitCommand{}
	result := cmd.getEvalCommand("bash")
	if !strings.Contains(result, "eval") {
		t.Errorf("Expected bash eval command to contain 'eval', got %q", result)
	}
	if !strings.Contains(result, "--shell=bash") {
		t.Errorf("Expected bash eval command to contain '--shell=bash', got %q", result)
	}
}

func TestInitCommand_AddShellIntegration_FishShell(t *testing.T) {
	tempDir := t.TempDir()
	rcPath := filepath.Join(tempDir, "config.fish")

	cmd := &InitCommand{}
	err := cmd.addShellIntegration(rcPath, "fish")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("Failed to read rc file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "| source") {
		t.Errorf("Expected fish-style 'source' piping, got: %s", contentStr)
	}
	if strings.Contains(contentStr, "eval") {
		t.Errorf("Fish integration should not use eval, got: %s", contentStr)
	}
}

func TestInitCommand_AddShellIntegration_FileWithoutTrailingNewline(t *testing.T) {
	tempDir := t.TempDir()
	rcPath := filepath.Join(tempDir, ".bashrc")

	// Create file WITHOUT trailing newline
	if err := os.WriteFile(rcPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to write rc file: %v", err)
	}

	cmd := &InitCommand{}
	err := cmd.addShellIntegration(rcPath, "bash")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("Failed to read rc file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "gw shell-integration") {
		t.Error("Expected shell integration to be added")
	}
}

func TestInitCommand_AddShellIntegration_OpenFileError(t *testing.T) {
	// Use a path within a non-existent directory
	rcPath := "/nonexistent/deep/path/.bashrc"

	cmd := &InitCommand{}
	err := cmd.addShellIntegration(rcPath, "bash")

	if err == nil || !strings.Contains(err.Error(), "failed to open") {
		t.Errorf("Expected 'failed to open' error, got: %v", err)
	}
}

func TestInitCommand_OfferShellIntegration_RcPathEmpty(t *testing.T) {
	tempDir := t.TempDir()

	// Set an unknown shell to get empty rcPath
	oldShell := os.Getenv("SHELL")
	os.Setenv("SHELL", "/bin/unknownshell")
	defer os.Setenv("SHELL", oldShell)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	reader := strings.NewReader("y\n")
	cmd := &InitCommand{
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: stderr,
	}

	err := cmd.offerShellIntegration(bufio.NewReader(reader))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	// Should show manual instructions since rcPath is empty
	if !strings.Contains(output, "To enable shell integration, add the following line") {
		t.Errorf("Expected manual instructions, got: %s", output)
	}
}

func TestInitCommand_OfferShellIntegration_AddShellIntegrationError(t *testing.T) {
	tempDir := t.TempDir()

	// Set bash shell
	oldShell := os.Getenv("SHELL")
	os.Setenv("SHELL", "/bin/bash")
	defer os.Setenv("SHELL", oldShell)

	// Create rcPath as a directory (to cause write error)
	rcPath := filepath.Join(tempDir, ".bashrc")
	os.Mkdir(rcPath, 0755)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	reader := strings.NewReader("y\n")
	cmd := NewInitCommandWithShell(strings.NewReader(""), stdout, stderr, filepath.Join(tempDir, ".gwrc"), rcPath)

	err := cmd.offerShellIntegration(bufio.NewReader(reader))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "Failed to add shell integration") {
		t.Errorf("Expected shell integration failure warning, got stderr: %s", stderrOutput)
	}

	// Should fall back to manual instructions
	output := stdout.String()
	if !strings.Contains(output, "To enable shell integration") {
		t.Errorf("Expected manual instructions after error, got: %s", output)
	}
}

func TestInitCommand_DetectShellType_ByRCFile(t *testing.T) {
	// Save original SHELL env
	oldShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", oldShell)

	tests := []struct {
		name     string
		rcFile   string
		expected string
	}{
		{
			name:     "detects zsh from .zshrc",
			rcFile:   ".zshrc",
			expected: "zsh",
		},
		{
			name:     "detects bash from .bashrc",
			rcFile:   ".bashrc",
			expected: "bash",
		},
		{
			name:     "detects fish from config.fish",
			rcFile:   ".config/fish/config.fish",
			expected: "fish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempHome := t.TempDir()

			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", tempHome)
			defer os.Setenv("HOME", oldHome)

			// Set unknown shell to force rc file detection
			os.Setenv("SHELL", "/bin/sh")

			// Create the rc file
			rcFilePath := filepath.Join(tempHome, tt.rcFile)
			os.MkdirAll(filepath.Dir(rcFilePath), 0755)
			os.WriteFile(rcFilePath, []byte("# shell config"), 0644)

			cmd := &InitCommand{}
			result := cmd.detectShellType()

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInitCommand_OfferShellIntegration_UserDeclinesSetup(t *testing.T) {
	reader := strings.NewReader("n\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := &InitCommand{
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: stderr,
	}

	err := cmd.offerShellIntegration(bufio.NewReader(reader))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Shell integration setup skipped") {
		t.Error("Expected skip message")
	}
	if !strings.Contains(output, "You can set it up later") {
		t.Error("Expected later instructions message")
	}
}

func TestInitCommand_HasShellIntegration_FileDoesNotExist(t *testing.T) {
	cmd := &InitCommand{}
	result := cmd.hasShellIntegration("/nonexistent/path/.bashrc", "eval")
	if result {
		t.Error("Expected false for non-existent file")
	}
}

func TestInitCommand_HasShellIntegration_CommandExists(t *testing.T) {
	tempDir := t.TempDir()
	rcPath := filepath.Join(tempDir, ".bashrc")
	evalCommand := `eval "$(gw shell-integration --show-script --shell=bash)"`
	os.WriteFile(rcPath, []byte("# config\n"+evalCommand+"\n"), 0644)

	cmd := &InitCommand{}
	result := cmd.hasShellIntegration(rcPath, evalCommand)
	if !result {
		t.Error("Expected true when eval command exists in file")
	}
}

func TestInitCommand_Execute_ReadStringErrorOnOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Create existing config
	existingConfig := &config.Config{AutoCD: true}
	if err := existingConfig.Save(configPath); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// Provide empty stdin (no newline) - causes ReadString to return EOF
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rcPath := filepath.Join(tempDir, ".bashrc")
	cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, rcPath)
	err := cmd.Execute()

	if err == nil || !strings.Contains(err.Error(), "failed to read input") {
		t.Errorf("Expected 'failed to read input' error, got: %v", err)
	}
}

func TestInitCommand_Execute_ReadStringErrorOnConfigItem(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Provide stdin that runs out mid-way (no config file exists, so no overwrite prompt)
	// Need to provide no input at all to trigger error on first config item prompt
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rcPath := filepath.Join(tempDir, ".bashrc")
	cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, rcPath)
	err := cmd.Execute()

	if err == nil || !strings.Contains(err.Error(), "failed to read input") {
		t.Errorf("Expected 'failed to read input' error, got: %v", err)
	}
}

func TestInitCommand_Execute_SaveError(t *testing.T) {
	tempDir := t.TempDir()
	// Use a path where we can't write (non-existent deeply nested read-only dir)
	configPath := filepath.Join(tempDir, "readonly", "deep", ".gwrc")

	// Make the parent directory read-only
	readOnlyDir := filepath.Join(tempDir, "readonly")
	os.MkdirAll(readOnlyDir, 0755)
	os.Chmod(readOnlyDir, 0444)
	defer os.Chmod(readOnlyDir, 0755)

	// Provide all inputs (5 config items)
	stdin := strings.NewReader("n\nn\nn\nn\nn\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	rcPath := filepath.Join(tempDir, ".bashrc")
	cmd := NewInitCommandWithShell(stdin, stdout, stderr, configPath, rcPath)
	err := cmd.Execute()

	if err == nil || !strings.Contains(err.Error(), "failed to save configuration") {
		t.Errorf("Expected 'failed to save configuration' error, got: %v", err)
	}
}

func TestFormatKeyForPrompt(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"auto_cd", "auto-cd"},
		{"update_iterm2_tab", "iTerm2 tab updates"},
		{"auto_remove_branch", "auto-remove branch"},
		{"fetch_before_command", "fetch before command"},
		{"some_other_key", "some other key"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := formatKeyForPrompt(tt.key)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
