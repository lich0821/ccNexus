//go:build windows
// +build windows

package tray

import (
	"runtime"

	"github.com/energye/systray"
)

var (
	showWindow  func()
	hideWindow  func()
	quitApp     func()
	mShow       *systray.MenuItem
	mQuit       *systray.MenuItem
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

// Setup initializes the system tray using energye/systray library
func Setup(icon []byte, showFunc func(), hideFunc func(), quitFunc func(), language string) {
	showWindow = showFunc
	hideWindow = hideFunc
	quitApp = quitFunc
	currentLang = language

	// 在独立 goroutine 中运行，并锁定 OS 线程
	// Windows 消息循环必须在创建窗口的同一线程中运行
	go func() {
		runtime.LockOSThread()
		systray.Run(func() {
			onReady(icon)
		}, onExit)
	}()
}

func onReady(icon []byte) {
	if len(icon) > 0 {
		systray.SetIcon(icon)
	}
	systray.SetTitle("ccNexus")

	texts := getMenuTexts(currentLang)
	systray.SetTooltip(texts.Tooltip)

	// 设置双击事件 - 双击托盘图标显示窗口
	systray.SetOnDClick(func(menu systray.IMenu) {
		if showWindow != nil {
			showWindow()
		}
	})

	// 设置右键事件 - 显示菜单（默认行为，可选）
	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})

	mShow = systray.AddMenuItem(texts.Show, texts.ShowTip)
	mShow.Click(func() {
		if showWindow != nil {
			showWindow()
		}
	})

	systray.AddSeparator()

	mQuit = systray.AddMenuItem(texts.Quit, texts.QuitTip)
	mQuit.Click(func() {
		if quitApp != nil {
			quitApp()
		}
		systray.Quit()
	})
}

func onExit() {
	// cleanup if needed
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
