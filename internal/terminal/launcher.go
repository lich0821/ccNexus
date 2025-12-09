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
	// Windows terminals
	case "cmd":
		psCmd := fmt.Sprintf(`Start-Process cmd.exe -ArgumentList '/k','cd /d "%s" && claude'`, dir)
		return exec.Command("powershell.exe", "-Command", psCmd)
	case "powershell":
		psCmd := fmt.Sprintf(`Start-Process powershell -ArgumentList '-NoExit','-Command','cd \"%s\"; claude'`, dir)
		return exec.Command("powershell", "-Command", psCmd)
	case "wt":
		return exec.Command("wt.exe", "new-tab", "--startingDirectory", dir, "cmd", "/k", "claude")
	case "gitbash":
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
		return exec.Command(gitBashPath, "--cd="+dir, "-i", "-c", "claude; exec bash")
	// Mac terminals
	case "terminal":
		script1 := `tell application "Terminal" to activate`
		script2 := fmt.Sprintf(`tell application "Terminal" to do script "cd '%s' && claude"`, dir)
		return exec.Command("osascript", "-e", script1, "-e", script2)
	case "iterm2":
		script := fmt.Sprintf(`tell application "iTerm" to create window with default profile command "cd '%s' && claude"`, dir)
		return exec.Command("osascript", "-e", script)
	case "ghostty":
		return exec.Command("ghostty", "-e", "bash", "-c", fmt.Sprintf("cd '%s' && claude; exec $SHELL", dir))
	case "alacritty":
		return exec.Command("alacritty", "--working-directory", dir, "-e", "bash", "-c", "claude; exec bash")
	case "kitty":
		return exec.Command("kitty", "--directory", dir, "bash", "-c", "claude; exec bash")
	case "wezterm":
		return exec.Command("wezterm", "start", "--cwd", dir, "--", "bash", "-c", "claude; exec bash")
	default:
		return nil
	}
}
