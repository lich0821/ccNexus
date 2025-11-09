package tray

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "tray_darwin.h"
*/
import "C"
import (
	"unsafe"
)

var (
	showWindow func()
	hideWindow func()
	quitApp    func()
)

//export goShowWindow
func goShowWindow() {
	if showWindow != nil {
		showWindow()
	}
}

//export goHideWindow
func goHideWindow() {
	if hideWindow != nil {
		hideWindow()
	}
}

//export goQuitApp
func goQuitApp() {
	if quitApp != nil {
		quitApp()
	}
}

// Setup initializes the system tray using native macOS APIs
func Setup(icon []byte, showFunc func(), hideFunc func(), quitFunc func(), language string) {
	showWindow = showFunc
	hideWindow = hideFunc
	quitApp = quitFunc

	if len(icon) > 0 {
		C.setupTray(unsafe.Pointer(&icon[0]), C.int(len(icon)))
	}
}

func Quit() {
	// Cleanup if needed
}

func UpdateLanguage(language string) {
	// TODO: Implement native menu update for macOS
}
