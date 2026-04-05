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
<stop offset="0%%" stop-color="#08111b"/>
<stop offset="100%%" stop-color="#14283a"/>
</linearGradient>
<linearGradient id="shell" x1="0" y1="0" x2="1" y2="1">
<stop offset="0%%" stop-color="#4fb3ff"/>
<stop offset="100%%" stop-color="#7dffca"/>
</linearGradient>
<linearGradient id="shield" x1="0" y1="0" x2="0" y2="1">
<stop offset="0%%" stop-color="#10263a"/>
<stop offset="100%%" stop-color="#0c1a29"/>
</linearGradient>
</defs>
<rect x="8" y="8" width="112" height="112" rx="28" fill="url(#bg)"/>
<rect x="14" y="14" width="100" height="100" rx="22" fill="none" stroke="#284865" stroke-width="2"/>
<rect x="24" y="28" width="58" height="42" rx="12" fill="#09131e" stroke="#2f5677" stroke-width="2"/>
<path d="M36 40l11 10-11 10" fill="none" stroke="url(#shell)" stroke-width="6" stroke-linecap="round" stroke-linejoin="round"/>
<path d="M53 60h16" fill="none" stroke="#d9f3ff" stroke-width="6" stroke-linecap="round"/>
<path d="M92 33c8 6 14 8 20 9v16c0 18-11 29-20 34-9-5-20-16-20-34V42c6-1 12-3 20-9Z" fill="url(#shield)" stroke="#4fb3ff" stroke-width="2.5" stroke-linejoin="round"/>
<path d="M92 49c-4 0-7 3-7 7v4h14v-4c0-4-3-7-7-7Zm-10 11h20v13c0 4-3 7-7 7h-6c-4 0-7-3-7-7V60Z" fill="#e7f6ff"/>
<circle cx="92" cy="66" r="2.8" fill="#0b1622"/>
<path d="M92 68.5v5" fill="none" stroke="#0b1622" stroke-width="2.4" stroke-linecap="round"/>
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
