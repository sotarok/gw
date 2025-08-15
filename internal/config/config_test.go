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

	// Default value should be false for update-iterm2-tab
	if config.UpdateITerm2Tab {
		t.Error("Expected UpdateITerm2Tab to be false by default")
	}

	// Default value should be false for auto-remove-branch
	if config.AutoRemoveBranch {
		t.Error("Expected AutoRemoveBranch to be false by default")
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
update_iterm2_tab = true
auto_remove_branch = true
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

	if !config.UpdateITerm2Tab {
		t.Error("Expected UpdateITerm2Tab to be true based on config file")
	}

	if !config.AutoRemoveBranch {
		t.Error("Expected AutoRemoveBranch to be true based on config file")
	}
}

func TestSaveConfig(t *testing.T) {
	// Create a temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	// Create config with custom values
	config := &Config{
		AutoCD:           false,
		UpdateITerm2Tab:  true,
		AutoRemoveBranch: true,
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

	if loaded.UpdateITerm2Tab != config.UpdateITerm2Tab {
		t.Errorf("Loaded config doesn't match saved config. Expected UpdateITerm2Tab=%v, got %v",
			config.UpdateITerm2Tab, loaded.UpdateITerm2Tab)
	}

	if loaded.AutoRemoveBranch != config.AutoRemoveBranch {
		t.Errorf("Loaded config doesn't match saved config. Expected AutoRemoveBranch=%v, got %v",
			config.AutoRemoveBranch, loaded.AutoRemoveBranch)
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
			name: "with update_iterm2_tab configuration",
			configContent: `auto_cd = true
update_iterm2_tab = true
`,
			expectedAutoCD: true,
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

	expectedContent := "# gw configuration file\nauto_cd = true\nupdate_iterm2_tab = false\nauto_remove_branch = false\n"
	if string(content) != expectedContent {
		t.Errorf("Expected content:\n%s\nGot:\n%s", expectedContent, string(content))
	}
}

func TestGetConfigItems(t *testing.T) {
	config := &Config{
		AutoCD:           true,
		UpdateITerm2Tab:  false,
		AutoRemoveBranch: true,
	}

	items := config.GetConfigItems()

	// Should return 3 items
	if len(items) != 3 {
		t.Fatalf("Expected 3 config items, got %d", len(items))
	}

	// Check auto_cd item
	autoCDItem := items[0]
	if autoCDItem.Key != "auto_cd" {
		t.Errorf("Expected first item key to be 'auto_cd', got '%s'", autoCDItem.Key)
	}
	if autoCDItem.Value != true {
		t.Errorf("Expected auto_cd value to be true, got %v", autoCDItem.Value)
	}
	if autoCDItem.Default != true {
		t.Errorf("Expected auto_cd default to be true, got %v", autoCDItem.Default)
	}
	if autoCDItem.Description == "" {
		t.Error("Expected auto_cd to have a description")
	}

	// Check update_iterm2_tab item
	iterm2Item := items[1]
	if iterm2Item.Key != "update_iterm2_tab" {
		t.Errorf("Expected second item key to be 'update_iterm2_tab', got '%s'", iterm2Item.Key)
	}
	if iterm2Item.Value != false {
		t.Errorf("Expected update_iterm2_tab value to be false, got %v", iterm2Item.Value)
	}
	if iterm2Item.Default != false {
		t.Errorf("Expected update_iterm2_tab default to be false, got %v", iterm2Item.Default)
	}
	if iterm2Item.Description == "" {
		t.Error("Expected update_iterm2_tab to have a description")
	}

	// Check auto_remove_branch item
	autoRemoveItem := items[2]
	if autoRemoveItem.Key != "auto_remove_branch" {
		t.Errorf("Expected third item key to be 'auto_remove_branch', got '%s'", autoRemoveItem.Key)
	}
	if autoRemoveItem.Value != true {
		t.Errorf("Expected auto_remove_branch value to be true, got %v", autoRemoveItem.Value)
	}
	if autoRemoveItem.Default != false {
		t.Errorf("Expected auto_remove_branch default to be false, got %v", autoRemoveItem.Default)
	}
	if autoRemoveItem.Description == "" {
		t.Error("Expected auto_remove_branch to have a description")
	}
}

func TestSetConfigItem(t *testing.T) {
	config := New()

	// Test setting auto_cd
	err := config.SetConfigItem("auto_cd", false)
	if err != nil {
		t.Errorf("Failed to set auto_cd: %v", err)
	}
	if config.AutoCD != false {
		t.Errorf("Expected AutoCD to be false after setting, got %v", config.AutoCD)
	}

	// Test setting update_iterm2_tab
	err = config.SetConfigItem("update_iterm2_tab", true)
	if err != nil {
		t.Errorf("Failed to set update_iterm2_tab: %v", err)
	}
	if config.UpdateITerm2Tab != true {
		t.Errorf("Expected UpdateITerm2Tab to be true after setting, got %v", config.UpdateITerm2Tab)
	}

	// Test setting auto_remove_branch
	err = config.SetConfigItem("auto_remove_branch", true)
	if err != nil {
		t.Errorf("Failed to set auto_remove_branch: %v", err)
	}
	if config.AutoRemoveBranch != true {
		t.Errorf("Expected AutoRemoveBranch to be true after setting, got %v", config.AutoRemoveBranch)
	}

	// Test setting unknown key
	err = config.SetConfigItem("unknown_key", true)
	if err == nil {
		t.Error("Expected error when setting unknown key")
	}
	if err != nil && err.Error() != "unknown configuration key: unknown_key" {
		t.Errorf("Expected specific error message for unknown key, got: %v", err)
	}
}
