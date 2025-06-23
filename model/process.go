package model

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/steventhorne/sheepdog/config"
)

var (
	styleIdle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a0a8b7"))
	styleRunning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8ebd6b"))
	styleErrored = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e55561"))
	styleExited = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cc9057"))
)

type processStatus string

const (
	statusIdle    processStatus = "idle"
	statusRunning processStatus = "running"
	statusExited  processStatus = "exited"
	statusErrored processStatus = "errored"
)

type logLevel string

const (
	logInfo  logLevel = "info"
	logError logLevel = "error"
)

type logEntry struct {
	msg   string
	level logLevel
}

type processMsg struct {
	id uuid.UUID
}

func updateProcess(id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		return processMsg{
			id: id,
		}
	}
}

type process struct {
	id      uuid.UUID
	name    string
	command []string
	autorun bool
	cwd     string

	ctx    context.Context
	cancel context.CancelFunc

	status processStatus
	log    []logEntry

	inboxCh  chan logEntry
	statusCh chan processStatus

	selected     bool
	ready        bool
	viewport     viewport.Model
	showViewport bool
}

func (m *process) SetSelected(selected bool) {
	m.selected = selected
}

func (m *process) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
}

func newProcess(config config.ProcessConfig) *process {
	p := &process{
		id:       uuid.New(),
		name:     config.Name,
		command:  config.Command,
		autorun:  config.Autorun,
		cwd:      config.Cwd,
		status:   statusIdle,
		inboxCh:  make(chan logEntry, 64),
		statusCh: make(chan processStatus, 10),
		log:      make([]logEntry, 0, 100),
	}

	return p
}

func (m *process) Init() tea.Cmd {
	if m.autorun {
		return m.Run()
	}

	return nil
}

func (m *process) pullInbox() {
	for {
		select {
		case entry := <-m.inboxCh:
			m.log = append(m.log, entry)
		default:
			return
		}
	}
}

func (m *process) pullStatus() {
	for {
		select {
		case status := <-m.statusCh:
			m.status = status
		default:
			return
		}
	}
}

func (m *process) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case processMsg:
		if msg.id != m.id {
			return m, nil
		}

		m.pullInbox()
		m.pullStatus()

		sb := &strings.Builder{}
		for _, line := range m.log {
			sb.WriteString(line.msg)
			sb.WriteString("\n")
		}

		atBottom := m.viewport.AtBottom()
		m.viewport.SetContent(sb.String())
		if atBottom {
			m.viewport.GotoBottom()
		}

		return m, updateProcess(m.id)
	case tea.WindowSizeMsg:
		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, 20)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = 20
		}
	}

	if m.showViewport {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *process) View() string {
	sb := &strings.Builder{}
	var style lipgloss.Style
	switch m.status {
	case statusIdle:
		style = styleIdle
	case statusExited:
		style = styleExited
	case statusErrored:
		style = styleErrored
	case statusRunning:
		style = styleRunning
	}

	if m.selected {
		style = style.Reverse(true)
	}

	fmt.Fprintf(sb, "%s\n", style.Render(m.name))
	if m.showViewport {
		fmt.Fprintf(sb, "> %s\n\n", strings.Join(m.command, " "))
		sb.WriteString(m.viewport.View())
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m *process) Run() tea.Cmd {
	if m.status == statusRunning {
		m.inboxCh <- logEntry{
			msg:   fmt.Sprintf("Process %q is already running.", m.name),
			level: logError,
		}
		return nil
	}

	m.ctx, m.cancel = context.WithCancel(context.Background())

	var cmd *exec.Cmd
	if len(m.command) > 1 {
		cmd = exec.CommandContext(m.ctx, m.command[0], m.command[1:]...)
	} else {
		cmd = exec.CommandContext(m.ctx, m.command[0])
	}

	if m.cwd != "" {
		if filepath.IsAbs(m.cwd) {
			cmd.Dir = m.cwd
		} else {
			cwd, err := os.Getwd()
			if err == nil {
				cmd.Dir = filepath.Join(cwd, m.cwd)
			}
		}
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		m.statusCh <- statusErrored
		m.inboxCh <- logEntry{
			msg:   err.Error(),
			level: logError,
		}
		return nil
	}

	m.status = statusRunning

	go streamPipeToChan(stdout, m.inboxCh, logInfo)
	go streamPipeToChan(stderr, m.inboxCh, logError)

	go func() {
		err := cmd.Wait()
		m.inboxCh <- logEntry{
			msg:   "exited with code 0",
			level: logError,
		}
		if err != nil {
			m.inboxCh <- logEntry{
				msg:   err.Error(),
				level: logError,
			}
			m.statusCh <- statusErrored
		} else {
			m.statusCh <- statusExited
		}
	}()

	return updateProcess(m.id)
}

func (m *process) Kill() tea.Cmd {
	if m.status != statusRunning {
		return nil
	}

	m.Cancel()
	return nil
}

func streamPipeToChan(r io.ReadCloser, ch chan logEntry, level logLevel) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		ch <- logEntry{
			msg:   line,
			level: level,
		}
	}
}
