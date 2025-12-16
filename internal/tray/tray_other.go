// +build !darwin,!windows

package tray

// Setup is a no-op on non-darwin/non-windows platforms
func Setup(icon []byte, showFunc func(), hideFunc func(), quitFunc func(), language string) {
	// TODO: Implement for Linux using appropriate libraries
}

func Quit() {
	// Cleanup if needed
}

func UpdateLanguage(language string) {
	// No-op on non-supported platforms
}
