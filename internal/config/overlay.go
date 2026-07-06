package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
)

// OriginSource identifies where a configuration value's effective setting
// came from: the built-in default, the global ~/.gwrc, or a project-local
// .gwrc override.
type OriginSource int

const (
	OriginDefault OriginSource = iota
	OriginGlobal
	OriginProject
)

// LoadWithPresence loads configuration from path in a single read, returning
// the parsed Config, the raw file bytes (used for trust-hash computation by
// callers), and a presentKeys map recording exactly which keys appeared as a
// "key = value" line in the file — independent of whether the parsed value
// equals the default. If the file does not exist, it returns a default
// Config, nil content, and an empty presentKeys map with a nil error,
// mirroring Load's tolerant behavior.
func LoadWithPresence(path string) (cfg *Config, content []byte, presentKeys map[string]bool, err error) {
	cfg = New()
	presentKeys = map[string]bool{}

	content, err = os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil, presentKeys, nil
		}
		return nil, nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key = value
		parts := strings.SplitN(line, "=", kvParts)
		if len(parts) != kvParts {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		presentKeys[key] = true

		if spec := fieldSpecByKey(key); spec != nil {
			spec.load(cfg, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return cfg, content, presentKeys, nil
}

// MergeHooks returns a new *Config equal to base except that each of the
// three hook keys present in presentKeys takes its value from overlay
// instead of base. Non-hook keys, and hook keys absent from presentKeys,
// always keep base's value. presentKeys[key] == true is the sole authority
// for whether a key overrides — the overlay's value is never compared
// against a default to infer presence.
func MergeHooks(base, overlay *Config, presentKeys map[string]bool) *Config {
	merged := *base
	result := &merged

	for i := range fieldSpecs {
		spec := &fieldSpecs[i]
		if spec.kind != kindString {
			continue
		}
		if presentKeys[spec.key] {
			spec.setString(result, spec.getString(overlay))
		}
	}

	return result
}
