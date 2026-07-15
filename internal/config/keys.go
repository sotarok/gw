package config

import "fmt"

// HookKeys returns the three configuration keys that a project-local .gwrc
// may override in v1.1 (post_start_hook, post_checkout_hook, pre_end_hook).
func HookKeys() []string {
	keys := make([]string, 0, numHookKeys)
	for i := range fieldSpecs {
		if fieldSpecs[i].kind == kindString {
			keys = append(keys, fieldSpecs[i].key)
		}
	}
	return keys
}

// IsHookKey reports whether key is one of the three project-overridable
// hook keys.
func IsHookKey(key string) bool {
	spec := fieldSpecByKey(key)
	return spec != nil && spec.kind == kindString
}

// IsKnownKey reports whether key is any recognized configuration key (hook
// or otherwise). Callers use this to distinguish a genuinely unknown key
// (silently ignored, same as the global config always has) from a known
// non-hook key that a project .gwrc declared but v1.1 does not apply.
func IsKnownKey(key string) bool {
	return fieldSpecByKey(key) != nil
}

// SetHookValue sets the value of one of the three hook keys by name. It
// returns an error if key is not a recognized hook key, keeping the key
// list defined exactly once (fieldSpecs) rather than duplicated by callers
// that resolve hook values dynamically (e.g. cmd.ResolveProjectConfig).
func (c *Config) SetHookValue(key, value string) error {
	spec := fieldSpecByKey(key)
	if spec == nil || spec.kind != kindString {
		return fmt.Errorf("unknown hook configuration key: %s", key)
	}
	spec.setString(c, value)
	return nil
}
