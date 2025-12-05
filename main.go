package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/lich0821/ccNexus/internal/singleinstance"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIconWindows []byte

//go:embed build/appicon.png
var trayIconOther []byte

func main() {
	// Check for single instance
	mutex, err := singleinstance.CreateMutex("Global\\ccNexus-SingleInstance-Mutex")
	if err != nil {
		// Another instance is already running, try to show it
		log.Printf("Another instance is already running, attempting to show existing window...")
		if singleinstance.FindAndShowExistingWindow("ccNexus") {
			log.Printf("Successfully brought existing window to foreground")
		} else {
			log.Printf("Could not find existing window, but another instance is running")
		}
		os.Exit(0)
	}
	defer mutex.Release()

	// Select appropriate tray icon based on OS
	var trayIcon []byte
	if os.PathSeparator == '\\' {
		// Windows
		trayIcon = trayIconWindows
	} else {
		// macOS, Linux, etc.
		trayIcon = trayIconOther
	}

	app := NewApp(trayIcon)

	// Load window size from SQLite storage
	windowWidth, windowHeight := 1024, 768 // defaults
	homeDir, err := os.UserHomeDir()
	if err == nil {
		dbPath := filepath.Join(homeDir, ".ccNexus", "ccnexus.db")
		if sqliteStorage, err := storage.NewSQLiteStorage(dbPath); err == nil {
			if w, err := sqliteStorage.GetConfig("windowWidth"); err == nil && w != "" {
				if width, err := strconv.Atoi(w); err == nil && width > 0 {
					windowWidth = width
				}
			}
			if h, err := sqliteStorage.GetConfig("windowHeight"); err == nil && h != "" {
				if height, err := strconv.Atoi(h); err == nil && height > 0 {
					windowHeight = height
				}
			}
			sqliteStorage.Close()
		}
	}

	err = wails.Run(&options.App{
		Title:       "ccNexus",
		Width:       windowWidth,
		Height:      windowHeight,
		StartHidden: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
		Frameless:     false,
		Fullscreen:    false,
		MinWidth:      800,
		MinHeight:     600,
		DisableResize: false,
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       false,
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "ccNexus",
				Message: "© 2024 ccNexus\n\nA smart API endpoint rotation proxy for Claude Code",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
