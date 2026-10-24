//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const executableSuffix = ""

type ownership struct{}

func configure(cmd *exec.Cmd)               { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func own(cmd *exec.Cmd) (*ownership, error) { return &ownership{}, nil }

func (o *ownership) terminate(cmd *exec.Cmd) {
	// Process.Signal serializes against Go's process release. Do not send a
	// negative-PID signal after Wait: the process group ID could be recycled.
	// Portable Unix support intentionally guarantees direct-child cleanup only.
	if err := cmd.Process.Kill(); err != nil && err != os.ErrProcessDone {
		fmt.Fprintln(os.Stderr, "godev: terminate application:", err)
	}
}
func (o *ownership) release() {}
