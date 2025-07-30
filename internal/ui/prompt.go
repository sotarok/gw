package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmPrompt shows a yes/no prompt to the user
func ConfirmPrompt(message string) (bool, error) {
	m := confirmModel{
		message: message,
		cursor:  0, // Default to "yes"
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, err
	}

	model := result.(confirmModel)
	return model.confirmed, nil
}

type confirmModel struct {
	message   string
	cursor    int // 0 = yes, 1 = no
	confirmed bool
	done      bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.cursor = 0
		case "right", "l":
			m.cursor = 1
		case "tab":
			m.cursor = (m.cursor + 1) % 2
		case "enter":
			m.confirmed = m.cursor == 0
			m.done = true
			return m, tea.Quit
		case "y", "Y":
			m.confirmed = true
			m.done = true
			return m, tea.Quit
		case "n", "N":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}

	var s strings.Builder
	s.WriteString(m.message + "\n\n")

	yesStyle := normalStyle
	noStyle := normalStyle

	if m.cursor == 0 {
		yesStyle = selectedStyle
	} else {
		noStyle = selectedStyle
	}

	s.WriteString(yesStyle.Render("[Yes]"))
	s.WriteString("  ")
	s.WriteString(noStyle.Render("[No]"))
	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render("(y/n, ←/→ to select, enter to confirm)"))

	return s.String()
}

// ShowEnvFilesList displays a list of environment files
func ShowEnvFilesList(files []string) {
	if len(files) == 0 {
		fmt.Println("No environment files found.")
		return
	}

	fmt.Println("\nThe following environment files will be copied:")
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

	for _, file := range files {
		fmt.Printf("  %s %s\n", headerStyle.Render("→"), fileStyle.Render(file))
	}
	fmt.Println()
}
