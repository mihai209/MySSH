package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	appsvc "myssh/internal/app"
	"myssh/internal/secret"
	"myssh/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	_ = os.Setenv("JSC_SIGNAL_FOR_GC", strconv.Itoa(int(syscall.SIGUSR2)))

	dataDirFlag := flag.String("data-dir", "", "override the app data directory")
	installFlag := flag.Bool("install", false, "install a desktop entry for MySSH")
	removeFlag := flag.Bool("remove", false, "remove the desktop entry for MySSH")
	flag.Parse()

	if *installFlag && *removeFlag {
		log.Fatal("use either --install or --remove, not both")
	}

	if *installFlag {
		if err := installDesktopEntry(); err != nil {
			log.Fatalf("install desktop entry: %v", err)
		}
		fmt.Println("MySSH desktop entry installed.")
		return
	}

	if *removeFlag {
		if err := removeDesktopEntry(); err != nil {
			log.Fatalf("remove desktop entry: %v", err)
		}
		fmt.Println("MySSH desktop entry removed.")
		return
	}

	dataDir, err := resolveDataDir(*dataDirFlag)
	if err != nil {
		log.Fatalf("resolve data dir: %v", err)
	}

	repo := store.NewProfileRepository(dataDir)
	service := appsvc.NewService(repo)
	secretStore, err := secret.NewStore("MySSH")
	if err != nil {
		log.Fatalf("init secure store: %v", err)
	}

	backend := NewApp(service, secretStore, dataDir)

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
