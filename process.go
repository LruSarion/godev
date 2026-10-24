package main

import (
	"os"
	"os/exec"
	"sync"
)

type process struct {
	cmd      *exec.Cmd
	done     chan error
	finished chan struct{}
	owned    *ownership
	once     sync.Once
}

func start(path string, args []string, stdin bool) (*process, error) {
	cmd := exec.Command(path, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if stdin {
		cmd.Stdin = os.Stdin
	}
	configure(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	owned, err := own(cmd)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return nil, err
	}
	p := &process{cmd: cmd, done: make(chan error, 1), finished: make(chan struct{}), owned: owned}
	go func() { p.done <- cmd.Wait(); close(p.finished) }()
	return p, nil
}

func (p *process) stop() {
	p.once.Do(func() {
		p.owned.terminate(p.cmd)
		<-p.finished
		p.owned.release()
	})
}
