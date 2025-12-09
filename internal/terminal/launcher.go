package terminal

import (
	"fmt"
	"os/exec"
)

// LaunchTerminal launches a terminal in the specified directory
func LaunchTerminal(terminalID, dir string) error {
	cmd := buildLaunchCommand(terminalID, dir)
	if cmd == nil {
		return fmt.Errorf("unsupported terminal: %s", terminalID)
	}
	return cmd.Start()
}

func buildLaunchCommand(terminalID, dir string) *exec.Cmd {
	switch terminalID {
	case "cmd":
		// CMD: 使用 PowerShell 的 Start-Process 启动 CMD 窗口
		psCmd := fmt.Sprintf(`Start-Process cmd.exe -ArgumentList '/k','cd /d "%s" && claude'`, dir)
		return exec.Command("powershell.exe", "-Command", psCmd)
	case "powershell":
		// PowerShell: 使用 Start-Process 在新窗口中启动
		psCmd := fmt.Sprintf(`Start-Process powershell -ArgumentList '-NoExit','-Command','cd \"%s\"; claude'`, dir)
		return exec.Command("powershell", "-Command", psCmd)
	case "wt":
		// Windows Terminal: 使用 new-tab 子命令避免路径解析问题
		return exec.Command("wt.exe", "new-tab", "--startingDirectory", dir, "cmd", "/k", "claude")
	case "gitbash":
		// Find Git Bash path
		terminals := detectWindowsTerminals()
		var gitBashPath string
		for _, t := range terminals {
			if t.ID == "gitbash" {
				gitBashPath = t.Path
				break
			}
		}
		if gitBashPath == "" {
			return nil
		}
		// Git Bash: git-bash.exe 使用 --cd 参数设置目录，-c 执行命令
		return exec.Command(gitBashPath, "--cd="+dir, "-i", "-c", "claude; exec bash")
	default:
		return nil
	}
}
