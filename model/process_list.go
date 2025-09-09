package model

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/steventhorne/sheepdog/config"
	"github.com/steventhorne/sheepdog/input"
	"github.com/steventhorne/sheepdog/style"
)

type processList struct {
	processes       []*process
	selectedProcessIndex int
	selectedProcess *process
}

func newProcessList(config config.Config) processList {
	pl := processList{
		processes:     make([]*process, 0),
	}

	for _, pConfig := range config.Processes {
		p := pl.getProcessFromConfig(pConfig, nil)

		pl.processes = append(pl.processes, p)
	}

	if len(pl.processes) > 0 {
		pl.selectedProcessIndex = 0
		pl.selectedProcess = pl.processes[0]
	}

	return pl
}

func (m *processList) getProcessFromConfig(pConfig config.ProcessConfig, parent *process) *process {
	isCommand := len(pConfig.Command) > 0
	isGroup := pConfig.GroupType != "" && len(pConfig.Children) > 0
	if isCommand && isGroup {
		log.Fatalf("Command %s is configured as both a command and a group. This is not allowed", pConfig.Name)
	}

	if !isCommand && !isGroup {
		log.Fatalf("Command %s is configured as neither a command nor a group. This is not allowed", pConfig.Name)
	}

	if isGroup && pConfig.GroupType != "sequential" && pConfig.GroupType != "parallel" {
		log.Fatalf("Invalid group type for group %s", pConfig.Name)
	}

	p := newProcess(pConfig)
	p.isGroup = isGroup

	if parent != nil {
		parent.children = append(parent.children, p)

		// override cwd if we don't explicitly have one and the parent does
		if parent.cwd != "" && p.cwd == "" {
			p.cwd = parent.cwd
		}
	}

	if isGroup {
		for _, cpConfig := range pConfig.Children {
			m.getProcessFromConfig(cpConfig, p)
		}
	}

	return p
}

func (m *processList) GetSelectedProcess() *process {
	return m.selectedProcess
}

func (m *processList) Init() tea.Cmd {
	if len(m.processes) > 0 {
		m.selectedProcess = m.processes[0]
		m.selectedProcess.isSelected = true
	}

	cmds := make([]tea.Cmd, 0, len(m.processes)+1)
	for _, p := range m.processes {
		cmd := p.Init()
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *processList) GetNthProcess(n int, includeHidden bool) *process {
	cur := -1
	for _, p := range m.processes {
		if p == nil {
			continue
		}

		cur++
		if cur == n {
			return p
		}

		if p.isGroup && (includeHidden || p.isFocused) {
			np, nc := m.getNthProcessFromProcess(p, cur, n, includeHidden)
			if np != nil {
				return np
			} else {
				cur = nc
			}
		}
	}

	return nil
}

func (m *processList) getNthProcessFromProcess(p *process, cur int, n int, includeHidden bool) (*process, int) {
	if !p.isGroup {
		return nil, cur
	}

	for _, cp := range p.children {
		if cp == nil {
			continue
		}

		cur++
		if cur == n {
			return cp, cur
		}

		if cp.isGroup && (includeHidden || cp.isFocused) {
			np, nc := m.getNthProcessFromProcess(cp, cur, n, includeHidden)
			if np != nil {
				return np, nc
			} else {
				cur = nc
			}
		}
	}

	return nil, cur
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
		case key.Matches(msg, input.DefaultKeyMap.Enter):
			if m.selectedProcess != nil {
				m.selectedProcess.isFocused = !m.selectedProcess.isFocused
			}
		case key.Matches(msg, input.DefaultKeyMap.Quit):
			for _, p := range m.processes {
				p.Cancel()
			}
		case key.Matches(msg, input.DefaultKeyMap.Down):
			if m.selectedProcess != nil && !m.selectedProcess.isGroup && m.selectedProcess.isFocused {
				break
			}

			tmp := m.selectedProcessIndex + 1
			np := m.GetNthProcess(tmp, false)
			if np != nil {
				m.selectedProcess.isSelected = false
				m.selectedProcess = np
				m.selectedProcess.isSelected = true
				m.selectedProcessIndex = tmp
			}
		case key.Matches(msg, input.DefaultKeyMap.Up):
			if m.selectedProcess != nil && !m.selectedProcess.isGroup && m.selectedProcess.isFocused {
				break
			}

			tmp := m.selectedProcessIndex - 1
			np := m.GetNthProcess(tmp, false)
			if np != nil {
				m.selectedProcess.isSelected = false
				m.selectedProcess = np
				m.selectedProcess.isSelected = true
				m.selectedProcessIndex = tmp
			}
		case key.Matches(msg, input.DefaultKeyMap.Run):
			cmds = append(cmds, m.selectedProcess.Run())
		case key.Matches(msg, input.DefaultKeyMap.Kill):
			cmd := m.selectedProcess.Kill()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func writeListViewForProcess(psb *strings.Builder, p *process, prefix string) {
	var sb strings.Builder

	sb.WriteString(prefix)
	if p.isGroup {
		if p.isFocused {
			sb.WriteString("⯆ ")
		} else {
			sb.WriteString("⯈ ")
		}
	} else {
		sb.WriteString("  ")
	}

	itemStyle := style.StyleItem
	switch p.GetStatus() {
	case statusIdle:
		itemStyle = style.StyleItemIdle
		sb.WriteString("   ")
	case statusRunning:
		itemStyle = style.StyleItemRunning
		sb.WriteString(" P ")
	case statusReady:
		itemStyle = style.StyleItemReady
		sb.WriteString(" R ")
	case statusErrored:
		itemStyle = style.StyleItemErrored
		sb.WriteString(" E ")
	case statusExited:
		sb.WriteString(" X ")
	default:
		sb.WriteString("   ")
	}

	sb.WriteString(p.name)

	if p.isSelected {
		itemStyle = itemStyle.Reverse(true)
	}
	psb.WriteString(itemStyle.Render(sb.String()))
	psb.WriteString("\n")

	if p.isGroup && p.isFocused {
		for _, cp := range p.children {
			writeListViewForProcess(psb, cp, prefix+"| ")
		}
	}
}

func (m *processList) View() string {
	var sb strings.Builder

	for _, p := range m.processes {
		writeListViewForProcess(&sb, p, "")
	}

	return fmt.Sprintf("\n%s\n%s", style.StyleListHeader.Render("Processes"), style.StyleList.Render(sb.String()))
}
