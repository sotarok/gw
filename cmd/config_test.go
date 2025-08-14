package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sotarok/gw/internal/config"
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
