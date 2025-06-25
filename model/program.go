package model

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/steventhorne/sheepdog/config"
	"github.com/steventhorne/sheepdog/input"
)

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
		case key.Matches(msg, input.DefaultKeyMap.Quit):
			cmds = append(cmds, tea.Quit)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.processes.View(), m.processes.GetSelectedProcess().ViewDetails())
}
