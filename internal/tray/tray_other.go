// +build !darwin

package tray

// Setup is a no-op on non-darwin platforms
func Setup(icon []byte, showFunc func(), quitFunc func()) {
	// TODO: Implement for Windows/Linux using appropriate libraries
}

func Quit() {
	// Cleanup if needed
}
