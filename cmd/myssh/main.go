package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"myssh/internal/app"
	"myssh/internal/store"
	"myssh/internal/ui"
)

func main() {
	dataDirFlag := flag.String("data-dir", "", "override the app data directory")
	flag.Parse()

	dataDir, err := resolveDataDir(*dataDirFlag)
	if err != nil {
		log.Fatalf("resolve data dir: %v", err)
	}

	repo := store.NewProfileRepository(dataDir)
	service := app.NewService(repo)
	if err := ui.Run(service, dataDir); err != nil {
		log.Fatalf("run ui: %v", err)
	}
}

func resolveDataDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "myssh"), nil
}
