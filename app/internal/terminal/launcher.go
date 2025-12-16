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
	// Find the terminal info to get the detected path
	terminals := DetectTerminals()
	var termInfo *TerminalInfo
	for i := range terminals {
		if terminals[i].ID == terminalID {
			termInfo = &terminals[i]
			break
		}
	}
	if termInfo == nil {
		return fmt.Errorf("terminal not found: %s", terminalID)
	}

	cmd := buildLaunchCommand(*termInfo, dir, sessionID)
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

// getExecutablePath returns the executable path for a terminal
// For macOS .app bundles, extracts the executable from Contents/MacOS/
func getExecutablePath(termInfo TerminalInfo) string {
	path := termInfo.Path
	// Check if it's a macOS .app bundle
	if len(path) > 4 && path[len(path)-4:] == ".app" {
		// Extract app name from path (e.g., "/Applications/Ghostty.app" -> "ghostty")
		appName := termInfo.ID
		// Map terminal IDs to their actual executable names
		execNames := map[string]string{
			"ghostty":   "ghostty",
			"alacritty": "alacritty",
			"kitty":     "kitty",
			"wezterm":   "wezterm-gui",
		}
		if execName, ok := execNames[appName]; ok {
			return path + "/Contents/MacOS/" + execName
		}
	}
	return path
}

func buildLaunchCommand(termInfo TerminalInfo, dir, sessionID string) *exec.Cmd {
	claudeCmd := getClaudeCommand(sessionID)

	switch termInfo.ID {
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
		if termInfo.Path == "" {
			return nil
		}
		return exec.Command(termInfo.Path, "--cd="+dir, "-i", "-c", claudeCmd+"; exec bash")
	// Mac terminals
	case "terminal":
		script1 := `tell application "Terminal" to activate`
		script2 := fmt.Sprintf(`tell application "Terminal" to do script "cd '%s' && %s"`, dir, claudeCmd)
		return exec.Command("osascript", "-e", script1, "-e", script2)
	case "iterm2":
		script := fmt.Sprintf(`tell application "iTerm" to create window with default profile command "cd '%s' && %s"`, dir, claudeCmd)
		return exec.Command("osascript", "-e", script)
	case "ghostty":
		execPath := getExecutablePath(termInfo)
		return exec.Command(execPath, "-e", "bash", "-c", fmt.Sprintf("cd '%s' && %s; exec $SHELL", dir, claudeCmd))
	case "alacritty":
		execPath := getExecutablePath(termInfo)
		return exec.Command(execPath, "--working-directory", dir, "-e", "bash", "-c", claudeCmd+"; exec bash")
	case "kitty":
		execPath := getExecutablePath(termInfo)
		return exec.Command(execPath, "--directory", dir, "bash", "-c", claudeCmd+"; exec bash")
	case "wezterm":
		execPath := getExecutablePath(termInfo)
		return exec.Command(execPath, "start", "--cwd", dir, "--", "bash", "-c", claudeCmd+"; exec bash")
	default:
		return nil
	}
}
