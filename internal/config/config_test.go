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
