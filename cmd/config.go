package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sotarok/gw/internal/config"
	"github.com/spf13/cobra"
)

var (
	configList bool
)

const (
	statusTrue  = "true"
	statusFalse = "false"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and edit gw configuration",
	Long: `View and edit gw configuration stored in ~/.gwrc file.
Use arrow keys or j/k to navigate, Enter to toggle values, and q to quit.

Use --list flag to view configuration in non-interactive mode.`,
	RunE: runConfig,
}

func init() {
	configCmd.Flags().BoolVar(&configList, "list", false, "List configuration values (non-interactive)")
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	configPath := config.GetConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if configList {
		// Non-interactive mode: just list the configuration
		fmt.Printf("Configuration file: %s\n\n", configPath)
		for _, item := range cfg.GetConfigItems() {
			status := statusFalse
			if item.Value {
				status = statusTrue
			}
			fmt.Printf("%-20s: %-5s  # %s (default: %v)\n", item.Key, status, item.Description, item.Default)
		}
		return nil
	}

	model := newConfigModel(cfg, configPath)
	p := tea.NewProgram(&model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run config UI: %w", err)
	}

	return nil
}

type configItem struct {
	title       string
	description string
	key         string
	value       bool
	defaultVal  bool
}

func (i configItem) Title() string {
	status := coloredError()
	if i.value {
		status = coloredSuccess()
	}
	return fmt.Sprintf("%s %s", status, i.title)
}

func (i configItem) Description() string {
	defaultStr := "false"
	if i.defaultVal {
		defaultStr = "true"
	}
	return fmt.Sprintf("%s (default: %s)", i.description, defaultStr)
}

func (i configItem) FilterValue() string { return i.title }

type configModel struct {
	list       list.Model
	config     *config.Config
	configPath string
	keys       keyMap
	help       help.Model
	width      int
	height     int
}

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Save   key.Binding
	Quit   key.Binding
	Help   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "toggle"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "save"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

func (k *keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.Save, k.Quit, k.Help}
}

func (k *keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Toggle, k.Save},
		{k.Quit, k.Help},
	}
}

func newConfigModel(cfg *config.Config, configPath string) configModel {
	items := []list.Item{}
	for _, item := range cfg.GetConfigItems() {
		items = append(items, configItem{
			title:       strings.ReplaceAll(item.Key, "_", " "),
			description: item.Description,
			key:         item.Key,
			value:       item.Value,
			defaultVal:  item.Default,
		})
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		BorderForeground(lipgloss.Color("170"))
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "gw configuration"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("25")).
		Padding(0, 1)

	return configModel{
		list:       l,
		config:     cfg,
		configPath: configPath,
		keys:       keys,
		help:       help.New(),
	}
}

func (m *configModel) Init() tea.Cmd {
	return nil
}

func (m *configModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 3)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Toggle):
			if item, ok := m.list.SelectedItem().(configItem); ok {
				item.value = !item.value
				if err := m.config.SetConfigItem(item.key, item.value); err != nil {
					m.list.Title = fmt.Sprintf("Error: %v", err)
					return m, nil
				}

				// Update the item in the list
				items := m.list.Items()
				for i, listItem := range items {
					if ci, ok := listItem.(configItem); ok && ci.key == item.key {
						items[i] = item
						break
					}
				}
				m.list.SetItems(items)
			}
			return m, nil

		case key.Matches(msg, m.keys.Save):
			if err := m.config.Save(m.configPath); err != nil {
				m.list.Title = fmt.Sprintf("Error saving config: %v", err)
			} else {
				m.list.Title = "Configuration saved!"
			}
			return m, nil

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *configModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	helpView := m.help.View(&m.keys)
	listHeight := m.height - lipgloss.Height(helpView) - 1

	m.list.SetHeight(listHeight)

	return fmt.Sprintf("%s\n%s", m.list.View(), helpView)
}
