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

	// Create new config
	cfg := config.New()

	// Ask about auto-cd
	fmt.Fprintln(c.stdout, "When creating a new worktree, do you want to automatically change to that directory?")
	fmt.Fprint(c.stdout, "Auto-cd to new worktree? (Y/n): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "n" || response == no {
		cfg.AutoCD = false
	}

	// Save configuration
	if err := cfg.Save(c.configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Fprintln(c.stdout)
	fmt.Fprintf(c.stdout, "Configuration saved to %s\n", c.configPath)
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Configuration summary:")
	fmt.Fprintf(c.stdout, "  Auto-cd to new worktree: %v\n", cfg.AutoCD)

	// If auto-cd is enabled, offer shell integration
	if cfg.AutoCD {
		if err := c.offerShellIntegration(reader); err != nil {
			// Don't fail the command, just warn
			fmt.Fprintf(c.stderr, "⚠ Shell integration setup failed: %v\n", err)
		}
	}

	return nil
}

func (c *InitCommand) offerShellIntegration(reader *bufio.Reader) error {
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Shell Integration")
	fmt.Fprintln(c.stdout, "=================")
	fmt.Fprintln(c.stdout, "To automatically change to the new worktree directory after 'gw start',")
	fmt.Fprintln(c.stdout, "we can add a shell function to your shell configuration file.")
	fmt.Fprintln(c.stdout)
	fmt.Fprint(c.stdout, "Would you like to set up shell integration? (y/N): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != yes {
		fmt.Fprintln(c.stdout, "Shell integration skipped.")
		fmt.Fprintln(c.stdout, "You can manually add the shell function later. See: gw help shell-integration")
		return nil
	}

	// Detect shell and rc file
	shell := os.Getenv("SHELL")
	rcPath := c.rcPath // Use test path if provided
	if rcPath == "" {
		rcPath = c.detectRCPath(shell)
	}

	if rcPath == "" {
		return fmt.Errorf("could not detect shell configuration file")
	}

	// Add shell function
	if err := c.addShellFunction(rcPath); err != nil {
		return err
	}

	fmt.Fprintln(c.stdout)
	fmt.Fprintf(c.stdout, "✓ Shell function added to %s\n", rcPath)
	fmt.Fprintln(c.stdout, "Please restart your shell or run:")
	fmt.Fprintf(c.stdout, "  source %s\n", rcPath)

	return nil
}

func (c *InitCommand) detectRCPath(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}

	switch {
	case strings.Contains(shell, "zsh"):
		return filepath.Join(home, ".zshrc")
	case strings.Contains(shell, "bash"):
		return filepath.Join(home, ".bashrc")
	default:
		// Try to detect based on existing files
		zshrc := filepath.Join(home, ".zshrc")
		if _, err := os.Stat(zshrc); err == nil {
			return zshrc
		}
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc
		}
		return ""
	}
}

func (c *InitCommand) addShellFunction(rcPath string) error {
	shellFunction := `
# gw shell integration
gw() {
    if [[ "$1" == "start" ]] && [[ "$*" != *"--print-path"* ]]; then
        local path=$(command gw "$@" --print-path 2>/dev/null)
        if [[ -n "$path" && -d "$path" ]]; then
            command gw "$@"
            cd "$path"
            return
        fi
    fi
    command gw "$@"
}`

	// Read existing content
	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", rcPath, err)
	}

	// Check if function already exists
	if strings.Contains(string(content), "# gw shell integration") {
		fmt.Fprintf(c.stdout, "Shell integration already exists in %s\n", rcPath)
		return nil
	}

	// Append shell function
	file, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", rcPath, err)
	}
	defer file.Close()

	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write to %s: %w", rcPath, err)
		}
	}
	if _, err := file.WriteString(shellFunction); err != nil {
		return fmt.Errorf("failed to write shell function: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write to %s: %w", rcPath, err)
	}

	return nil
}
