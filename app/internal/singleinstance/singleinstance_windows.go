package singleinstance

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/lich0821/ccNexus/internal/logger"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex  = kernel32.NewProc("CreateMutexW")
	procReleaseMutex = kernel32.NewProc("ReleaseMutex")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
	user32           = syscall.NewLazyDLL("user32.dll")
	procFindWindow   = user32.NewProc("FindWindowW")
	procShowWindow   = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

const (
	ERROR_ALREADY_EXISTS = 183
	SW_RESTORE           = 9
)

// Mutex represents a Windows mutex for single instance checking
type Mutex struct {
	handle syscall.Handle
}

// CreateMutex creates a named mutex for single instance checking
func CreateMutex(name string) (*Mutex, error) {
	mutexName, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("failed to convert mutex name: %w", err)
	}

	ret, _, err := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(mutexName)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("failed to create mutex: %w", err)
	}

	// Check if mutex already exists
	if err.(syscall.Errno) == ERROR_ALREADY_EXISTS {
		// Close the handle since we don't need it
		procCloseHandle.Call(ret)
		return nil, fmt.Errorf("another instance is already running")
	}

	return &Mutex{handle: syscall.Handle(ret)}, nil
}

// Release releases the mutex
func (m *Mutex) Release() error {
	if m.handle != 0 {
		ret, _, err := procReleaseMutex.Call(uintptr(m.handle))
		if ret == 0 {
			return fmt.Errorf("failed to release mutex: %w", err)
		}
		procCloseHandle.Call(uintptr(m.handle))
		m.handle = 0
	}
	return nil
}

// FindAndShowExistingWindow finds and shows the existing application window
func FindAndShowExistingWindow(windowTitle string) bool {
	titlePtr, err := syscall.UTF16PtrFromString(windowTitle)
	if err != nil {
		logger.Warn("Failed to convert window title: %v", err)
		return false
	}

	// Find the window by title
	hwnd, _, _ := procFindWindow.Call(
		0,
		uintptr(unsafe.Pointer(titlePtr)),
	)

	if hwnd == 0 {
		logger.Warn("Could not find existing window")
		return false
	}

	// Restore the window if minimized
	procShowWindow.Call(hwnd, SW_RESTORE)

	// Bring the window to foreground
	procSetForegroundWindow.Call(hwnd)

	logger.Info("Brought existing window to foreground")
	return true
}
