//go:build !windows

package model

import (
	"syscall"
)

func (c *Cmd) setProcessGroup() {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func (c *Cmd) killProcessTree() error {
	if c.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(c.Process.Pid)
	if err != nil {
		return c.Process.Kill()
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}
