package iterm2

import (
	"bytes"
	"testing"
)

func TestUpdateTabName(t *testing.T) {
	tests := []struct {
		name           string
		repoName       string
		identifier     string
		expectedOutput string
	}{
		{
			name:           "updates tab with repo and issue number",
			repoName:       "my-repo",
			identifier:     "123",
			expectedOutput: "\033]0;my-repo 123\007",
		},
		{
			name:           "updates tab with repo and branch name",
			repoName:       "awesome-project",
			identifier:     "feature/new-feature",
			expectedOutput: "\033]0;awesome-project feature/new-feature\007",
		},
		{
			name:           "handles empty identifier",
			repoName:       "test-repo",
			identifier:     "",
			expectedOutput: "\033]0;test-repo\007",
		},
		{
			name:           "handles empty repo name",
			repoName:       "",
			identifier:     "456",
			expectedOutput: "\033]0;456\007",
		},
		{
			name:           "handles both empty",
			repoName:       "",
			identifier:     "",
			expectedOutput: "\033]0;\007",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			err := UpdateTabName(&output, tt.repoName, tt.identifier)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if output.String() != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, output.String())
			}
		})
	}
}

func TestIsITerm2(t *testing.T) {
	tests := []struct {
		name     string
		termEnv  string
		expected bool
	}{
		{
			name:     "detects iTerm2",
			termEnv:  "iTerm2",
			expected: true,
		},
		{
			name:     "detects iTerm.app",
			termEnv:  "iTerm.app",
			expected: true,
		},
		{
			name:     "not iTerm2 - Terminal.app",
			termEnv:  "Apple_Terminal",
			expected: false,
		},
		{
			name:     "not iTerm2 - xterm",
			termEnv:  "xterm-256color",
			expected: false,
		},
		{
			name:     "empty TERM_PROGRAM",
			termEnv:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the environment check function
			originalFunc := isITerm2Env
			defer func() { isITerm2Env = originalFunc }()

			isITerm2Env = func() string {
				return tt.termEnv
			}

			result := IsITerm2()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestResetTabName(t *testing.T) {
	var output bytes.Buffer
	err := ResetTabName(&output)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// iTerm2 reset sequence
	expectedOutput := "\033]0;\007"
	if output.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output.String())
	}
}

func TestFormatTabName(t *testing.T) {
	tests := []struct {
		name       string
		repoName   string
		identifier string
		expected   string
	}{
		{
			name:       "formats with both values",
			repoName:   "my-repo",
			identifier: "123",
			expected:   "my-repo 123",
		},
		{
			name:       "formats with only repo",
			repoName:   "my-repo",
			identifier: "",
			expected:   "my-repo",
		},
		{
			name:       "formats with only identifier",
			repoName:   "",
			identifier: "feature/branch",
			expected:   "feature/branch",
		},
		{
			name:       "trims whitespace",
			repoName:   "  my-repo  ",
			identifier: "  123  ",
			expected:   "my-repo 123",
		},
		{
			name:       "empty values",
			repoName:   "",
			identifier: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTabName(tt.repoName, tt.identifier)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestShouldUpdateTab(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		termEnv  string
		expected bool
	}{
		{
			name:     "enabled and in iTerm2",
			enabled:  true,
			termEnv:  "iTerm2",
			expected: true,
		},
		{
			name:     "enabled but not in iTerm2",
			enabled:  true,
			termEnv:  "xterm",
			expected: false,
		},
		{
			name:     "disabled but in iTerm2",
			enabled:  false,
			termEnv:  "iTerm2",
			expected: false,
		},
		{
			name:     "disabled and not in iTerm2",
			enabled:  false,
			termEnv:  "xterm",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the environment check function
			originalFunc := isITerm2Env
			defer func() { isITerm2Env = originalFunc }()

			isITerm2Env = func() string {
				return tt.termEnv
			}

			result := ShouldUpdateTab(tt.enabled)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractIssueFromBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "extracts issue number from standard branch",
			branch:   "123/impl",
			expected: "123",
		},
		{
			name:     "extracts issue number from feature branch",
			branch:   "456/feature",
			expected: "456",
		},
		{
			name:     "returns full branch name if no slash",
			branch:   "main",
			expected: "main",
		},
		{
			name:     "returns full branch name for feature/branch format",
			branch:   "feature/new-feature",
			expected: "feature/new-feature",
		},
		{
			name:     "handles empty branch",
			branch:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractIssueFromBranch(tt.branch)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractIssueNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extracts pure number",
			input:    "123",
			expected: "123",
		},
		{
			name:     "extracts from issue/number format",
			input:    "issue/456",
			expected: "456",
		},
		{
			name:     "extracts from number/impl format",
			input:    "789/impl",
			expected: "789",
		},
		{
			name:     "returns input if not a number pattern",
			input:    "feature/branch",
			expected: "feature/branch",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractIssueNumber(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetIdentifierFromBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "gets issue number from standard format",
			branch:   "123/impl",
			expected: "123",
		},
		{
			name:     "keeps feature branch format",
			branch:   "feature/new-feature",
			expected: "feature/new-feature",
		},
		{
			name:     "keeps main branch",
			branch:   "main",
			expected: "main",
		},
		{
			name:     "handles complex branch name",
			branch:   "bugfix/issue-789/fix-bug",
			expected: "bugfix/issue-789/fix-bug",
		},
		{
			name:     "handles empty branch",
			branch:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIdentifierFromBranch(tt.branch)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsNumericIssue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "numeric string",
			input:    "123",
			expected: true,
		},
		{
			name:     "numeric with leading zeros",
			input:    "00456",
			expected: true,
		},
		{
			name:     "non-numeric string",
			input:    "feature",
			expected: false,
		},
		{
			name:     "mixed alphanumeric",
			input:    "123abc",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericIssue(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
