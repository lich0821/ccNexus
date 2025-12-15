package terminal

import (
	"fmt"
	"os/exec"
	"runtime"
)

// LaunchTerminal launches a terminal in the specified directory
func LaunchTerminal(terminalID, dir string) error {
	return LaunchTerminalWithSession(terminalID, dir, "")
}

// LaunchTerminalWithSession launches a terminal with optional session resume
func LaunchTerminalWithSession(terminalID, dir, sessionID string) error {
	cmd := buildLaunchCommand(terminalID, dir, sessionID)
	if cmd == nil {
		return fmt.Errorf("unsupported terminal: %s", terminalID)
	}
	return cmd.Start()
}

// getClaudeCommand returns the claude command with optional session resume
// On macOS, prepends npm initialization to handle lazy-loaded Node environments (nvm, fnm, etc.)
func getClaudeCommand(sessionID string) string {
	cmd := "claude"
	if sessionID != "" {
		cmd = fmt.Sprintf("claude -r %s", sessionID)
	}
	if runtime.GOOS == "darwin" {
		return "npm --version >/dev/null 2>&1; " + cmd
	}
	return cmd
}

func buildLaunchCommand(terminalID, dir, sessionID string) *exec.Cmd {
	claudeCmd := getClaudeCommand(sessionID)

	switch terminalID {
	// Windows terminals
	case "cmd":
		psCmd := fmt.Sprintf(`Start-Process cmd.exe -ArgumentList '/k','cd /d "%s" && %s'`, dir, claudeCmd)
		return exec.Command("powershell.exe", "-Command", psCmd)
	case "powershell":
		shell := "powershell"
		if _, err := exec.LookPath("pwsh.exe"); err == nil {
			shell = "pwsh"
		}
		psCmd := fmt.Sprintf(`Start-Process %s -ArgumentList '-NoExit','-Command','cd \"%s\"; %s'`, shell, dir, claudeCmd)
		return exec.Command(shell+".exe", "-Command", psCmd)
	case "wt":
		shell := "powershell.exe"
		if _, err := exec.LookPath("pwsh.exe"); err == nil {
			shell = "pwsh.exe"
		}
		return exec.Command("wt.exe", "-d", dir, shell, "-NoExit", "-Command", claudeCmd)
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
		return exec.Command(gitBashPath, "--cd="+dir, "-i", "-c", claudeCmd+"; exec bash")
	// Mac terminals
	case "terminal":
		script1 := `tell application "Terminal" to activate`
		script2 := fmt.Sprintf(`tell application "Terminal" to do script "cd '%s' && %s"`, dir, claudeCmd)
		return exec.Command("osascript", "-e", script1, "-e", script2)
	case "iterm2":
		script := fmt.Sprintf(`tell application "iTerm" to create window with default profile command "cd '%s' && %s"`, dir, claudeCmd)
		return exec.Command("osascript", "-e", script)
	case "ghostty":
		return exec.Command("ghostty", "-e", "bash", "-c", fmt.Sprintf("cd '%s' && %s; exec $SHELL", dir, claudeCmd))
	case "alacritty":
		return exec.Command("alacritty", "--working-directory", dir, "-e", "bash", "-c", claudeCmd+"; exec bash")
	case "kitty":
		return exec.Command("kitty", "--directory", dir, "bash", "-c", claudeCmd+"; exec bash")
	case "wezterm":
		return exec.Command("wezterm", "start", "--cwd", dir, "--", "bash", "-c", claudeCmd+"; exec bash")
	default:
		return nil
	}
}
