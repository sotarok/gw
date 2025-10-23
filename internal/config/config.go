package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	trueValue = "true"

	// Config keys
	autoCDKey           = "auto_cd"
	updateITerm2TabKey  = "update_iterm2_tab"
	autoRemoveBranchKey = "auto_remove_branch"
	copyEnvsKey         = "copy_envs"
)

// Item represents a single configuration item with metadata
type Item struct {
	Key         string
	Value       bool
	Description string
	Default     bool
}

// Config represents the gw configuration
type Config struct {
	AutoCD           bool  `toml:"auto_cd"`
	UpdateITerm2Tab  bool  `toml:"update_iterm2_tab"`
	AutoRemoveBranch bool  `toml:"auto_remove_branch"`
	CopyEnvs         *bool `toml:"copy_envs"` // Pointer to distinguish between unset and false
}

// New creates a new Config with default values
func New() *Config {
	return &Config{
		AutoCD:           true,  // Default to true for backward compatibility
		UpdateITerm2Tab:  false, // Default to false to avoid unexpected behavior
		AutoRemoveBranch: false, // Default to false to avoid unexpected behavior
		CopyEnvs:         nil,   // nil means not configured, will prompt user
	}
}

// Load loads configuration from the specified file path
func Load(path string) (*Config, error) {
	config := New()

	// If file doesn't exist, return default config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key = value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case autoCDKey:
			config.AutoCD = value == trueValue
		case updateITerm2TabKey:
			config.UpdateITerm2Tab = value == trueValue
		case autoRemoveBranchKey:
			config.AutoRemoveBranch = value == trueValue
		case copyEnvsKey:
			boolValue := value == trueValue
			config.CopyEnvs = &boolValue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return config, nil
}

// Save saves the configuration to the specified file path
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var copyEnvsStr string
	if c.CopyEnvs != nil {
		copyEnvsStr = fmt.Sprintf("copy_envs = %v\n", *c.CopyEnvs)
	} else {
		copyEnvsStr = "# copy_envs = false  # Uncomment to set default behavior\n"
	}

	content := fmt.Sprintf(`# gw configuration file
auto_cd = %v
update_iterm2_tab = %v
auto_remove_branch = %v
%s`, c.AutoCD, c.UpdateITerm2Tab, c.AutoRemoveBranch, copyEnvsStr)

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the default configuration file path
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to HOME environment variable
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".gwrc")
}

// GetConfigItems returns all configuration items with their descriptions
func (c *Config) GetConfigItems() []Item {
	copyEnvsValue := false
	if c.CopyEnvs != nil {
		copyEnvsValue = *c.CopyEnvs
	}

	return []Item{
		{
			Key:         autoCDKey,
			Value:       c.AutoCD,
			Description: "Automatically change directory to the new worktree after creation",
			Default:     true,
		},
		{
			Key:         updateITerm2TabKey,
			Value:       c.UpdateITerm2Tab,
			Description: "Update iTerm2 tab title with worktree information (macOS only)",
			Default:     false,
		},
		{
			Key:         autoRemoveBranchKey,
			Value:       c.AutoRemoveBranch,
			Description: "Automatically delete local branch after successful worktree removal",
			Default:     false,
		},
		{
			Key:         copyEnvsKey,
			Value:       copyEnvsValue,
			Description: "Automatically copy .env files to new worktrees (prompt if not set)",
			Default:     false,
		},
	}
}

// SetConfigItem sets a configuration value by key
func (c *Config) SetConfigItem(key string, value bool) error {
	switch key {
	case autoCDKey:
		c.AutoCD = value
	case updateITerm2TabKey:
		c.UpdateITerm2Tab = value
	case autoRemoveBranchKey:
		c.AutoRemoveBranch = value
	case copyEnvsKey:
		c.CopyEnvs = &value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	return nil
}
