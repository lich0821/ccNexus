//go:build !windows
// +build !windows

package singleinstance

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Mutex represents a Unix file lock for single instance checking
type Mutex struct {
	lockFile *os.File
	lockPath string
}

// CreateMutex creates a file lock for single instance checking
// Uses flock (file locking) which is the standard Unix way to implement single instance
func CreateMutex(name string) (*Mutex, error) {
	// Convert Windows-style mutex name to Unix lock file path
	// "Global\\ccNexus-SingleInstance-Mutex" -> "/tmp/ccNexus-SingleInstance.lock"
	lockName := filepath.Base(name)
	lockPath := filepath.Join(os.TempDir(), lockName+".lock")

	// Try to create and open the lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	// LOCK_EX: exclusive lock
	// LOCK_NB: non-blocking (fail immediately if already locked)
	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("another instance is already running")
	}

	return &Mutex{
		lockFile: lockFile,
		lockPath: lockPath,
	}, nil
}

// Release releases the file lock and removes the lock file
func (m *Mutex) Release() error {
	if m.lockFile != nil {
		// Release the lock
		syscall.Flock(int(m.lockFile.Fd()), syscall.LOCK_UN)

		// Close the file
		m.lockFile.Close()

		// Remove the lock file
		os.Remove(m.lockPath)

		m.lockFile = nil
	}
	return nil
}

// FindAndShowExistingWindow attempts to show the existing application window
// On Unix/macOS, this is not supported due to lack of cross-platform window management APIs
// Returns false to indicate the window could not be activated
func FindAndShowExistingWindow(windowTitle string) bool {
	// Window activation is not supported on Unix/macOS platforms
	// Users will see the "another instance is running" message but the window won't be activated
	return false
}
