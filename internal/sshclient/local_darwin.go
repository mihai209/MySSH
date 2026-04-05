//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package sshclient

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/creack/pty"
)

func startLocalShell() (string, *os.File, func() error, func(cols int, rows int) error, func() error, error) {
	shell, err := resolveLocalShell()
	if err != nil {
		return "", nil, nil, nil, nil, err
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 120, Rows: 40})
	if err != nil {
		return "", nil, nil, nil, nil, fmt.Errorf("start local shell: %w", err)
	}

	waitFn := func() error {
		return cmd.Wait()
	}

	resizeFn := func(cols int, rows int) error {
		return pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	}

	closeFn := func() error {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGHUP)
		}
		return ptmx.Close()
	}

	return shell, ptmx, waitFn, resizeFn, closeFn, nil
}

func resolveLocalShell() (string, error) {
	candidates := []string{}
	if shell := os.Getenv("SHELL"); shell != "" {
		candidates = append(candidates, shell)
	}
	candidates = append(candidates, "fish", "zsh", "bash", "sh")

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if filepath.IsAbs(candidate) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
			continue
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf("no usable local shell found")
}
