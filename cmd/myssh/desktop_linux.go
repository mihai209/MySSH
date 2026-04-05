//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const desktopFileName = "myssh.desktop"

const desktopEntryTemplate = `[Desktop Entry]
Type=Application
Version=1.0
Name=MySSH
Comment=Secure desktop SSH workspace
Exec=%s
Icon=%s
Terminal=false
Categories=Network;Utility;
StartupNotify=true
`

const desktopIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128">
<defs>
<linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
<stop offset="0%%" stop-color="#0d1b2a"/>
<stop offset="100%%" stop-color="#17324a"/>
</linearGradient>
<linearGradient id="accent" x1="0" y1="0" x2="1" y2="1">
<stop offset="0%%" stop-color="#4fb3ff"/>
<stop offset="100%%" stop-color="#7dffca"/>
</linearGradient>
</defs>
<rect x="8" y="8" width="112" height="112" rx="28" fill="url(#bg)"/>
<rect x="14" y="14" width="100" height="100" rx="22" fill="none" stroke="#31587a" stroke-width="2"/>
<path d="M32 42l20 22-20 22" fill="none" stroke="url(#accent)" stroke-width="10" stroke-linecap="round" stroke-linejoin="round"/>
<path d="M65 86h28" fill="none" stroke="#e8f3ff" stroke-width="10" stroke-linecap="round"/>
</svg>
`

func installDesktopEntry() error {
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	executablePath, err = filepath.Abs(executablePath)
	if err != nil {
		return fmt.Errorf("normalize executable path: %w", err)
	}

	applicationsDir, err := resolveApplicationsDir()
	if err != nil {
		return err
	}
	iconsDir, err := resolveIconsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(applicationsDir, 0o755); err != nil {
		return fmt.Errorf("create applications dir: %w", err)
	}
	if err := os.MkdirAll(iconsDir, 0o755); err != nil {
		return fmt.Errorf("create icons dir: %w", err)
	}

	iconPath := filepath.Join(iconsDir, "myssh.svg")
	if err := os.WriteFile(iconPath, []byte(desktopIconSVG), 0o644); err != nil {
		return fmt.Errorf("write desktop icon: %w", err)
	}

	entryPath := filepath.Join(applicationsDir, desktopFileName)
	entry := fmt.Sprintf(desktopEntryTemplate, shellQuote(executablePath), shellQuote(iconPath))
	if err := os.WriteFile(entryPath, []byte(entry), 0o644); err != nil {
		return fmt.Errorf("write desktop entry: %w", err)
	}

	_ = exec.Command("update-desktop-database", applicationsDir).Run()
	return nil
}

func removeDesktopEntry() error {
	applicationsDir, err := resolveApplicationsDir()
	if err != nil {
		return err
	}
	iconsDir, err := resolveIconsDir()
	if err != nil {
		return err
	}

	entryPath := filepath.Join(applicationsDir, desktopFileName)
	iconPath := filepath.Join(iconsDir, "myssh.svg")

	if err := removeIfExists(entryPath); err != nil {
		return fmt.Errorf("remove desktop entry: %w", err)
	}
	if err := removeIfExists(iconPath); err != nil {
		return fmt.Errorf("remove desktop icon: %w", err)
	}

	_ = exec.Command("update-desktop-database", applicationsDir).Run()
	return nil
}

func resolveApplicationsDir() (string, error) {
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome != "" {
		return filepath.Join(dataHome, "applications"), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(homeDir, ".local", "share", "applications"), nil
}

func resolveIconsDir() (string, error) {
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome != "" {
		return filepath.Join(dataHome, "icons", "hicolor", "scalable", "apps"), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(homeDir, ".local", "share", "icons", "hicolor", "scalable", "apps"), nil
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
