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
)

// Config represents the gw configuration
type Config struct {
	AutoCD          bool `toml:"auto_cd"`
	UpdateITerm2Tab bool `toml:"update_iterm2_tab"`
}

// New creates a new Config with default values
func New() *Config {
	return &Config{
		AutoCD:          true,  // Default to true for backward compatibility
		UpdateITerm2Tab: false, // Default to false to avoid unexpected behavior
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
		case "auto_cd":
			config.AutoCD = value == trueValue
		case "update_iterm2_tab":
			config.UpdateITerm2Tab = value == trueValue
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

	content := fmt.Sprintf(`# gw configuration file
auto_cd = %v
update_iterm2_tab = %v
`, c.AutoCD, c.UpdateITerm2Tab)

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
