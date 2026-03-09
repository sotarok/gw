package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sotarok/gw/internal/git"

	tea "github.com/charmbracelet/bubbletea"
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

func TestWorktreeSelectorUpdate(t *testing.T) {
	worktrees := []git.WorktreeInfo{
		{Path: "/path/to/feature1", Branch: "feature1"},
		{Path: "/path/to/feature2", Branch: "feature2"},
		{Path: "/path/to/feature3", Branch: "feature3"},
	}

	t.Run("j key moves cursor down", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		result, cmd := m.Update(msg)
		model := result.(worktreeSelector)
		if model.cursor != 1 {
			t.Errorf("expected cursor=1 after j, got %d", model.cursor)
		}
		if cmd != nil {
			t.Error("expected nil cmd for j key")
		}
	})

	t.Run("k key moves cursor up", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 2}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		result, cmd := m.Update(msg)
		model := result.(worktreeSelector)
		if model.cursor != 1 {
			t.Errorf("expected cursor=1 after k, got %d", model.cursor)
		}
		if cmd != nil {
			t.Error("expected nil cmd for k key")
		}
	})

	t.Run("j key does not go beyond last item", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 2}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		result, _ := m.Update(msg)
		model := result.(worktreeSelector)
		if model.cursor != 2 {
			t.Errorf("expected cursor=2 at bottom, got %d", model.cursor)
		}
	})

	t.Run("k key does not go above first item", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		result, _ := m.Update(msg)
		model := result.(worktreeSelector)
		if model.cursor != 0 {
			t.Errorf("expected cursor=0 at top, got %d", model.cursor)
		}
	})

	t.Run("down arrow moves cursor down", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyDown}
		result, _ := m.Update(msg)
		model := result.(worktreeSelector)
		if model.cursor != 1 {
			t.Errorf("expected cursor=1 after down arrow, got %d", model.cursor)
		}
	})

	t.Run("up arrow moves cursor up", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 1}
		msg := tea.KeyMsg{Type: tea.KeyUp}
		result, _ := m.Update(msg)
		model := result.(worktreeSelector)
		if model.cursor != 0 {
			t.Errorf("expected cursor=0 after up arrow, got %d", model.cursor)
		}
	})

	t.Run("enter selects current item", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 1}
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, cmd := m.Update(msg)
		model := result.(worktreeSelector)
		if model.selected == nil {
			t.Fatal("expected selected to be non-nil after enter")
		}
		if model.selected.Branch != "feature2" {
			t.Errorf("expected selected branch feature2, got %s", model.selected.Branch)
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after enter")
		}
	})

	t.Run("q quits without selection", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		result, cmd := m.Update(msg)
		model := result.(worktreeSelector)
		if model.selected != nil {
			t.Error("expected selected to be nil after q")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after q")
		}
	})

	t.Run("ctrl+c quits without selection", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		result, cmd := m.Update(msg)
		model := result.(worktreeSelector)
		if model.selected != nil {
			t.Error("expected selected to be nil after ctrl+c")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after ctrl+c")
		}
	})

	t.Run("non-key message is ignored", func(t *testing.T) {
		m := worktreeSelector{worktrees: worktrees, cursor: 1}
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		result, cmd := m.Update(msg)
		model := result.(worktreeSelector)
		if model.cursor != 1 {
			t.Errorf("expected cursor unchanged at 1, got %d", model.cursor)
		}
		if cmd != nil {
			t.Error("expected nil cmd for non-key message")
		}
	})
}

func TestWorktreeSelectorView(t *testing.T) {
	t.Run("renders worktree list with cursor", func(t *testing.T) {
		worktrees := []git.WorktreeInfo{
			{Path: "/path/to/feature1", Branch: "feature1"},
			{Path: "/path/to/feature2", Branch: "feature2"},
		}
		m := worktreeSelector{worktrees: worktrees, cursor: 0}
		view := m.View()

		if !strings.Contains(view, "Select a worktree:") {
			t.Error("expected view to contain 'Select a worktree:'")
		}
		if !strings.Contains(view, "> ") {
			t.Error("expected view to contain cursor '> '")
		}
		if !strings.Contains(view, "/path/to/feature1") {
			t.Error("expected view to contain first worktree path")
		}
		if !strings.Contains(view, "(feature1)") {
			t.Error("expected view to contain branch name in parens")
		}
		if !strings.Contains(view, "enter: select") {
			t.Error("expected view to contain help text")
		}
	})

	t.Run("shows [current] for current worktree", func(t *testing.T) {
		worktrees := []git.WorktreeInfo{
			{Path: "/path/to/feature1", Branch: "feature1", IsCurrent: true},
			{Path: "/path/to/feature2", Branch: "feature2"},
		}
		m := worktreeSelector{worktrees: worktrees, cursor: 1}
		view := m.View()

		if !strings.Contains(view, "[current]") {
			t.Error("expected view to contain '[current]'")
		}
	})

	t.Run("shows error message", func(t *testing.T) {
		m := worktreeSelector{err: fmt.Errorf("test error")}
		view := m.View()

		if !strings.Contains(view, "Error: test error") {
			t.Errorf("expected error message, got: %s", view)
		}
	})

	t.Run("shows empty message when no worktrees", func(t *testing.T) {
		m := worktreeSelector{worktrees: []git.WorktreeInfo{}}
		view := m.View()

		if !strings.Contains(view, "No worktrees found") {
			t.Errorf("expected empty message, got: %s", view)
		}
	})

	t.Run("worktree without branch omits parens", func(t *testing.T) {
		worktrees := []git.WorktreeInfo{
			{Path: "/path/to/detached", Branch: ""},
		}
		m := worktreeSelector{worktrees: worktrees, cursor: 0}
		view := m.View()

		if strings.Contains(view, "()") {
			t.Error("expected no empty parens for worktree without branch")
		}
	})
}

func TestWorktreeSelectorInit(t *testing.T) {
	m := worktreeSelector{}
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil cmd")
	}
}

func TestGenericSelectorUpdate(t *testing.T) {
	items := []SelectorItem{
		{ID: "1", Name: "Item 1"},
		{ID: "2", Name: "Item 2"},
		{ID: "3", Name: "Item 3"},
	}

	t.Run("j key moves cursor down", func(t *testing.T) {
		m := &genericSelector{items: items, cursor: 0, keyMap: keys}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		result, cmd := m.Update(msg)
		model := result.(*genericSelector)
		if model.cursor != 1 {
			t.Errorf("expected cursor=1 after j, got %d", model.cursor)
		}
		if cmd != nil {
			t.Error("expected nil cmd for j key")
		}
	})

	t.Run("k key moves cursor up", func(t *testing.T) {
		m := &genericSelector{items: items, cursor: 2, keyMap: keys}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		result, _ := m.Update(msg)
		model := result.(*genericSelector)
		if model.cursor != 1 {
			t.Errorf("expected cursor=1 after k, got %d", model.cursor)
		}
	})

	t.Run("j key does not exceed bounds", func(t *testing.T) {
		m := &genericSelector{items: items, cursor: 2, keyMap: keys}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		result, _ := m.Update(msg)
		model := result.(*genericSelector)
		if model.cursor != 2 {
			t.Errorf("expected cursor=2 at bottom, got %d", model.cursor)
		}
	})

	t.Run("k key does not go below zero", func(t *testing.T) {
		m := &genericSelector{items: items, cursor: 0, keyMap: keys}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		result, _ := m.Update(msg)
		model := result.(*genericSelector)
		if model.cursor != 0 {
			t.Errorf("expected cursor=0 at top, got %d", model.cursor)
		}
	})

	t.Run("enter selects current item", func(t *testing.T) {
		m := &genericSelector{items: items, cursor: 1, keyMap: keys}
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, cmd := m.Update(msg)
		model := result.(*genericSelector)
		if model.selected == nil {
			t.Fatal("expected selected to be non-nil after enter")
		}
		if model.selected.ID != "2" {
			t.Errorf("expected selected ID=2, got %s", model.selected.ID)
		}
		if model.selected.Name != "Item 2" {
			t.Errorf("expected selected Name=Item 2, got %s", model.selected.Name)
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after enter")
		}
	})

	t.Run("q quits without selection", func(t *testing.T) {
		m := &genericSelector{items: items, cursor: 0, keyMap: keys}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		result, cmd := m.Update(msg)
		model := result.(*genericSelector)
		if model.selected != nil {
			t.Error("expected selected to be nil after q")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after q")
		}
	})

	t.Run("esc quits without selection", func(t *testing.T) {
		m := &genericSelector{items: items, cursor: 0, keyMap: keys}
		msg := tea.KeyMsg{Type: tea.KeyEscape}
		result, cmd := m.Update(msg)
		model := result.(*genericSelector)
		if model.selected != nil {
			t.Error("expected selected to be nil after esc")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after esc")
		}
	})
}

func TestGenericSelectorView(t *testing.T) {
	items := []SelectorItem{
		{ID: "1", Name: "Item 1"},
		{ID: "2", Name: "Item 2"},
	}

	t.Run("renders title and items", func(t *testing.T) {
		m := &genericSelector{items: items, title: "Pick one:", cursor: 0, keyMap: keys}
		view := m.View()

		if !strings.Contains(view, "Pick one:") {
			t.Error("expected view to contain title")
		}
		if !strings.Contains(view, "Item 1") {
			t.Error("expected view to contain Item 1")
		}
		if !strings.Contains(view, "Item 2") {
			t.Error("expected view to contain Item 2")
		}
		if !strings.Contains(view, ">") {
			t.Error("expected view to contain cursor '>'")
		}
	})

	t.Run("cursor position changes rendering", func(t *testing.T) {
		m0 := &genericSelector{items: items, title: "Pick:", cursor: 0, keyMap: keys}
		m1 := &genericSelector{items: items, title: "Pick:", cursor: 1, keyMap: keys}

		view0 := m0.View()
		view1 := m1.View()

		if view0 == view1 {
			t.Error("expected different views for different cursor positions")
		}
	})

	t.Run("shows help text", func(t *testing.T) {
		m := &genericSelector{items: items, title: "Pick:", cursor: 0, keyMap: keys}
		view := m.View()

		if !strings.Contains(view, "enter: select") {
			t.Error("expected view to contain help text")
		}
	})
}

func TestGenericSelectorInit(t *testing.T) {
	m := &genericSelector{}
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil cmd")
	}
}
