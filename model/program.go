package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/steventhorne/sheepdog/config"
)

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
		id := uuid.New()
		p := newProcess(id, pConfig.Name, pConfig.Command)
		m.processesByID[id] = p
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
		key := msg.String()
		if msg.Type == tea.KeyCtrlC {
			for _, p := range m.processesByID {
				p.Cancel()
			}
			cmds = append(cmds, tea.Quit)
		} else if key == "j" {
			if !m.processes[m.selectedProcess].showViewport {
				m.processes[m.selectedProcess].SetSelected(false)
				m.selectedProcess++
				m.processes[m.selectedProcess].SetSelected(true)
				m.selectedProcess = m.selectedProcess % len(m.processes)
			}
		} else if key == "k" {
			if !m.processes[m.selectedProcess].showViewport {
				m.processes[m.selectedProcess].SetSelected(false)
				m.selectedProcess--
				m.processes[m.selectedProcess].SetSelected(true)
				if m.selectedProcess < 0 {
					m.selectedProcess = m.selectedProcess + len(m.processes)
				}
			}
		} else if msg.Type == tea.KeyEnter {
			m.processes[m.selectedProcess].showViewport = !m.processes[m.selectedProcess].showViewport
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
