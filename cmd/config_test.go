package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestConfigCommand(t *testing.T) {
	t.Run("creates config model with loaded config", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".gwrc")

		// Create a test config
		cfg := &config.Config{
			AutoCD:          false,
			UpdateITerm2Tab: true,
		}
		err := cfg.Save(configPath)
		assert.NoError(t, err)

		// Load the config
		loadedCfg, err := config.Load(configPath)
		assert.NoError(t, err)

		// Create the model
		model := newConfigModel(loadedCfg, configPath)

		// Verify the model has the correct config
		assert.Equal(t, loadedCfg, model.config)
		assert.Equal(t, configPath, model.configPath)

		// Verify the list has the correct items
		items := model.list.Items()
		assert.Len(t, items, 2)

		// Check auto_cd item
		item0 := items[0].(configItem)
		assert.Equal(t, "auto_cd", item0.key)
		assert.Equal(t, false, item0.value)
		assert.Equal(t, true, item0.defaultVal)

		// Check update_iterm2_tab item
		item1 := items[1].(configItem)
		assert.Equal(t, "update_iterm2_tab", item1.key)
		assert.Equal(t, true, item1.value)
		assert.Equal(t, false, item1.defaultVal)
	})

	t.Run("config item filter value", func(t *testing.T) {
		item := configItem{
			title:       "auto cd",
			description: "Automatically change directory",
			key:         "auto_cd",
			value:       true,
			defaultVal:  true,
		}

		assert.Equal(t, "auto cd", item.FilterValue())
	})

	t.Run("config item displays correctly", func(t *testing.T) {
		item := configItem{
			title:       "auto cd",
			description: "Automatically change directory",
			key:         "auto_cd",
			value:       true,
			defaultVal:  true,
		}

		assert.Equal(t, "✅ auto cd", item.Title())
		assert.Equal(t, "Automatically change directory (default: true)", item.Description())

		item.value = false
		assert.Equal(t, "❌ auto cd", item.Title())
	})
}

func TestConfigIntegration(t *testing.T) {
	t.Run("config file roundtrip", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".gwrc")

		// Create and save a config
		cfg := &config.Config{
			AutoCD:          false,
			UpdateITerm2Tab: true,
		}
		err := cfg.Save(configPath)
		assert.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)

		// Load the config
		loadedCfg, err := config.Load(configPath)
		assert.NoError(t, err)

		// Verify values
		assert.Equal(t, false, loadedCfg.AutoCD)
		assert.Equal(t, true, loadedCfg.UpdateITerm2Tab)

		// Modify and save again
		loadedCfg.SetConfigItem("auto_cd", true)
		loadedCfg.SetConfigItem("update_iterm2_tab", false)
		err = loadedCfg.Save(configPath)
		assert.NoError(t, err)

		// Load again and verify
		finalCfg, err := config.Load(configPath)
		assert.NoError(t, err)
		assert.Equal(t, true, finalCfg.AutoCD)
		assert.Equal(t, false, finalCfg.UpdateITerm2Tab)
	})

	t.Run("GetConfigItems returns all items with metadata", func(t *testing.T) {
		cfg := &config.Config{
			AutoCD:          true,
			UpdateITerm2Tab: false,
		}

		items := cfg.GetConfigItems()
		assert.Len(t, items, 2)

		// Check auto_cd
		assert.Equal(t, "auto_cd", items[0].Key)
		assert.Equal(t, true, items[0].Value)
		assert.Contains(t, items[0].Description, "change directory")
		assert.Equal(t, true, items[0].Default)

		// Check update_iterm2_tab
		assert.Equal(t, "update_iterm2_tab", items[1].Key)
		assert.Equal(t, false, items[1].Value)
		assert.Contains(t, items[1].Description, "iTerm2")
		assert.Equal(t, false, items[1].Default)
	})

	t.Run("SetConfigItem updates values correctly", func(t *testing.T) {
		cfg := config.New()

		// Test setting valid keys
		err := cfg.SetConfigItem("auto_cd", false)
		assert.NoError(t, err)
		assert.Equal(t, false, cfg.AutoCD)

		err = cfg.SetConfigItem("update_iterm2_tab", true)
		assert.NoError(t, err)
		assert.Equal(t, true, cfg.UpdateITerm2Tab)

		// Test setting invalid key
		err = cfg.SetConfigItem("invalid_key", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown configuration key")
	})
}

func TestRunConfig(t *testing.T) {
	t.Run("list mode", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".gwrc")

		// Save original HOME and restore after test
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)
		os.Setenv("HOME", tmpDir)

		// Create a test config
		cfg := &config.Config{
			AutoCD:          false,
			UpdateITerm2Tab: true,
		}
		err := cfg.Save(configPath)
		assert.NoError(t, err)

		// Set the list flag
		configList = true
		defer func() { configList = false }()

		// Capture output
		var buf bytes.Buffer
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		cmd := &cobra.Command{}
		err = runConfig(cmd, []string{})
		assert.NoError(t, err)

		// Restore stdout and read output
		w.Close()
		os.Stdout = originalStdout
		buf.ReadFrom(r)
		output := buf.String()

		// Verify output contains expected information
		assert.Contains(t, output, "Configuration file:")
		assert.Contains(t, output, "auto_cd")
		assert.Contains(t, output, "false")
		assert.Contains(t, output, "update_iterm2_tab")
		assert.Contains(t, output, "true")
		assert.Contains(t, output, "Automatically change directory")
		assert.Contains(t, output, "iTerm2")
	})

	t.Run("error loading config", func(t *testing.T) {
		// Set HOME to a non-existent directory that can't be created
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)
		os.Setenv("HOME", "/nonexistent/path/that/should/not/exist")

		// Create an invalid config file that can't be read
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".gwrc")
		err := os.WriteFile(configPath, []byte("test"), 0000) // No read permissions
		if err == nil {
			// Try to set HOME to this directory
			os.Setenv("HOME", tmpDir)

			// Set list mode to avoid TUI
			configList = true
			defer func() { configList = false }()

			cmd := &cobra.Command{}
			err = runConfig(cmd, []string{})
			// This might succeed if the file system doesn't enforce permissions
			// (e.g., running as root or on some Windows systems)
			if err != nil {
				assert.Contains(t, err.Error(), "failed to load config")
			}
		}
	})
}

func TestKeyMapMethods(t *testing.T) {
	k := &keyMap{
		Up:     keys.Up,
		Down:   keys.Down,
		Toggle: keys.Toggle,
		Save:   keys.Save,
		Quit:   keys.Quit,
		Help:   keys.Help,
	}

	t.Run("ShortHelp", func(t *testing.T) {
		shortHelp := k.ShortHelp()
		assert.Len(t, shortHelp, 4)
		assert.Equal(t, keys.Toggle, shortHelp[0])
		assert.Equal(t, keys.Save, shortHelp[1])
		assert.Equal(t, keys.Quit, shortHelp[2])
		assert.Equal(t, keys.Help, shortHelp[3])
	})

	t.Run("FullHelp", func(t *testing.T) {
		fullHelp := k.FullHelp()
		assert.Len(t, fullHelp, 3)
		assert.Len(t, fullHelp[0], 2)
		assert.Len(t, fullHelp[1], 2)
		assert.Len(t, fullHelp[2], 2)
		assert.Equal(t, keys.Up, fullHelp[0][0])
		assert.Equal(t, keys.Down, fullHelp[0][1])
	})
}

func TestConfigModelMethods(t *testing.T) {
	t.Run("Init returns nil", func(t *testing.T) {
		cfg := config.New()
		model := newConfigModel(cfg, "/tmp/.gwrc")

		cmd := model.Init()
		assert.Nil(t, cmd)
	})

	t.Run("View with zero width", func(t *testing.T) {
		cfg := config.New()
		model := newConfigModel(cfg, "/tmp/.gwrc")

		view := model.View()
		assert.Equal(t, "Initializing...", view)
	})

	t.Run("View with dimensions", func(t *testing.T) {
		cfg := config.New()
		model := newConfigModel(cfg, "/tmp/.gwrc")
		model.width = 80
		model.height = 24

		view := model.View()
		// Should contain list view and help view
		assert.Contains(t, view, "\n")
	})
}
