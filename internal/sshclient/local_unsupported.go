//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package sshclient

import (
	"fmt"
	"os"
)

func startLocalShell() (string, *os.File, func() error, func(cols int, rows int) error, func() error, error) {
	return "", nil, nil, nil, nil, fmt.Errorf("local terminal is currently supported on Linux, macOS, and BSD")
}
