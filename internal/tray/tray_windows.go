//go:build windows
// +build windows

package tray

import (
	"time"

	"github.com/getlantern/systray"
	"github.com/lich0821/ccNexus/internal/logger"
)

var (
	showWindow   func()
	hideWindow   func()
	quitApp      func()
	mShow        *systray.MenuItem
	mQuit        *systray.MenuItem
	currentLang  string
	windowOpChan chan func()
	trayIconData []byte
)

// Tray menu texts
var menuTexts = map[string]struct {
	Show    string
	ShowTip string
	Quit    string
	QuitTip string
	Tooltip string
}{
	"zh-CN": {
		Show:    "显示窗口",
		ShowTip: "显示主窗口",
		Quit:    "退出程序",
		QuitTip: "退出 ccNexus",
		Tooltip: "ccNexus - API 端点轮换代理",
	},
	"en": {
		Show:    "Show Window",
		ShowTip: "Show the main window",
		Quit:    "Quit",
		QuitTip: "Quit ccNexus",
		Tooltip: "ccNexus - API Endpoint Rotation Proxy",
	},
}

// Setup initializes the system tray using systray library
func Setup(icon []byte, showFunc func(), hideFunc func(), quitFunc func(), language string) {
	showWindow = showFunc
	hideWindow = hideFunc
	quitApp = quitFunc
	currentLang = language

	// Initialize the window operation channel with buffer size 1
	// This ensures operations are serialized and prevents goroutine accumulation
	windowOpChan = make(chan func(), 1)

	// Start a single worker goroutine to handle all window operations
	go windowOperationWorker()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("System tray setup panic: %v", r)
			}
		}()

		systray.Run(func() {
			onReady(icon)
		}, func() {
			onExit()
		})
	}()
}

// windowOperationWorker processes window operations serially with timeout
func windowOperationWorker() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Window operation worker panic: %v, restarting...", r)
			// Restart the worker if it crashes
			go windowOperationWorker()
		}
	}()

	for op := range windowOpChan {
		if op != nil {
			done := make(chan struct{})
			go func() {
				defer func() {
					recover()
					close(done)
				}()
				op()
			}()

			select {
			case <-done:
			case <-time.After(3 * time.Second):
			}
		}
	}
}

func onReady(icon []byte) {
	if len(icon) > 0 {
		trayIconData = icon
		systray.SetIcon(icon)
	}
	systray.SetTitle("ccNexus")

	// Set initial menu items
	updateMenuTexts(currentLang)

	texts := getMenuTexts(currentLang)
	systray.SetTooltip(texts.Tooltip)

	mShow = systray.AddMenuItem(texts.Show, texts.ShowTip)
	systray.AddSeparator()
	mQuit = systray.AddMenuItem(texts.Quit, texts.QuitTip)

	// Handle menu clicks in a separate goroutine
	go func() {
		// Add panic recovery to prevent the goroutine from crashing
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Tray menu event handler panic recovered: %v", r)
				// Restart the event handler
				go handleMenuEvents()
			}
		}()
		handleMenuEvents()
	}()
}

// handleMenuEvents handles the menu click events
func handleMenuEvents() {
	// Add panic recovery for this handler as well
	defer func() {
		recover()
	}()

	// Refresh tray icon periodically to keep Windows message loop active
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-mShow.ClickedCh:
			if showWindow != nil {
				// Send operation to the worker channel (non-blocking)
				// Use select with default to avoid blocking if channel is full
				select {
				case windowOpChan <- showWindow:
					// Operation queued successfully
				default:
					// Channel is full, skip duplicate request
				}
			}

		case <-mQuit.ClickedCh:
			if quitApp != nil {
				quitApp()
			}
			systray.Quit()
			return

		case <-heartbeat.C:
			if len(trayIconData) > 0 {
				systray.SetIcon(trayIconData)
			}
		}
	}
}

func onExit() {
	// Close the window operation channel
	if windowOpChan != nil {
		close(windowOpChan)
	}
}

func Quit() {
	systray.Quit()
}

// UpdateLanguage updates the tray menu language
func UpdateLanguage(language string) {
	defer func() {
		recover()
	}()

	currentLang = language
	if mShow != nil && mQuit != nil {
		texts := getMenuTexts(language)
		systray.SetTooltip(texts.Tooltip)
		mShow.SetTitle(texts.Show)
		mShow.SetTooltip(texts.ShowTip)
		mQuit.SetTitle(texts.Quit)
		mQuit.SetTooltip(texts.QuitTip)
	}
}

func getMenuTexts(lang string) struct {
	Show    string
	ShowTip string
	Quit    string
	QuitTip string
	Tooltip string
} {
	if texts, ok := menuTexts[lang]; ok {
		return texts
	}
	return menuTexts["en"]
}

func updateMenuTexts(lang string) {
	currentLang = lang
}
