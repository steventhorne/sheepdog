//go:build windows

package model

import (
	"os/exec"
	"strconv"
	"syscall"
)

func (c *Cmd) setProcessGroup() {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func (c *Cmd) killProcessTree() error {
	if c.Process == nil {
		return nil
	}

	exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(c.Process.Pid)).Run()
	return nil
}
