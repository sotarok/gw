package iterm2

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// isITerm2Env is a variable function for testing
var isITerm2Env = func() string {
	return os.Getenv("TERM_PROGRAM")
}

// IsITerm2 checks if the current terminal is iTerm2
func IsITerm2() bool {
	termProgram := isITerm2Env()
	return strings.Contains(termProgram, "iTerm")
}

// ShouldUpdateTab checks if tab name should be updated based on config and environment
func ShouldUpdateTab(enabled bool) bool {
	return enabled && IsITerm2()
}

// UpdateTabName updates the iTerm2 tab name with the given repository and identifier
func UpdateTabName(w io.Writer, repoName, identifier string) error {
	tabName := FormatTabName(repoName, identifier)
	// iTerm2 escape sequence for setting tab title
	_, err := fmt.Fprintf(w, "\033]0;%s\007", tabName)
	return err
}

// ResetTabName resets the iTerm2 tab name to default
func ResetTabName(w io.Writer) error {
	// iTerm2 escape sequence for resetting tab title
	_, err := fmt.Fprintf(w, "\033]0;\007")
	return err
}

// FormatTabName formats the tab name from repository name and identifier
func FormatTabName(repoName, identifier string) string {
	repoName = strings.TrimSpace(repoName)
	identifier = strings.TrimSpace(identifier)

	if repoName == "" && identifier == "" {
		return ""
	}

	if repoName == "" {
		return identifier
	}

	if identifier == "" {
		return repoName
	}

	return fmt.Sprintf("%s %s", repoName, identifier)
}

// ExtractIssueFromBranch extracts issue number or returns branch name
// For branches like "123/impl", returns "123"
// For branches like "feature/new-feature", returns the full branch name
func ExtractIssueFromBranch(branch string) string {
	if branch == "" {
		return ""
	}

	// Check if branch follows the pattern "number/something"
	parts := strings.Split(branch, "/")
	if len(parts) >= 2 && isNumericIssue(parts[0]) {
		return parts[0]
	}

	return branch
}

// GetIdentifierFromBranch gets the appropriate identifier from a branch name
// For issue branches (123/impl), returns just the issue number
// For other branches, returns the full branch name
func GetIdentifierFromBranch(branch string) string {
	if branch == "" {
		return ""
	}

	// Check if branch follows the pattern "number/something"
	parts := strings.Split(branch, "/")
	if len(parts) >= 2 && isNumericIssue(parts[0]) {
		return parts[0]
	}

	return branch
}

// ExtractIssueNumber attempts to extract just the issue number from various formats
func ExtractIssueNumber(input string) string {
	if input == "" {
		return ""
	}

	// If it's already just a number, return it
	if isNumericIssue(input) {
		return input
	}

	// Try to extract number from patterns like "issue/123" or "123/impl"
	parts := strings.Split(input, "/")
	for _, part := range parts {
		if isNumericIssue(part) {
			return part
		}
	}

	return input
}

// isNumericIssue checks if a string contains only digits
func isNumericIssue(s string) bool {
	if s == "" {
		return false
	}

	match, _ := regexp.MatchString(`^\d+$`, s)
	return match
}
