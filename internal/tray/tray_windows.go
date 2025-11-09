//go:build windows
// +build windows

package tray

import (
	"github.com/getlantern/systray"
)

var (
	showWindow func()
	hideWindow func()
	quitApp    func()
	mShow      *systray.MenuItem
	mQuit      *systray.MenuItem
	currentLang string
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

	go func() {
		systray.Run(func() {
			onReady(icon)
		}, func() {
			onExit()
		})
	}()
}

func onReady(icon []byte) {
	if len(icon) > 0 {
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
		for {
			select {
			case <-mShow.ClickedCh:
				if showWindow != nil {
					showWindow()
				}
			case <-mQuit.ClickedCh:
				if quitApp != nil {
					quitApp()
				}
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Cleanup if needed
}

func Quit() {
	systray.Quit()
}

// UpdateLanguage updates the tray menu language
func UpdateLanguage(language string) {
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
