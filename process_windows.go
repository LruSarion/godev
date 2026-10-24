//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const executableSuffix = ".exe"

var (
	kernel        = syscall.NewLazyDLL("kernel32.dll")
	createJob     = kernel.NewProc("CreateJobObjectW")
	assignJob     = kernel.NewProc("AssignProcessToJobObject")
	terminateJob  = kernel.NewProc("TerminateJobObject")
	resumeProcess = syscall.NewLazyDLL("ntdll.dll").NewProc("NtResumeProcess")
)

type ownership struct {
	handle syscall.Handle
	job    syscall.Handle
}

func configure(cmd *exec.Cmd) {
	// Parent handles Ctrl+C; children are terminated through their owned job.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x00000004}
}

func own(cmd *exec.Cmd) (*ownership, error) {
	// Retain a separate process handle until cleanup completes. Its PID cannot
	// be recycled while taskkill is using it, even if cmd.Wait has returned.
	h, err := syscall.OpenProcess(0x0001|0x0100|0x0400|0x0800|0x100000, false, uint32(cmd.Process.Pid))
	if err != nil {
		return nil, err
	}
	job, _, e := createJob.Call(0, 0)
	if job == 0 {
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("create process job: %w", e)
	}
	ok, _, e := assignJob.Call(job, uintptr(h))
	if ok == 0 {
		syscall.CloseHandle(syscall.Handle(job))
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("assign process job: %w", e)
	}
	// Start suspended so no child can escape before job assignment.
	status, _, _ := resumeProcess.Call(uintptr(h))
	if status != 0 {
		terminateJob.Call(job, 1)
		syscall.CloseHandle(syscall.Handle(job))
		syscall.CloseHandle(h)
		return nil, fmt.Errorf("resume process: NTSTATUS 0x%x", status)
	}
	return &ownership{h, syscall.Handle(job)}, nil
}

func (o *ownership) terminate(cmd *exec.Cmd) {
	// Use native taskkill, never a shell or PowerShell. Job termination also
	// covers descendants when the original parent has already exited.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tool := filepath.Join(os.Getenv("SystemRoot"), "System32", "taskkill.exe")
	kill := exec.CommandContext(ctx, tool, "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
	_ = kill.Run()
	ok, _, err := terminateJob.Call(uintptr(o.job), 1)
	if ok == 0 {
		fmt.Fprintln(os.Stderr, "godev: terminate job:", err)
		_ = cmd.Process.Kill()
	}
}

func (o *ownership) release() {
	_ = syscall.CloseHandle(o.job)
	_ = syscall.CloseHandle(o.handle)
}
