package model

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/google/uuid"
	"github.com/steventhorne/sheepdog/config"
)

var (
	styleList = lipgloss.NewStyle().
		Width(30).
		Border(lipgloss.NormalBorder(), false, true, false, false)
	styleIdle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a0a8b7"))
	styleRunning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8ebd6b"))
	styleErrored = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e55561"))
	styleExited = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cc9057"))
)

type processList struct {
	processes       []*process
	processesByID   map[uuid.UUID]*process
	selectedProcess int
}

func newProcessList(config config.Config) processList {
	pl := processList{
		processes:     make([]*process, 0),
		processesByID: make(map[uuid.UUID]*process),
	}

	for _, pConfig := range config.Processes {
		p := newProcess(pConfig)
		pl.processesByID[p.id] = p
		pl.processes = append(pl.processes, p)
	}

	return pl
}

func (m *processList) GetSelectedProcess() *process {
	return m.processes[m.selectedProcess]
}

func (m *processList) Init() tea.Cmd {
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

func (m *processList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(m.processes)+1)
	for _, p := range m.processes {
		_, cmd := p.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Quit):
			for _, p := range m.processes {
				p.Cancel()
			}
		case key.Matches(msg, DefaultKeyMap.Down):
			if m.selectedProcess < len(m.processes) - 1 {
				m.processes[m.selectedProcess].SetSelected(false)
				m.selectedProcess++
				m.processes[m.selectedProcess].SetSelected(true)
			}
		case key.Matches(msg, DefaultKeyMap.Up):
			if m.selectedProcess > 0 {
				m.processes[m.selectedProcess].SetSelected(false)
				m.selectedProcess--
				m.processes[m.selectedProcess].SetSelected(true)
			}
		case key.Matches(msg, DefaultKeyMap.Run):
			cmds = append(cmds, m.processes[m.selectedProcess].Run())
		case key.Matches(msg, DefaultKeyMap.Kill):
			cmd := m.processes[m.selectedProcess].Kill()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *processList) getProcessEnum() list.Enumerator {
	return func(items list.Items, i int) string {
		var s string
		if m.processes[i].selected {
			s = "- "
		} else {
			s = "  "
		}
		switch m.processes[i].status {
		case statusIdle:
			return s + " "
		case statusRunning:
			return s + "R"
		case statusErrored:
			return s + "E"
		case statusExited:
			return s + " "
		default:
			return s + " "
		}
	}
}

func (m *processList) getProcessEnumStyle() list.StyleFunc {
	return func(items list.Items, i int) lipgloss.Style {
		var s lipgloss.Style
		switch m.processes[i].status {
		case statusRunning:
			s = styleRunning
		case statusErrored:
			s = styleErrored
		default:
			s = styleIdle
		}
		return s.MarginRight(1)
	}
}

func (m *processList) getProcessItemStyle() list.StyleFunc {
	return func(items list.Items, i int) lipgloss.Style {
		s := lipgloss.NewStyle()
		if m.processes[i].selected {
			s = s.Reverse(true)
		}
		return s
	}
}

func (m *processList) View() string {
	procs := make([]any, len(m.processes))
	for i, p := range m.processes {
		procs[i] = p
	}

	l := list.New(procs...).
		Enumerator(m.getProcessEnum()).
		EnumeratorStyleFunc(m.getProcessEnumStyle()).
		ItemStyleFunc(m.getProcessItemStyle())

	return styleList.Render(fmt.Sprint(l))
}
