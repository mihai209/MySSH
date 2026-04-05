//go:build linux

package sshclient

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

func startLocalShell() (string, *os.File, func() error, func(cols int, rows int) error, func() error, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	masterFD, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", nil, nil, nil, nil, fmt.Errorf("open ptmx: %w", err)
	}

	master := os.NewFile(uintptr(masterFD), "/dev/ptmx")
	cleanupMaster := func() {
		_ = master.Close()
	}

	if err := unix.IoctlSetPointerInt(masterFD, unix.TIOCSPTLCK, 0); err != nil {
		cleanupMaster()
		return "", nil, nil, nil, nil, fmt.Errorf("unlock pty: %w", err)
	}

	ptyNumber, err := unix.IoctlGetInt(masterFD, unix.TIOCGPTN)
	if err != nil {
		cleanupMaster()
		return "", nil, nil, nil, nil, fmt.Errorf("read pty number: %w", err)
	}

	slavePath := filepath.Join("/dev/pts", strconv.Itoa(ptyNumber))
	slave, err := os.OpenFile(slavePath, os.O_RDWR, 0)
	if err != nil {
		cleanupMaster()
		return "", nil, nil, nil, nil, fmt.Errorf("open pty slave: %w", err)
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    int(slave.Fd()),
	}

	if err := cmd.Start(); err != nil {
		_ = slave.Close()
		cleanupMaster()
		return "", nil, nil, nil, nil, fmt.Errorf("start local shell: %w", err)
	}

	_ = slave.Close()

	waitFn := func() error {
		return cmd.Wait()
	}

	resizeFn := func(cols int, rows int) error {
		return unix.IoctlSetWinsize(masterFD, unix.TIOCSWINSZ, &unix.Winsize{
			Col: uint16(cols),
			Row: uint16(rows),
		})
	}

	closeFn := func() error {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGHUP)
		}
		return master.Close()
	}

	return shell, master, waitFn, resizeFn, closeFn, nil
}
