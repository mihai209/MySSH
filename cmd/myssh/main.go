package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	appsvc "myssh/internal/app"
	"myssh/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dataDirFlag := flag.String("data-dir", "", "override the app data directory")
	flag.Parse()

	dataDir, err := resolveDataDir(*dataDirFlag)
	if err != nil {
		log.Fatalf("resolve data dir: %v", err)
	}

	repo := store.NewProfileRepository(dataDir)
	service := appsvc.NewService(repo)
	backend := NewApp(service, dataDir)

	if err := wails.Run(&options.App{
		Title:     "MySSH",
		Width:     1360,
		Height:    880,
		OnStartup: backend.startup,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			backend,
		},
	}); err != nil {
		log.Fatalf("run wails: %v", err)
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
