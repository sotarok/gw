package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmModelUpdate(t *testing.T) {
	t.Run("y key confirms", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 1}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if !model.confirmed {
			t.Error("expected confirmed=true after y")
		}
		if !model.done {
			t.Error("expected done=true after y")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after y")
		}
	})

	t.Run("Y key confirms", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 1}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if !model.confirmed {
			t.Error("expected confirmed=true after Y")
		}
		if !model.done {
			t.Error("expected done=true after Y")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after Y")
		}
	})

	t.Run("n key denies", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.confirmed {
			t.Error("expected confirmed=false after n")
		}
		if !model.done {
			t.Error("expected done=true after n")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after n")
		}
	})

	t.Run("N key denies", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.confirmed {
			t.Error("expected confirmed=false after N")
		}
		if !model.done {
			t.Error("expected done=true after N")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after N")
		}
	})

	t.Run("left arrow moves cursor to yes", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 1}
		msg := tea.KeyMsg{Type: tea.KeyLeft}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.cursor != 0 {
			t.Errorf("expected cursor=0 after left arrow, got %d", model.cursor)
		}
		if model.done {
			t.Error("expected done=false after left arrow")
		}
		if cmd != nil {
			t.Error("expected nil cmd for left arrow")
		}
	})

	t.Run("right arrow moves cursor to no", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyRight}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.cursor != 1 {
			t.Errorf("expected cursor=1 after right arrow, got %d", model.cursor)
		}
		if model.done {
			t.Error("expected done=false after right arrow")
		}
		if cmd != nil {
			t.Error("expected nil cmd for right arrow")
		}
	})

	t.Run("h key moves cursor to yes", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 1}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.cursor != 0 {
			t.Errorf("expected cursor=0 after h, got %d", model.cursor)
		}
		if cmd != nil {
			t.Error("expected nil cmd for h key")
		}
	})

	t.Run("l key moves cursor to no", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.cursor != 1 {
			t.Errorf("expected cursor=1 after l, got %d", model.cursor)
		}
		if cmd != nil {
			t.Error("expected nil cmd for l key")
		}
	})

	t.Run("tab toggles cursor", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyTab}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.cursor != 1 {
			t.Errorf("expected cursor=1 after tab from 0, got %d", model.cursor)
		}
		if cmd != nil {
			t.Error("expected nil cmd for tab key")
		}

		// Tab again to toggle back
		result2, _ := model.Update(msg)
		model2 := result2.(confirmModel)
		if model2.cursor != 0 {
			t.Errorf("expected cursor=0 after tab from 1, got %d", model2.cursor)
		}
	})

	t.Run("enter confirms when cursor on yes", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if !model.confirmed {
			t.Error("expected confirmed=true when cursor=0 and enter")
		}
		if !model.done {
			t.Error("expected done=true after enter")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after enter")
		}
	})

	t.Run("enter denies when cursor on no", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 1}
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.confirmed {
			t.Error("expected confirmed=false when cursor=1 and enter")
		}
		if !model.done {
			t.Error("expected done=true after enter")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after enter")
		}
	})

	t.Run("q quits with confirmed=false", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.confirmed {
			t.Error("expected confirmed=false after q")
		}
		if !model.done {
			t.Error("expected done=true after q")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after q")
		}
	})

	t.Run("ctrl+c quits with confirmed=false", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.confirmed {
			t.Error("expected confirmed=false after ctrl+c")
		}
		if !model.done {
			t.Error("expected done=true after ctrl+c")
		}
		if cmd == nil {
			t.Error("expected tea.Quit cmd after ctrl+c")
		}
	})

	t.Run("non-key message is ignored", func(t *testing.T) {
		m := confirmModel{message: "Continue?", cursor: 0}
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		result, cmd := m.Update(msg)
		model := result.(confirmModel)
		if model.cursor != 0 {
			t.Errorf("expected cursor unchanged at 0, got %d", model.cursor)
		}
		if model.done {
			t.Error("expected done=false for non-key message")
		}
		if cmd != nil {
			t.Error("expected nil cmd for non-key message")
		}
	})
}

func TestConfirmModelView(t *testing.T) {
	t.Run("shows message and options", func(t *testing.T) {
		m := confirmModel{message: "Delete this?", cursor: 0}
		view := m.View()

		if !strings.Contains(view, "Delete this?") {
			t.Error("expected view to contain message")
		}
		if !strings.Contains(view, "[Yes]") {
			t.Error("expected view to contain [Yes]")
		}
		if !strings.Contains(view, "[No]") {
			t.Error("expected view to contain [No]")
		}
		if !strings.Contains(view, "y/n") {
			t.Error("expected view to contain help text")
		}
	})

	t.Run("returns empty when done", func(t *testing.T) {
		m := confirmModel{message: "Delete this?", done: true}
		view := m.View()

		if view != "" {
			t.Errorf("expected empty view when done, got: %q", view)
		}
	})

	t.Run("both cursor positions render Yes and No", func(t *testing.T) {
		m0 := confirmModel{message: "Delete?", cursor: 0}
		m1 := confirmModel{message: "Delete?", cursor: 1}

		view0 := m0.View()
		view1 := m1.View()

		// Both views should contain [Yes] and [No] regardless of cursor
		for _, view := range []string{view0, view1} {
			if !strings.Contains(view, "[Yes]") {
				t.Error("expected view to contain [Yes]")
			}
			if !strings.Contains(view, "[No]") {
				t.Error("expected view to contain [No]")
			}
		}
	})
}

func TestConfirmModelInit(t *testing.T) {
	m := confirmModel{}
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil cmd")
	}
}

func TestShowEnvFilesList(t *testing.T) {
	t.Run("shows files list", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		ShowEnvFilesList([]string{".env", ".env.local"})

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "environment files will be copied") {
			t.Error("expected header text in output")
		}
		if !strings.Contains(output, ".env") {
			t.Error("expected .env in output")
		}
		if !strings.Contains(output, ".env.local") {
			t.Error("expected .env.local in output")
		}
	})

	t.Run("shows empty message for no files", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		ShowEnvFilesList([]string{})

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "No environment files found") {
			t.Errorf("expected empty message, got: %s", output)
		}
	})
}
