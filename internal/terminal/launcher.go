package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LaunchTerminal launches a terminal in the specified directory
func LaunchTerminal(terminalID, dir string) error {
	return LaunchTerminalWithSession(terminalID, dir, "")
}

// LaunchTerminalWithSession launches a terminal with optional session resume
func LaunchTerminalWithSession(terminalID, dir, sessionID string) error {
	cliCmd := getClaudeCommand(sessionID, "")
	return launchTerminalWithCli(terminalID, dir, cliCmd)
}

// LaunchTerminalWithCustomCmd launches a terminal with a custom CLI command
func LaunchTerminalWithCustomCmd(terminalID, dir, customCmd string) error {
	return LaunchSessionTerminalWithCustomCmd(terminalID, dir, "", customCmd)
}

// LaunchSessionTerminalWithCustomCmd launches a terminal with a custom CLI command and optional session
func LaunchSessionTerminalWithCustomCmd(terminalID, dir, sessionID, customCmd string) error {
	cliCmd := getClaudeCommand(sessionID, customCmd)
	return launchTerminalWithCli(terminalID, dir, cliCmd)
}

// LaunchCodexTerminal launches a terminal with Codex
func LaunchCodexTerminal(terminalID, dir string) error {
	return LaunchCodexTerminalWithSession(terminalID, dir, "")
}

// LaunchCodexTerminalWithSession launches a terminal with Codex and optional session
func LaunchCodexTerminalWithSession(terminalID, dir, sessionID string) error {
	cliCmd := getCodexCommand(sessionID)
	return launchTerminalWithCli(terminalID, dir, cliCmd)
}

// launchTerminalWithCli is the common implementation for launching terminals
func launchTerminalWithCli(terminalID, dir, cliCmd string) error {
	// Validate directory exists
	if dir != "" {
		if info, err := os.Stat(dir); err != nil {
			return fmt.Errorf("directory does not exist: %s", dir)
		} else if !info.IsDir() {
			return fmt.Errorf("path is not a directory: %s", dir)
		}
	}

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

	cmd := buildLaunchCommandWithCli(*termInfo, dir, cliCmd)
	if cmd == nil {
		return fmt.Errorf("unsupported terminal: %s", terminalID)
	}
	return cmd.Start()
}

// shellType represents the type of shell
type shellType int

const (
	shellBash shellType = iota
	shellZsh
	shellFish
	shellOther
)

// getShellType determines the shell type from the shell path
func getShellType(shell string) shellType {
	base := filepath.Base(shell)
	switch {
	case strings.Contains(base, "zsh"):
		return shellZsh
	case strings.Contains(base, "fish"):
		return shellFish
	case strings.Contains(base, "bash"):
		return shellBash
	default:
		return shellOther
	}
}

// getClaudeCommand returns the claude command with optional session resume
// On macOS, prepends npm initialization to handle lazy-loaded Node environments (nvm, fnm, etc.)
func getClaudeCommand(sessionID, customCmd string) string {
	base := "claude"
	if customCmd != "" {
		base = customCmd
	}
	cmd := base
	if sessionID != "" {
		cmd = fmt.Sprintf("%s -r %s", base, shellEscape(sessionID))
	}
	if runtime.GOOS == "darwin" {
		// Trigger npm lazy-loading for nvm/fnm environments
		return "npm --version >/dev/null 2>&1; " + cmd
	}
	return cmd
}

// getCodexCommand returns the codex command with optional session resume
func getCodexCommand(sessionID string) string {
	cmd := "codex"
	if sessionID != "" {
		cmd = fmt.Sprintf("codex resume %s", shellEscape(sessionID))
	}
	if runtime.GOOS == "darwin" {
		return "npm --version >/dev/null 2>&1; " + cmd
	}
	return cmd
}

// getUserShell returns the user's default shell, validated to exist
func getUserShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = getDefaultShell()
	}
	// Verify shell exists
	if _, err := os.Stat(shell); err != nil {
		return getDefaultShell()
	}
	return shell
}

// getDefaultShell returns the system default shell
func getDefaultShell() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	// Try common shells in order of preference
	shells := []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	for _, sh := range shells {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	return "/bin/sh"
}

// shellEscape escapes a string for safe use in shell commands
// Uses single quotes and escapes embedded single quotes
func shellEscape(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}

// getShellInitCommand returns the command to source shell config files
// This is necessary because shell -c runs in non-interactive mode
// which doesn't load .bashrc/.zshrc where PATH is often configured (e.g., nvm, fnm)
func getShellInitCommand(shell string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	shellT := getShellType(shell)
	switch shellT {
	case shellZsh:
		// zsh: try .zshrc first, then .zprofile
		zshrc := filepath.Join(homeDir, ".zshrc")
		return fmt.Sprintf("[ -f %s ] && source %s", shellEscape(zshrc), shellEscape(zshrc))

	case shellFish:
		// fish uses different syntax and auto-loads config
		// Just ensure config.fish is sourced
		fishConfig := filepath.Join(homeDir, ".config", "fish", "config.fish")
		return fmt.Sprintf("test -f %s; and source %s", shellEscape(fishConfig), shellEscape(fishConfig))

	case shellBash:
		// bash: try .bashrc first (most common for PATH), then .bash_profile
		bashrc := filepath.Join(homeDir, ".bashrc")
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		return fmt.Sprintf("[ -f %s ] && source %s || { [ -f %s ] && source %s; }",
			shellEscape(bashrc), shellEscape(bashrc),
			shellEscape(bashProfile), shellEscape(bashProfile))

	default:
		// Generic POSIX: try .profile
		profile := filepath.Join(homeDir, ".profile")
		return fmt.Sprintf("[ -f %s ] && . %s", shellEscape(profile), shellEscape(profile))
	}
}

// getExecutablePath returns the executable path for a terminal
// For macOS .app bundles, extracts the executable from Contents/MacOS/
func getExecutablePath(termInfo TerminalInfo) string {
	path := termInfo.Path

	// Check if it's a macOS .app bundle
	if !strings.HasSuffix(path, ".app") {
		return path
	}

	// Map terminal IDs to their actual executable names inside .app bundle
	execNames := map[string]string{
		"ghostty":   "ghostty",
		"alacritty": "alacritty",
		"kitty":     "kitty",
		"wezterm":   "wezterm-gui",
	}

	if execName, ok := execNames[termInfo.ID]; ok {
		execPath := filepath.Join(path, "Contents", "MacOS", execName)
		// Verify executable exists
		if _, err := os.Stat(execPath); err == nil {
			return execPath
		}
	}

	return path
}

// getExecShellCommand returns the command to keep the shell running after claude exits
func getExecShellCommand(shell string) string {
	shellT := getShellType(shell)
	if shellT == shellFish {
		return "; exec fish"
	}
	return "; exec $SHELL"
}

// buildThirdPartyTerminalCommand builds the command string for third-party terminals
func buildThirdPartyTerminalCommand(shell, dir, claudeCmd string) string {
	initCmd := getShellInitCommand(shell)
	execShell := getExecShellCommand(shell)
	escapedDir := shellEscape(dir)

	shellT := getShellType(shell)
	if shellT == shellFish {
		// Fish has different syntax
		return fmt.Sprintf("%s; cd %s; and %s%s", initCmd, escapedDir, claudeCmd, execShell)
	}
	return fmt.Sprintf("%s; cd %s && %s%s", initCmd, escapedDir, claudeCmd, execShell)
}

func buildLaunchCommand(termInfo TerminalInfo, dir, sessionID string) *exec.Cmd {
	claudeCmd := getClaudeCommand(sessionID, "")
	return buildLaunchCommandWithCli(termInfo, dir, claudeCmd)
}

func buildLaunchCommandWithCli(termInfo TerminalInfo, dir, cliCmd string) *exec.Cmd {
	switch termInfo.ID {
	// Windows terminals
	case "cmd":
		return buildWindowsCmdCommand(dir, cliCmd)
	case "powershell":
		return buildWindowsPowerShellCommand(dir, cliCmd)
	case "wt":
		return buildWindowsTerminalCommand(dir, cliCmd)
	case "gitbash":
		return buildGitBashCommand(termInfo, dir, cliCmd)

	// Mac terminals
	case "terminal":
		return buildMacTerminalCommand(dir, cliCmd)
	case "iterm2":
		return buildITerm2Command(dir, cliCmd)
	case "ghostty":
		return buildGhosttyCommand(termInfo, dir, cliCmd)
	case "alacritty":
		return buildAlacrittyCommand(termInfo, dir, cliCmd)
	case "kitty":
		return buildKittyCommand(termInfo, dir, cliCmd)
	case "wezterm":
		return buildWezTermCommand(termInfo, dir, cliCmd)

	default:
		return nil
	}
}

// Windows terminal builders

func buildWindowsCmdCommand(dir, claudeCmd string) *exec.Cmd {
	// Use PowerShell to start CMD in a new window
	psCmd := fmt.Sprintf(`Start-Process cmd.exe -ArgumentList '/k','cd /d "%s" && %s'`, dir, claudeCmd)
	return exec.Command("powershell.exe", "-Command", psCmd)
}

func buildWindowsPowerShellCommand(dir, claudeCmd string) *exec.Cmd {
	shell := "powershell"
	if _, err := exec.LookPath("pwsh.exe"); err == nil {
		shell = "pwsh"
	}
	psCmd := fmt.Sprintf(`Start-Process %s -ArgumentList '-NoExit','-Command','cd \"%s\"; %s'`, shell, dir, claudeCmd)
	return exec.Command(shell+".exe", "-Command", psCmd)
}

func buildWindowsTerminalCommand(dir, claudeCmd string) *exec.Cmd {
	shell := "powershell.exe"
	if _, err := exec.LookPath("pwsh.exe"); err == nil {
		shell = "pwsh.exe"
	}
	return exec.Command("wt.exe", "-d", dir, "--", shell, "-NoExit", "-Command", claudeCmd)
}

func buildGitBashCommand(termInfo TerminalInfo, dir, claudeCmd string) *exec.Cmd {
	if termInfo.Path == "" {
		return nil
	}
	return exec.Command(termInfo.Path, "--cd="+dir, "-i", "-c", claudeCmd+"; exec bash")
}

// Mac terminal builders

func buildMacTerminalCommand(dir, claudeCmd string) *exec.Cmd {
	// Use AppleScript for Terminal.app
	escapedDir := strings.ReplaceAll(dir, "'", "'\\''")
	script1 := `tell application "Terminal" to activate`
	script2 := fmt.Sprintf(`tell application "Terminal" to do script "cd '%s' && %s"`, escapedDir, claudeCmd)
	return exec.Command("osascript", "-e", script1, "-e", script2)
}

func buildITerm2Command(dir, claudeCmd string) *exec.Cmd {
	// Use AppleScript for iTerm2
	escapedDir := strings.ReplaceAll(dir, "'", "'\\''")
	script := fmt.Sprintf(`tell application "iTerm" to create window with default profile command "cd '%s' && %s"`, escapedDir, claudeCmd)
	return exec.Command("osascript", "-e", script)
}

func buildGhosttyCommand(termInfo TerminalInfo, dir, claudeCmd string) *exec.Cmd {
	execPath := getExecutablePath(termInfo)
	shell := getUserShell()
	cmdStr := buildThirdPartyTerminalCommand(shell, dir, claudeCmd)

	shellT := getShellType(shell)
	if shellT == shellFish {
		// Fish doesn't support -l in the same way, use -l for login
		return exec.Command(execPath, "-e", shell, "-l", "-c", cmdStr)
	}
	return exec.Command(execPath, "-e", shell, "-l", "-c", cmdStr)
}

func buildAlacrittyCommand(termInfo TerminalInfo, dir, claudeCmd string) *exec.Cmd {
	execPath := getExecutablePath(termInfo)
	shell := getUserShell()
	cmdStr := buildThirdPartyTerminalCommand(shell, dir, claudeCmd)
	return exec.Command(execPath, "--working-directory", dir, "-e", shell, "-l", "-c", cmdStr)
}

func buildKittyCommand(termInfo TerminalInfo, dir, claudeCmd string) *exec.Cmd {
	execPath := getExecutablePath(termInfo)
	shell := getUserShell()
	cmdStr := buildThirdPartyTerminalCommand(shell, dir, claudeCmd)
	return exec.Command(execPath, "--directory", dir, shell, "-l", "-c", cmdStr)
}

func buildWezTermCommand(termInfo TerminalInfo, dir, claudeCmd string) *exec.Cmd {
	execPath := getExecutablePath(termInfo)
	shell := getUserShell()
	cmdStr := buildThirdPartyTerminalCommand(shell, dir, claudeCmd)
	return exec.Command(execPath, "start", "--cwd", dir, "--", shell, "-l", "-c", cmdStr)
}
