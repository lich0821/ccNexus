//go:build windows
// +build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// ApplyUpdate 应用更新：生成更新脚本并执行，实现无感更新
func ApplyUpdate(newExePath string) error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	currentExe, _ = filepath.EvalSymlinks(currentExe)

	// 生成更新脚本
	scriptPath := filepath.Join(filepath.Dir(newExePath), "update.bat")
	script := fmt.Sprintf(`@echo off
chcp 65001 >nul
taskkill /F /PID %d >nul 2>&1
timeout /t 2 /nobreak >nul
copy /y "%s" "%s" >nul
start "" "%s"
del "%%~f0"
`, os.Getpid(), newExePath, currentExe, currentExe)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to create update script: %w", err)
	}

	// 启动更新脚本（静默执行，不显示窗口）
	cmd := exec.Command("cmd", "/c", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start update script: %w", err)
	}

	return nil
}
