//go:build windows

package model

import (
	"os/exec"
	"strconv"
	"syscall"
)

// createNoWindow (CREATE_NO_WINDOW) gives the child its own hidden console.
// Without it, children attach to sheepdog's console and calls like
// SetConsoleTitle (cmd.exe, batch TITLE, npm shims) retitle our window.
// Not defined in the syscall package, so declared here.
const createNoWindow = 0x08000000

func (c *Cmd) setProcessGroup() {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow}
}

func (c *Cmd) killProcessTree() error {
	if c.Process == nil {
		return nil
	}

	exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(c.Process.Pid)).Run()
	return nil
}
