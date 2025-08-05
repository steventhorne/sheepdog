package model

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aymanbagabas/go-pty"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/steventhorne/sheepdog/config"
	"github.com/steventhorne/sheepdog/input"
	"github.com/steventhorne/sheepdog/style"
)

type processStatus string

const (
	statusIdle    processStatus = "idle"
	statusRunning processStatus = "running"
	statusReady   processStatus = "ready"
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

func processTick(id uuid.UUID) tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return processMsg{id: id}
	})
}

type process struct {
	id      uuid.UUID
	name    string
	command []string
	autorun bool
	cwd     string

	pty    pty.Pty
	ctx    context.Context
	cancel context.CancelFunc

	status processStatus
	log    []logEntry

	inboxCh  chan logEntry
	statusCh chan processStatus

	selected     bool
	focused      bool
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

func (m *process) String() string {
	return m.name
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

func (m *process) loadViewportFromInbox() {
	m.pullInbox()
	m.pullStatus()

	sb := &strings.Builder{}
	for _, line := range m.log {
		sb.WriteString(line.msg)
		sb.WriteString("\n")
	}

	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(sb.String()))
	if atBottom {
		m.viewport.GotoBottom()
	}
}

func (m *process) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, input.DefaultKeyMap.Enter):
			m.focused = !m.focused
		}
	case processMsg:
		if msg.id != m.id {
			return m, nil
		}

		m.loadViewportFromInbox()

		if len(m.inboxCh) > 0 || len(m.statusCh) > 0 || m.status == statusRunning || m.status == statusReady {
			return m, processTick(m.id)
		}
		return m, nil
	case tea.WindowSizeMsg:
		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width+style.WidthViewportOffset, msg.Height+style.HeightViewportOffset)
			m.viewport.KeyMap.Down.SetEnabled(false)
			m.viewport.KeyMap.Up.SetEnabled(false)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width + style.WidthViewportOffset
			m.viewport.Height = msg.Height + style.HeightViewportOffset
		}
	}

	if m.selected {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *process) View() string {
	return style.StyleDetails.Render(lipgloss.JoinVertical(lipgloss.Center, style.StyleDetailsHeader.Width(m.viewport.Width).Render(strings.Join(m.command, " ")), m.viewport.View()))
}

func (m *process) Run() tea.Cmd {
	if m.status == statusRunning || m.status == statusReady {
		m.inboxCh <- logEntry{
			msg:   fmt.Sprintf("Process %q is already running.", m.name),
			level: logError,
		}
		return nil
	}

	m.ctx, m.cancel = context.WithCancel(context.Background())

	var err error

	// resolve cmd name
	cmdPath, err := exec.LookPath(m.command[0])
	if err != nil {
		m.inboxCh <- logEntry{
			msg:   err.Error(),
			level: logInfo,
		}
		m.statusCh <- statusErrored
		m.loadViewportFromInbox()
	}

	var cmd *Cmd
	if len(m.command) > 1 {
		cmd = NewCommand(m.ctx, cmdPath, m.command[1:]...)
	} else {
		cmd = NewCommand(m.ctx, cmdPath)
	}

	cmd.Env = os.Environ()

	if m.cwd != "" {
		if filepath.IsAbs(m.cwd) {
			cmd.Dir = m.cwd
		} else {
			cwd, err := os.Getwd()
			if err == nil {
				cmd.Dir = filepath.Join(cwd, m.cwd)
			}
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			m.inboxCh <- logEntry{
				msg:   err.Error(),
				level: logError,
			}
			m.statusCh <- statusErrored
			m.loadViewportFromInbox()
		}
		cmd.Dir = cwd
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err = cmd.Start()
	if err != nil {
		m.inboxCh <- logEntry{
			msg:   err.Error(),
			level: logError,
		}
		m.statusCh <- statusErrored
		m.loadViewportFromInbox()
		return nil
	}

	// TODO: make this running if the config has a ready check
	m.status = statusReady

	go streamPipeToChan(stdout, m.inboxCh, logInfo)
	go streamPipeToChan(stderr, m.inboxCh, logError)

	go func() {
		err := cmd.Wait()
		if err != nil {
			m.inboxCh <- logEntry{
				msg:   fmt.Sprintf("%v", err),
				level: logError,
			}
			m.statusCh <- statusErrored
		} else {
			m.inboxCh <- logEntry{
				msg:   "exited with code 0",
				level: logInfo,
			}
			m.statusCh <- statusExited
		}
	}()

	return processTick(m.id)
}

func (m *process) RunPty() tea.Cmd {
	if m.status == statusRunning || m.status == statusReady {
		m.inboxCh <- logEntry{
			msg:   fmt.Sprintf("Process %q is already running.", m.name),
			level: logError,
		}
		return nil
	}

	m.ctx, m.cancel = context.WithCancel(context.Background())

	var err error
	if m.pty == nil {
		m.pty, err = pty.New()
		if err != nil {
			m.inboxCh <- logEntry{
				msg:   err.Error(),
				level: logError,
			}
			m.statusCh <- statusErrored
			m.loadViewportFromInbox()
		}
		m.pty.Resize(10000, 1)
	}

	// resolve cmd name
	cmdPath, err := exec.LookPath(m.command[0])
	if err != nil {
		m.inboxCh <- logEntry{
			msg:   err.Error(),
			level: logInfo,
		}
		m.statusCh <- statusErrored
		m.loadViewportFromInbox()
	}

	var cmd *pty.Cmd
	if len(m.command) > 1 {
		cmd = m.pty.CommandContext(m.ctx, cmdPath, m.command[1:]...)
	} else {
		cmd = m.pty.CommandContext(m.ctx, cmdPath)
	}

	cmd.Env = os.Environ()

	if m.cwd != "" {
		if filepath.IsAbs(m.cwd) {
			cmd.Dir = m.cwd
		} else {
			cwd, err := os.Getwd()
			if err == nil {
				cmd.Dir = filepath.Join(cwd, m.cwd)
			}
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			m.inboxCh <- logEntry{
				msg:   err.Error(),
				level: logError,
			}
			m.statusCh <- statusErrored
			m.loadViewportFromInbox()
		}
		cmd.Dir = cwd
	}

	err = cmd.Start()
	if err != nil {
		m.inboxCh <- logEntry{
			msg:   err.Error(),
			level: logError,
		}
		m.statusCh <- statusErrored
		m.loadViewportFromInbox()
		return nil
	}

	// TODO: make this running if the config has a ready check
	m.status = statusReady

	go streamPipeToChan(m.pty, m.inboxCh, logInfo)

	go func() {
		err := cmd.Wait()
		if err != nil {
			m.inboxCh <- logEntry{
				msg:   fmt.Sprintf("%v", err),
				level: logError,
			}
			m.statusCh <- statusErrored
		} else {
			m.inboxCh <- logEntry{
				msg:   "exited with code 0",
				level: logInfo,
			}
			m.statusCh <- statusExited
		}
	}()

	return processTick(m.id)
}

func (m *process) Kill() tea.Cmd {
	if m.status != statusRunning && m.status != statusReady {
		return nil
	}

	m.Cancel()
	return nil
}

func (m *process) cleanUp() {
	if m.pty != nil {
		m.pty.Close()
	}
}

var ansiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -~]|\x1b\][^\a]*\a|\x1b\][^\x1b]*\x1b\\`)

func stripControlSequencesButKeepSGR(input string) string {
	return ansiSequence.ReplaceAllStringFunc(input, func(seq string) string {
		// Keep SGR sequences like ESC [ 31 m
		if strings.HasSuffix(seq, "m") {
			return seq
		}
		return ""
	})
}

func streamPipeToChan(r io.ReadCloser, ch chan logEntry, level logLevel) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		clean := stripControlSequencesButKeepSGR(line)

		ch <- logEntry{
			msg:   clean,
			level: level,
		}
	}
}
