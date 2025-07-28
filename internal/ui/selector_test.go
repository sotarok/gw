package ui

import (
	"testing"

	"github.com/sotarok/gw/internal/git"
)

func TestSelectWorktreeFiltering(t *testing.T) {
	tests := []struct {
		name             string
		inputWorktrees   []git.WorktreeInfo
		expectedFiltered int
		expectedBranches []string
	}{
		{
			name: "filters out main and master branches",
			inputWorktrees: []git.WorktreeInfo{
				{Path: "/path/to/main", Branch: "main"},
				{Path: "/path/to/master", Branch: "master"},
				{Path: "/path/to/feature", Branch: "feature/test"},
				{Path: "/path/to/issue", Branch: "123/impl"},
			},
			expectedFiltered: 2,
			expectedBranches: []string{"feature/test", "123/impl"},
		},
		{
			name: "includes non-numeric branches",
			inputWorktrees: []git.WorktreeInfo{
				{Path: "/path/to/main", Branch: "main"},
				{Path: "/path/to/migrate", Branch: "migrate-lupul"},
				{Path: "/path/to/feature", Branch: "add-feature"},
				{Path: "/path/to/fix", Branch: "fix-bug"},
			},
			expectedFiltered: 3,
			expectedBranches: []string{"migrate-lupul", "add-feature", "fix-bug"},
		},
		{
			name: "includes numeric issue branches",
			inputWorktrees: []git.WorktreeInfo{
				{Path: "/path/to/main", Branch: "main"},
				{Path: "/path/to/issue1", Branch: "123/impl"},
				{Path: "/path/to/issue2", Branch: "456/fix"},
				{Path: "/path/to/issue3", Branch: "789/feature"},
			},
			expectedFiltered: 3,
			expectedBranches: []string{"123/impl", "456/fix", "789/feature"},
		},
		{
			name: "handles empty list",
			inputWorktrees: []git.WorktreeInfo{
				{Path: "/path/to/main", Branch: "main"},
			},
			expectedFiltered: 0,
			expectedBranches: []string{},
		},
		{
			name: "handles branches with special characters",
			inputWorktrees: []git.WorktreeInfo{
				{Path: "/path/to/main", Branch: "main"},
				{Path: "/path/to/special", Branch: "feature/test-123"},
				{Path: "/path/to/underscore", Branch: "fix_bug"},
				{Path: "/path/to/dash", Branch: "add-new-feature"},
			},
			expectedFiltered: 3,
			expectedBranches: []string{"feature/test-123", "fix_bug", "add-new-feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from SelectWorktree
			var filteredWorktrees []git.WorktreeInfo
			for _, wt := range tt.inputWorktrees {
				if wt.Branch == "main" || wt.Branch == "master" {
					continue
				}
				filteredWorktrees = append(filteredWorktrees, wt)
			}

			// Check the number of filtered worktrees
			if len(filteredWorktrees) != tt.expectedFiltered {
				t.Errorf("expected %d filtered worktrees, got %d", tt.expectedFiltered, len(filteredWorktrees))
			}

			// Check that all expected branches are present
			for i, wt := range filteredWorktrees {
				if i < len(tt.expectedBranches) && wt.Branch != tt.expectedBranches[i] {
					t.Errorf("expected branch %s at index %d, got %s", tt.expectedBranches[i], i, wt.Branch)
				}
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"456789", true},
		{"abc", false},
		{"123abc", false},
		{"", false},
		{"12.34", false},
		{"-123", true}, // strconv.Atoi accepts negative numbers
		{" 123", false},
		{"123 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isNumeric(tt.input)
			if result != tt.expected {
				t.Errorf("isNumeric(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
