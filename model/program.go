package model

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/steventhorne/sheepdog/config"
)

type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Run   key.Binding
	Kill  key.Binding
	Quit  key.Binding
	Enter key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),        // actual keybindings
		key.WithHelp("↑/k", "move up"), // corresponding help text
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "move down"),
	),
	Run: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "run process"),
	),
	Kill: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "kill process"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "enter"),
	),
}

type model struct {
	processes processList
}

func NewModel(config config.Config) model {
	m := model{
		processes: newProcessList(config),
	}

	return m
}

func (m model) Init() tea.Cmd {
	return m.processes.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	_, cmd = m.processes.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Quit):
			cmds = append(cmds, tea.Quit)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.processes.View(), m.processes.GetSelectedProcess().viewport.View())
}
