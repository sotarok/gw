package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sotarok/gw/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gw configuration",
	Long:  `Initialize gw configuration by creating a .gwrc file in your home directory.`,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := config.GetConfigPath()
	initCmd := NewInitCommand(os.Stdin, os.Stdout, os.Stderr, configPath)
	return initCmd.Execute()
}

// InitCommand handles the init command logic
type InitCommand struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	configPath string
	rcPath     string // For testing shell integration
}

// NewInitCommand creates a new init command handler
func NewInitCommand(stdin io.Reader, stdout, stderr io.Writer, configPath string) *InitCommand {
	return &InitCommand{
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		configPath: configPath,
	}
}

// NewInitCommandWithShell creates a new init command handler with custom shell rc path (for testing)
func NewInitCommandWithShell(stdin io.Reader, stdout, stderr io.Writer, configPath, rcPath string) *InitCommand {
	return &InitCommand{
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		configPath: configPath,
		rcPath:     rcPath,
	}
}

const (
	yes = "yes"
	no  = "no"

	shellBash = "bash"
	shellZsh  = "zsh"
	shellFish = "fish"
)

// Execute runs the init command
func (c *InitCommand) Execute() error {
	fmt.Fprintln(c.stdout, "Welcome to gw configuration!")
	fmt.Fprintln(c.stdout)

	reader := bufio.NewReader(c.stdin)

	// Check if config already exists
	if _, err := os.Stat(c.configPath); err == nil {
		fmt.Fprintf(c.stdout, "Configuration file already exists at %s\n", c.configPath)
		fmt.Fprint(c.stdout, "Do you want to overwrite it? (y/N): ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != yes {
			fmt.Fprintln(c.stdout, "Configuration initialization canceled.")
			return nil
		}
		fmt.Fprintln(c.stdout)
	}

	// Create new config with defaults
	cfg := config.New()

	// Get all config items from the config definition
	items := cfg.GetConfigItems()

	// Prompt for each configuration item
	for _, item := range items {
		if err := c.promptForConfigItem(reader, cfg, item); err != nil {
			return err
		}
	}

	// Save configuration
	if err := cfg.Save(c.configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Fprintln(c.stdout)
	fmt.Fprintf(c.stdout, "Configuration saved to %s\n", c.configPath)
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Configuration summary:")

	// Show summary of all settings
	for _, item := range cfg.GetConfigItems() {
		fmt.Fprintf(c.stdout, "  %s: %v\n", item.Description, item.Value)
	}

	// If auto-cd is enabled, offer shell integration
	if cfg.AutoCD {
		if err := c.offerShellIntegration(reader); err != nil {
			// Don't fail the command, just warn
			fmt.Fprintf(c.stderr, "⚠ Shell integration setup failed: %v\n", err)
		}
	}

	return nil
}

func (c *InitCommand) promptForConfigItem(reader *bufio.Reader, cfg *config.Config, item config.Item) error {
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, item.Description)

	// Format the prompt with default value
	prompt := fmt.Sprintf("Enable %s? ", formatKeyForPrompt(item.Key))
	if item.Default {
		prompt += "(Y/n): "
	} else {
		prompt += "(y/N): "
	}

	fmt.Fprint(c.stdout, prompt)

	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	// Determine the value based on response and default
	var newValue bool
	if response == "" {
		// Use default if no input
		newValue = item.Default
	} else if response == "y" || response == yes {
		newValue = true
	} else if response == "n" || response == no {
		newValue = false
	} else {
		// Invalid input, use default
		fmt.Fprintf(c.stdout, "Invalid input, using default: %v\n", item.Default)
		newValue = item.Default
	}

	// Update the config value
	if err := cfg.SetConfigItem(item.Key, newValue); err != nil {
		return fmt.Errorf("failed to set %s: %w", item.Key, err)
	}

	return nil
}

// formatKeyForPrompt converts config key to human-readable format
func formatKeyForPrompt(key string) string {
	// Convert snake_case to human readable
	switch key {
	case "auto_cd":
		return "auto-cd"
	case "update_iterm2_tab":
		return "iTerm2 tab updates"
	case "auto_remove_branch":
		return "auto-remove branch"
	default:
		return strings.ReplaceAll(key, "_", " ")
	}
}

func (c *InitCommand) offerShellIntegration(reader *bufio.Reader) error {
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Shell Integration")
	fmt.Fprintln(c.stdout, "=================")
	fmt.Fprintln(c.stdout, "To automatically change to the new worktree directory after 'gw start',")
	fmt.Fprintln(c.stdout, "you need to add shell integration to your shell configuration file.")
	fmt.Fprintln(c.stdout)
	fmt.Fprint(c.stdout, "Would you like to set up shell integration? (y/N): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != yes {
		fmt.Fprintln(c.stdout, "Shell integration setup skipped.")
		fmt.Fprintln(c.stdout, "You can set it up later by following the instructions in the documentation.")
		return nil
	}

	// Detect shell and rc file
	shell := c.detectShellType()
	rcPath := c.rcPath // Use test path if provided
	if rcPath == "" {
		rcPath = c.detectRCPath(shell)
	}

	if rcPath == "" {
		// Can't detect rc file, show manual instructions
		return c.showManualInstructions(shell)
	}

	// Check if eval command already exists
	evalCommand := c.getEvalCommand(shell)
	if c.hasShellIntegration(rcPath, evalCommand) {
		return c.showUpdateInstructions(rcPath, shell)
	}

	// Add shell integration
	if err := c.addShellIntegration(rcPath, shell); err != nil {
		// If failed, show manual instructions
		fmt.Fprintf(c.stderr, "⚠ Failed to add shell integration: %v\n", err)
		return c.showManualInstructions(shell)
	}

	fmt.Fprintln(c.stdout)
	fmt.Fprintf(c.stdout, "✓ Shell integration added to %s\n", rcPath)
	fmt.Fprintln(c.stdout, "Please restart your shell or run:")
	fmt.Fprintf(c.stdout, "  source %s\n", rcPath)

	return nil
}

func (c *InitCommand) detectShellType() string {
	// First try SHELL environment variable
	shellPath := os.Getenv("SHELL")
	shell := filepath.Base(shellPath)

	switch {
	case strings.Contains(shell, shellZsh):
		return shellZsh
	case strings.Contains(shell, shellBash):
		return shellBash
	case strings.Contains(shell, shellFish):
		return shellFish
	default:
		// Try to detect based on existing rc files
		home, _ := os.UserHomeDir()
		if home != "" {
			if _, err := os.Stat(filepath.Join(home, ".zshrc")); err == nil {
				return shellZsh
			}
			if _, err := os.Stat(filepath.Join(home, ".bashrc")); err == nil {
				return shellBash
			}
			if _, err := os.Stat(filepath.Join(home, ".config", "fish", "config.fish")); err == nil {
				return shellFish
			}
		}
		return "unknown"
	}
}

func (c *InitCommand) detectRCPath(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	switch shell {
	case shellZsh:
		return filepath.Join(home, ".zshrc")
	case shellBash:
		return filepath.Join(home, ".bashrc")
	case shellFish:
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return ""
	}
}

func (c *InitCommand) getEvalCommand(shell string) string {
	switch shell {
	case shellFish:
		return "gw shell-integration --show-script"
	default:
		return fmt.Sprintf("eval \"$(gw shell-integration --show-script --shell=%s)\"", shell)
	}
}

func (c *InitCommand) hasShellIntegration(rcPath, evalCommand string) bool {
	content, err := os.ReadFile(rcPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), evalCommand)
}

func (c *InitCommand) showManualInstructions(shell string) error {
	fmt.Fprintln(c.stdout)
	fmt.Fprintf(c.stdout, "Detected shell: %s\n", shell)
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "To enable shell integration, add the following line to your shell configuration file:")
	fmt.Fprintln(c.stdout)

	switch shell {
	case shellZsh:
		fmt.Fprintln(c.stdout, "  # Add to ~/.zshrc")
		fmt.Fprintf(c.stdout, "  eval \"$(gw shell-integration --show-script --shell=%s)\"\n", shellZsh)
	case shellBash:
		fmt.Fprintln(c.stdout, "  # Add to ~/.bashrc")
		fmt.Fprintf(c.stdout, "  eval \"$(gw shell-integration --show-script --shell=%s)\"\n", shellBash)
	case shellFish:
		fmt.Fprintln(c.stdout, "  # Add to ~/.config/fish/config.fish")
		fmt.Fprintf(c.stdout, "  gw shell-integration --show-script --shell=%s | source\n", shellFish)
	default:
		fmt.Fprintln(c.stdout, "  # For bash (add to ~/.bashrc)")
		fmt.Fprintf(c.stdout, "  eval \"$(gw shell-integration --show-script --shell=%s)\"\n", shellBash)
		fmt.Fprintln(c.stdout)
		fmt.Fprintln(c.stdout, "  # For zsh (add to ~/.zshrc)")
		fmt.Fprintf(c.stdout, "  eval \"$(gw shell-integration --show-script --shell=%s)\"\n", shellZsh)
		fmt.Fprintln(c.stdout)
		fmt.Fprintln(c.stdout, "  # For fish (add to ~/.config/fish/config.fish)")
		fmt.Fprintf(c.stdout, "  gw shell-integration --show-script --shell=%s | source\n", shellFish)
	}

	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "After adding the line, restart your shell or source your configuration file.")

	return nil
}

func (c *InitCommand) showUpdateInstructions(rcPath, shell string) error {
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "⚠ Shell integration already exists in your configuration.")
	fmt.Fprintf(c.stdout, "  File: %s\n", rcPath)
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "To update or reinstall, please:")
	fmt.Fprintln(c.stdout, "1. Remove the existing gw shell integration from your file")
	fmt.Fprintln(c.stdout, "2. Add the following line:")
	fmt.Fprintf(c.stdout, "   %s\n", c.getEvalCommand(shell))

	return nil
}

func (c *InitCommand) addShellIntegration(rcPath, shell string) error {
	// Read existing content
	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", rcPath, err)
	}

	// Open file for appending
	file, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", rcPath, err)
	}
	defer file.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write to %s: %w", rcPath, err)
		}
	}

	// Add shell integration comment and command
	shellIntegration := "\n# gw shell integration\n"
	if shell == shellFish {
		shellIntegration += fmt.Sprintf("gw shell-integration --show-script --shell=%s | source\n", shell)
	} else {
		shellIntegration += fmt.Sprintf("eval \"$(gw shell-integration --show-script --shell=%s)\"\n", shell)
	}

	if _, err := file.WriteString(shellIntegration); err != nil {
		return fmt.Errorf("failed to write shell integration: %w", err)
	}

	return nil
}
