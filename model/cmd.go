package model

import (
	"context"
	"os/exec"
)

type Cmd struct {
	ctx        context.Context
	terminated chan struct{}
	*exec.Cmd
}

func NewCommand(ctx context.Context, command string, args ...string) *Cmd {
	return &Cmd{
		ctx:        ctx,
		terminated: make(chan struct{}),
		Cmd:        exec.Command(command, args...),
	}
}

func (c *Cmd) Start() error {
	c.setProcessGroup()

	err := c.Cmd.Start()
	if err != nil {
		return err
	}
	go func() {
		select {
		case <-c.terminated:
			return
		case <-c.ctx.Done():
		}
		p := c.Process
		if p == nil {
			return
		}
		c.killProcessTree()
	}()
	return nil
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Cmd) Wait() error {
	defer close(c.terminated)
	return c.Cmd.Wait()
}
