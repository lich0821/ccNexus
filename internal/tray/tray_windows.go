// +build windows

package tray

import (
	"github.com/getlantern/systray"
)

var (
	showWindow func()
	hideWindow func()
	quitApp    func()
)

// Setup initializes the system tray using systray library
func Setup(icon []byte, showFunc func(), hideFunc func(), quitFunc func()) {
	showWindow = showFunc
	hideWindow = hideFunc
	quitApp = quitFunc

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
	systray.SetTooltip("ccNexus - API Proxy")

	mShow := systray.AddMenuItem("Show", "Show window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit ccNexus")

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
