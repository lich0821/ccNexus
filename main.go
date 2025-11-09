package main

import (
	"embed"
	"log"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

func main() {
	app := NewApp(trayIcon)

	// Load configuration to get window size
	configPath, err := config.GetConfigPath()
	if err != nil {
		log.Printf("Warning: Failed to get config path: %v, using defaults", err)
		configPath = "config.json"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("Warning: Failed to load config: %v, using defaults", err)
		cfg = config.DefaultConfig()
	}

	// Get window size from config
	windowWidth, windowHeight := cfg.GetWindowSize()
	// Use defaults if not set or invalid
	if windowWidth <= 0 {
		windowWidth = 1024
	}
	if windowHeight <= 0 {
		windowHeight = 768
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
