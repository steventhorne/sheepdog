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
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/steventhorne/sheepdog/config"
	"github.com/steventhorne/sheepdog/style"
)

type processStatus int

const (
	statusIdle processStatus = iota
	statusExited
	statusReady
	statusRunning
	statusErrored
)

func (s processStatus) String() string {
	switch s {
	case statusIdle:
		return "idle    "
	case statusExited:
		return "exited  "
	case statusReady:
		return "ready   "
	case statusRunning:
		return "starting"
	case statusErrored:
		return "error   "
	default:
		return "null    "
	}
}

type logLevel string

const (
	logInfo  logLevel = "info"
	logError logLevel = "error"
)

// logBufferSize is the number of log lines buffered per process before new
// lines are dropped to keep the reader from blocking.
const logBufferSize = 1024

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
	id          uuid.UUID
	name        string
	command     []string
	autorun     bool
	cwd         string
	readyRegexp *regexp.Regexp

	isGroup           bool
	groupType         string
	children          []*process
	startupChildIndex int

	pty    pty.Pty
	ctx    context.Context
	cancel context.CancelFunc

	status processStatus
	log    []logEntry

	inboxCh  chan logEntry
	statusCh chan processStatus

	isSelected   bool
	isFocused    bool
	isReady      bool
	viewport     viewport.Model
	showViewport bool
}

func (m *process) IsFocused() bool {
	return m.isFocused
}

func (m *process) Cancel() {
	if m.isGroup {
		for _, cp := range m.children {
			cp.Cancel()
		}
	}

	if m.cancel != nil {
		m.cancel()
	}
}

func (m *process) String() string {
	return m.name
}

func newProcess(config config.ProcessConfig) *process {
	p := &process{
		id:          uuid.New(),
		name:        config.Name,
		command:     config.Command,
		autorun:     config.Autorun,
		cwd:         config.Cwd,
		readyRegexp: nil,
		isGroup:     len(config.Children) > 0,
		groupType:   config.GroupType,
		children:    make([]*process, 0, len(config.Children)),
		status:      statusIdle,
		inboxCh:     make(chan logEntry, logBufferSize),
		statusCh:    make(chan processStatus, 10),
		log:         make([]logEntry, 0, 100),
	}

	if config.ReadyRegexp != "" {
		rg, err := regexp.Compile(config.ReadyRegexp)
		if err != nil {
			p.log = append(p.log, logEntry{
				msg:   fmt.Sprintf("process %s has an invalid ready regexp", p.name),
				level: logError,
			})
		} else {
			p.readyRegexp = rg
		}
	}

	return p
}

func (m *process) Init() tea.Cmd {
	if m.autorun {
		return m.Run()
	}

	if m.isGroup {
		for _, cp := range m.children {
			cp.Init()
		}
	}

	return nil
}

func (m *process) GetStatus() processStatus {
	if m.isGroup {
		s := statusIdle
		for _, cp := range m.children {
			cs := cp.GetStatus()
			if cs > s {
				s = cs
			}
		}
		return s
	}
	return m.status
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

	if m.isGroup {
		for _, cp := range m.children {
			_, cmd := cp.Update(msg)
			cmds = append(cmds, cmd)
		}

		if m.groupType == "sequential" && m.startupChildIndex < len(m.children) {
			if m.children[m.startupChildIndex].GetStatus() == statusReady {
				m.startupChildIndex++
				if m.startupChildIndex < len(m.children) {
					cmds = append(cmds, m.children[m.startupChildIndex].Run())
				}
			}
		}
	}

	switch msg := msg.(type) {
	case processMsg:
		if msg.id != m.id {
			return m, tea.Batch(cmds...)
		}

		m.loadViewportFromInbox()

		if len(m.inboxCh) > 0 || len(m.statusCh) > 0 || m.status == statusRunning || m.status == statusReady {
			return m, processTick(m.id)
		}
		return m, tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		if !m.isReady {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width+style.WidthViewportOffset, msg.Height+style.HeightViewportOffset)
			m.viewport.KeyMap.Down.SetEnabled(false)
			m.viewport.KeyMap.Up.SetEnabled(false)
			m.isReady = true
		} else {
			m.viewport.Width = msg.Width + style.WidthViewportOffset
			m.viewport.Height = msg.Height + style.HeightViewportOffset
		}
	}

	if m.isSelected {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *process) View() string {
	if m.isGroup {
		return style.StyleDetails.Render(lipgloss.JoinVertical(lipgloss.Center, style.StyleDetailsHeader.Width(m.viewport.Width).Render(fmt.Sprintf("%s ##  %s", m.GetStatus(), m.name)), m.FocusedView()))
	} else {
		return style.StyleDetails.Render(lipgloss.JoinVertical(lipgloss.Center, style.StyleDetailsHeader.Width(m.viewport.Width).Render(fmt.Sprintf("%s ##  %s", m.GetStatus(), strings.Join(m.command, " "))), m.FocusedView()))
	}
}

func (m *process) FocusedView() string {
	if m.isGroup {

		var sb strings.Builder
		for _, cp := range m.children {
			fmt.Fprintf(&sb, "%s ##  %s\n", cp.GetStatus(), cp.name)
		}
		return lipgloss.NewStyle().Width(m.viewport.Width).Height(m.viewport.Height).Render(sb.String())
	} else {
		return m.viewport.View()
	}
}

func (m *process) Run() tea.Cmd {
	if m.isGroup {
		if m.groupType == "parallel" {
			cmds := make([]tea.Cmd, 0, len(m.children)+1)
			for _, cp := range m.children {
				cmds = append(cmds, cp.Run())
			}
			cmds = append(cmds, processTick(m.id))
			return tea.Batch(cmds...)
		} else {
			m.startupChildIndex = 0
			return tea.Batch(m.children[0].Run(), processTick(m.id))
		}
	}

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

	if m.readyRegexp != nil {
		m.status = statusRunning
	} else {
		m.status = statusReady
	}

	go streamPipeToChan(stdout, m.inboxCh, m.readyRegexp, m.statusCh, logInfo)
	go streamPipeToChan(stderr, m.inboxCh, m.readyRegexp, m.statusCh, logError)

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

	go streamPipeToChan(m.pty, m.inboxCh, m.readyRegexp, m.statusCh, logInfo)

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
	if m.isGroup {
		for _, cp := range m.children {
			cp.Kill()
		}

		return nil
	}

	if m.status != statusRunning && m.status != statusReady {
		return nil
	}

	m.Cancel()
	return nil
}

func (m *process) CleanUp() {
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

func stripControlSequences(input string) string {
	return ansiSequence.ReplaceAllString(input, "")
}

func streamPipeToChan(r io.ReadCloser, ch chan logEntry, readyRegex *regexp.Regexp, statusCh chan processStatus, level logLevel) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		clean := stripControlSequencesButKeepSGR(line)

		if readyRegex != nil {
			superClean := stripControlSequences(line)
			match := readyRegex.Match([]byte(superClean))
			if match {
				statusCh <- statusReady
			}
		}

		entry := logEntry{msg: clean, level: level}
		select {
		case ch <- entry:
		default:
			// Drop the log line if the buffer is full to avoid
			// blocking the reader. This ensures the process stdout
			// is continually drained even when the UI is busy.
		}
	}
}
