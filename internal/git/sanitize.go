package git

import (
	"regexp"
	"strings"
)

// SanitizeBranchNameForDirectory converts a branch name to a safe directory name
// by replacing problematic characters with hyphens
func SanitizeBranchNameForDirectory(branchName string) string {
	// Define characters that are problematic in directory names across different OS
	// Windows: \ / : * ? " < > |
	// Unix: mainly / (null character is also problematic but unlikely in branch names)

	// Replace common path separators
	sanitized := strings.ReplaceAll(branchName, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")

	// Replace other problematic characters
	re := regexp.MustCompile(`[*?:<>"|]`)
	sanitized = re.ReplaceAllString(sanitized, "-")

	// Replace multiple consecutive hyphens with a single hyphen
	re = regexp.MustCompile(`-+`)
	sanitized = re.ReplaceAllString(sanitized, "-")

	// Trim leading and trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// If the result is empty (very unlikely), use a default
	if sanitized == "" {
		sanitized = "branch"
	}

	return sanitized
}
