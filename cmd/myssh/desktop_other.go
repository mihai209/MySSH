//go:build !linux

package main

import "fmt"

func installDesktopEntry() error {
	return fmt.Errorf("--install is currently supported only on Linux")
}

func removeDesktopEntry() error {
	return fmt.Errorf("--remove is currently supported only on Linux")
}
