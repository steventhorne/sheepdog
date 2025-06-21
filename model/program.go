package model

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/steventhorne/sheepdog/config"
)

type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Run   key.Binding
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
	processes       []*process
	processesByID   map[uuid.UUID]*process
	selectedProcess int
}

func NewModel(config config.Config) model {
	m := model{
		processes:     make([]*process, 0),
		processesByID: make(map[uuid.UUID]*process),
	}

	for _, pConfig := range config.Processes {
		p := newProcess(pConfig)
		m.processesByID[p.id] = p
		m.processes = append(m.processes, p)
	}

	return m
}

func (m model) Init() tea.Cmd {
	if len(m.processesByID) > 0 {
		m.processes[m.selectedProcess].SetSelected(true)
	}

	cmds := make([]tea.Cmd, 0, len(m.processesByID)+1)
	for _, p := range m.processesByID {
		cmd := p.Init()
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(m.processesByID)+1)
	for _, p := range m.processesByID {
		_, cmd := p.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Quit):
			for _, p := range m.processesByID {
				p.Cancel()
			}
			cmds = append(cmds, tea.Quit)
		case key.Matches(msg, DefaultKeyMap.Down):
			if !m.processes[m.selectedProcess].showViewport {
				m.processes[m.selectedProcess].SetSelected(false)
				m.selectedProcess++
				m.processes[m.selectedProcess].SetSelected(true)
				m.selectedProcess = m.selectedProcess % len(m.processes)
			}
		case key.Matches(msg, DefaultKeyMap.Up):
			if !m.processes[m.selectedProcess].showViewport {
				m.processes[m.selectedProcess].SetSelected(false)
				m.selectedProcess--
				m.processes[m.selectedProcess].SetSelected(true)
				if m.selectedProcess < 0 {
					m.selectedProcess = m.selectedProcess + len(m.processes)
				}
			}
		case key.Matches(msg, DefaultKeyMap.Enter):
			m.processes[m.selectedProcess].showViewport = !m.processes[m.selectedProcess].showViewport
		case key.Matches(msg, DefaultKeyMap.Run):
			cmds = append(cmds, m.processes[m.selectedProcess].Run())
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	sb := &strings.Builder{}
	for _, p := range m.processes {
		sb.WriteString(p.View())
	}
	return sb.String()
}
