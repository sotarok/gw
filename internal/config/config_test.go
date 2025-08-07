package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	config := New()

	// Default value should be true for auto-cd
	if !config.AutoCD {
		t.Error("Expected AutoCD to be true by default")
	}
}

func TestLoadConfig_FileNotExists(t *testing.T) {
	// Create a temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error when config file doesn't exist, got: %v", err)
	}

	// Should return default config
	if !config.AutoCD {
		t.Error("Expected AutoCD to be true by default")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create a temp directory and config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Write test config
	configContent := `# gw configuration file
auto_cd = false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.AutoCD {
		t.Error("Expected AutoCD to be false based on config file")
	}
}

func TestSaveConfig(t *testing.T) {
	// Create a temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Create config with custom values
	config := &Config{
		AutoCD: false,
	}

	err := config.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load and verify
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loaded.AutoCD != config.AutoCD {
		t.Errorf("Loaded config doesn't match saved config. Expected AutoCD=%v, got %v",
			config.AutoCD, loaded.AutoCD)
	}
}

func TestGetConfigPath(t *testing.T) {
	// Test with HOME environment variable
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	testHome := "/test/home"
	os.Setenv("HOME", testHome)

	path := GetConfigPath()
	expected := filepath.Join(testHome, ".gwrc")

	if path != expected {
		t.Errorf("Expected config path %s, got %s", expected, path)
	}
}

func TestLoadConfig_InvalidFormat(t *testing.T) {
	// Create a temp directory and config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	tests := []struct {
		name           string
		configContent  string
		expectedAutoCD bool
	}{
		{
			name: "invalid line format (missing =)",
			configContent: `# gw configuration file
auto_cd true
`,
			expectedAutoCD: true, // Should use default
		},
		{
			name: "comments and empty lines",
			configContent: `# This is a comment
# Another comment

auto_cd = false

# Trailing comment
`,
			expectedAutoCD: false,
		},
		{
			name: "unknown configuration key",
			configContent: `auto_cd = true
unknown_key = value
`,
			expectedAutoCD: true,
		},
		{
			name:           "invalid boolean value",
			configContent:  `auto_cd = yes`,
			expectedAutoCD: false, // Any non-"true" value is false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test config
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			config, err := Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if config.AutoCD != tt.expectedAutoCD {
				t.Errorf("Expected AutoCD to be %v, got %v", tt.expectedAutoCD, config.AutoCD)
			}
		})
	}
}

func TestLoadConfig_FilePermissionError(t *testing.T) {
	// Skip this test on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a temp directory and config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Create a file with no read permissions
	if err := os.WriteFile(configPath, []byte("auto_cd = true"), 0000); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Try to load the config
	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error when reading file without permissions")
	}
}

func TestSaveConfig_DirectoryCreation(t *testing.T) {
	// Create a temp directory with nested path
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nested", "dir", ".gwrc")

	config := &Config{
		AutoCD: true,
	}

	// Save should create the directory structure
	err := config.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if !os.FileMode(0600).IsRegular() {
		t.Error("Config file should have 0600 permissions")
	}

	expectedContent := "# gw configuration file\nauto_cd = true\n"
	if string(content) != expectedContent {
		t.Errorf("Expected content:\n%s\nGot:\n%s", expectedContent, string(content))
	}
}
