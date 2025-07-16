package git

import "testing"

func TestSanitizeBranchNameForDirectory(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple branch name",
			input:    "feature-auth",
			expected: "feature-auth",
		},
		{
			name:     "branch with forward slash",
			input:    "feature/auth",
			expected: "feature-auth",
		},
		{
			name:     "branch with multiple slashes",
			input:    "feature/auth/login",
			expected: "feature-auth-login",
		},
		{
			name:     "branch with backslash",
			input:    "feature\\auth",
			expected: "feature-auth",
		},
		{
			name:     "branch with asterisk",
			input:    "feature*auth",
			expected: "feature-auth",
		},
		{
			name:     "branch with question mark",
			input:    "feature?auth",
			expected: "feature-auth",
		},
		{
			name:     "branch with colon",
			input:    "feature:auth",
			expected: "feature-auth",
		},
		{
			name:     "branch with angle brackets",
			input:    "feature<auth>",
			expected: "feature-auth",
		},
		{
			name:     "branch with pipe",
			input:    "feature|auth",
			expected: "feature-auth",
		},
		{
			name:     "branch with quotes",
			input:    `feature"auth"`,
			expected: "feature-auth",
		},
		{
			name:     "branch with multiple problematic characters",
			input:    "feature/auth:*test?<>|\"",
			expected: "feature-auth-test",
		},
		{
			name:     "branch with consecutive slashes",
			input:    "feature//auth///test",
			expected: "feature-auth-test",
		},
		{
			name:     "branch starting and ending with slashes",
			input:    "/feature/auth/",
			expected: "feature-auth",
		},
		{
			name:     "only problematic characters",
			input:    "///***???",
			expected: "branch",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "branch",
		},
		{
			name:     "remote branch",
			input:    "origin/feature/auth",
			expected: "origin-feature-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBranchNameForDirectory(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeBranchNameForDirectory(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
