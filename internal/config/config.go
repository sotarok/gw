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
	autoCDKey             = "auto_cd"
	updateITerm2TabKey    = "update_iterm2_tab"
	autoRemoveBranchKey   = "auto_remove_branch"
	copyEnvsKey           = "copy_envs"
	fetchBeforeCommandKey = "fetch_before_command"
	postStartHookKey      = "post_start_hook"
	postCheckoutHookKey   = "post_checkout_hook"
	preEndHookKey         = "pre_end_hook"
)

// fieldKind classifies how a config field is parsed, presented and saved.
type fieldKind int

const (
	kindBool         fieldKind = iota // plain bool toggle
	kindOptionalBool                  // copy_envs: nil = unset (prompt the user)
	kindString                        // hook commands
)

// fieldSpec is the single source of truth for one configuration key. Load,
// Save, GetConfigItems and SetConfigItem all consume this table instead of
// repeating the key list. Adding a key means adding one entry here.
type fieldSpec struct {
	key         string
	kind        fieldKind
	description string // used by GetConfigItems
	defaultBool bool   // default for GetConfigItems (kindBool / kindOptionalBool)

	// load applies a raw string value (right-hand side of "key = value") to c.
	load func(c *Config, value string)
	// setBool applies a bool value to c (Load via fieldSpec.load handles parsing;
	// SetConfigItem uses this). Only set for kindBool / kindOptionalBool.
	setBool func(c *Config, value bool)
	// getBool reads the effective bool value (kindBool / kindOptionalBool) for
	// GetConfigItems.
	getBool func(c *Config) bool
}

// fieldSpecs is the ordered single source of truth for all configuration keys.
var fieldSpecs = []fieldSpec{
	{
		key:         autoCDKey,
		kind:        kindBool,
		description: "Automatically change directory to the new worktree after creation",
		defaultBool: true,
		load:        func(c *Config, v string) { c.AutoCD = v == trueValue },
		setBool:     func(c *Config, v bool) { c.AutoCD = v },
		getBool:     func(c *Config) bool { return c.AutoCD },
	},
	{
		key:         updateITerm2TabKey,
		kind:        kindBool,
		description: "Update iTerm2 tab title with worktree information (macOS only)",
		defaultBool: false,
		load:        func(c *Config, v string) { c.UpdateITerm2Tab = v == trueValue },
		setBool:     func(c *Config, v bool) { c.UpdateITerm2Tab = v },
		getBool:     func(c *Config) bool { return c.UpdateITerm2Tab },
	},
	{
		key:         autoRemoveBranchKey,
		kind:        kindBool,
		description: "Automatically delete local branch after successful worktree removal",
		defaultBool: false,
		load:        func(c *Config, v string) { c.AutoRemoveBranch = v == trueValue },
		setBool:     func(c *Config, v bool) { c.AutoRemoveBranch = v },
		getBool:     func(c *Config) bool { return c.AutoRemoveBranch },
	},
	{
		key:         copyEnvsKey,
		kind:        kindOptionalBool,
		description: "Automatically copy .env files to new worktrees (prompt if not set)",
		defaultBool: false,
		load: func(c *Config, v string) {
			boolValue := v == trueValue
			c.CopyEnvs = &boolValue
		},
		setBool: func(c *Config, v bool) { c.CopyEnvs = &v },
		getBool: func(c *Config) bool { return c.CopyEnvs != nil && *c.CopyEnvs },
	},
	{
		key:         fetchBeforeCommandKey,
		kind:        kindBool,
		description: "Run git fetch --all --prune before commands to sync remote branch info",
		defaultBool: true,
		load:        func(c *Config, v string) { c.FetchBeforeCommand = v == trueValue },
		setBool:     func(c *Config, v bool) { c.FetchBeforeCommand = v },
		getBool:     func(c *Config) bool { return c.FetchBeforeCommand },
	},
	{
		key:  postStartHookKey,
		kind: kindString,
		load: func(c *Config, v string) { c.PostStartHook = v },
	},
	{
		key:  postCheckoutHookKey,
		kind: kindString,
		load: func(c *Config, v string) { c.PostCheckoutHook = v },
	},
	{
		key:  preEndHookKey,
		kind: kindString,
		load: func(c *Config, v string) { c.PreEndHook = v },
	},
}

// fieldSpecByKey returns the fieldSpec for key, or nil if unknown.
func fieldSpecByKey(key string) *fieldSpec {
	for i := range fieldSpecs {
		if fieldSpecs[i].key == key {
			return &fieldSpecs[i]
		}
	}
	return nil
}

// Item represents a single configuration item with metadata
type Item struct {
	Key         string
	Value       bool
	Description string
	Default     bool
}

// Config represents the gw configuration
type Config struct {
	AutoCD             bool   `toml:"auto_cd"`
	UpdateITerm2Tab    bool   `toml:"update_iterm2_tab"`
	AutoRemoveBranch   bool   `toml:"auto_remove_branch"`
	CopyEnvs           *bool  `toml:"copy_envs"` // Pointer to distinguish between unset and false
	FetchBeforeCommand bool   `toml:"fetch_before_command"`
	PostStartHook      string `toml:"post_start_hook"`
	PostCheckoutHook   string `toml:"post_checkout_hook"`
	PreEndHook         string `toml:"pre_end_hook"`
}

// New creates a new Config with default values
func New() *Config {
	return &Config{
		AutoCD:             true,  // Default to true for backward compatibility
		UpdateITerm2Tab:    false, // Default to false to avoid unexpected behavior
		AutoRemoveBranch:   false, // Default to false to avoid unexpected behavior
		CopyEnvs:           nil,   // nil means not configured, will prompt user
		FetchBeforeCommand: true,  // Default to true to ensure remote info is up-to-date
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

		if spec := fieldSpecByKey(key); spec != nil {
			spec.load(config, value)
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

	// Emit the plain bool toggles by iterating fieldSpecs (kindBool entries are
	// in the exact required output order; kindOptionalBool / kindString are
	// rendered separately below).
	var boolLines string
	for i := range fieldSpecs {
		spec := &fieldSpecs[i]
		if spec.kind != kindBool {
			continue
		}
		boolLines += fmt.Sprintf("%s = %v\n", spec.key, spec.getBool(c))
	}

	var copyEnvsStr string
	if c.CopyEnvs != nil {
		copyEnvsStr = fmt.Sprintf("%s = %v\n", copyEnvsKey, *c.CopyEnvs)
	} else {
		copyEnvsStr = fmt.Sprintf("# %s = false  # Uncomment to set default behavior\n", copyEnvsKey)
	}

	var postHookLines string
	postHookLines += saveHookLine(postStartHookKey, c.PostStartHook)
	postHookLines += saveHookLine(postCheckoutHookKey, c.PostCheckoutHook)

	preHookLines := saveHookLine(preEndHookKey, c.PreEndHook)

	content := fmt.Sprintf(`# gw configuration file
%s%s
# Hook commands executed after successful worktree operations
# Available env vars: GW_WORKTREE_PATH, GW_BRANCH_NAME, GW_REPO_NAME, GW_COMMAND
%s
# Hook commands executed before a worktree is removed (from end/clean)
# Runs with cwd set to the worktree. Same env vars as above; GW_COMMAND is "end" or "clean"
%s`, boolLines, copyEnvsStr, postHookLines, preHookLines)

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// saveHookLine renders a single hook key for Save: an active assignment when a
// value is set, otherwise a commented-out placeholder.
func saveHookLine(key, value string) string {
	if value != "" {
		return fmt.Sprintf("%s = %s\n", key, value)
	}
	return fmt.Sprintf("# %s =\n", key)
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

// GetConfigItems returns all configuration items with their descriptions.
// Only bool-valued items are returned; hook (string) keys are intentionally
// excluded because the interactive `gw config` UI is a bool toggle.
func (c *Config) GetConfigItems() []Item {
	items := make([]Item, 0, len(fieldSpecs))
	for i := range fieldSpecs {
		spec := &fieldSpecs[i]
		if spec.kind == kindString {
			continue
		}
		items = append(items, Item{
			Key:         spec.key,
			Value:       spec.getBool(c),
			Description: spec.description,
			Default:     spec.defaultBool,
		})
	}
	return items
}

// SetConfigItem sets a configuration value by key
func (c *Config) SetConfigItem(key string, value bool) error {
	spec := fieldSpecByKey(key)
	if spec == nil || spec.setBool == nil {
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	spec.setBool(c, value)
	return nil
}
