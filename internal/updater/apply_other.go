//go:build !windows
// +build !windows

package updater

import "fmt"

// ApplyUpdate 应用更新（非Windows平台不支持自动更新）
func ApplyUpdate(newExePath string) error {
	return fmt.Errorf("auto update only supported on Windows")
}
