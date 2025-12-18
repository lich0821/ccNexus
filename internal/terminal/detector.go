package terminal

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// TerminalInfo represents an available terminal
type TerminalInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// DetectTerminals detects available terminals on the system
func DetectTerminals() []TerminalInfo {
	switch runtime.GOOS {
	case "windows":
		return detectWindowsTerminals()
	case "darwin":
		return detectMacTerminals()
	default:
		return []TerminalInfo{}
	}
}

func detectWindowsTerminals() []TerminalInfo {
	var terminals []TerminalInfo

	// CMD (always available)
	terminals = append(terminals, TerminalInfo{
		ID:   "cmd",
		Name: "CMD",
		Path: "cmd.exe",
	})

	// PowerShell
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		terminals = append(terminals, TerminalInfo{
			ID:   "powershell",
			Name: "PowerShell",
			Path: "powershell.exe",
		})
	}

	// Windows Terminal
	if _, err := exec.LookPath("wt.exe"); err == nil {
		terminals = append(terminals, TerminalInfo{
			ID:   "wt",
			Name: "Windows Terminal",
			Path: "wt.exe",
		})
	}

	// Git Bash - 使用 git-bash.exe 而不是 bash.exe
	gitBashPaths := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "git-bash.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "git-bash.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Git", "git-bash.exe"),
	}
	for _, path := range gitBashPaths {
		if _, err := os.Stat(path); err == nil {
			terminals = append(terminals, TerminalInfo{
				ID:   "gitbash",
				Name: "Git Bash",
				Path: path,
			})
			break
		}
	}

	return terminals
}

func detectMacTerminals() []TerminalInfo {
	var terminals []TerminalInfo

	// Terminal.app - 系统自带，始终可用
	terminals = append(terminals, TerminalInfo{
		ID:   "terminal",
		Name: "Terminal.app",
		Path: "Terminal",
	})

	// iTerm2
	homeDir, _ := os.UserHomeDir()
	iterm2Paths := []string{
		"/Applications/iTerm.app",
		filepath.Join(homeDir, "Applications", "iTerm.app"),
	}
	for _, path := range iterm2Paths {
		if _, err := os.Stat(path); err == nil {
			terminals = append(terminals, TerminalInfo{
				ID:   "iterm2",
				Name: "iTerm2",
				Path: path,
			})
			break
		}
	}

	// Ghostty
	ghosttyPaths := []string{"/Applications/Ghostty.app", filepath.Join(homeDir, "Applications", "Ghostty.app")}
	for _, path := range ghosttyPaths {
		if _, err := os.Stat(path); err == nil {
			terminals = append(terminals, TerminalInfo{ID: "ghostty", Name: "Ghostty", Path: path})
			break
		}
	}
	if len(terminals) == 2 { // 没找到 .app，尝试命令行
		if _, err := exec.LookPath("ghostty"); err == nil {
			terminals = append(terminals, TerminalInfo{ID: "ghostty", Name: "Ghostty", Path: "ghostty"})
		}
	}

	// Alacritty
	alacrittyPaths := []string{"/Applications/Alacritty.app", filepath.Join(homeDir, "Applications", "Alacritty.app")}
	found := false
	for _, path := range alacrittyPaths {
		if _, err := os.Stat(path); err == nil {
			terminals = append(terminals, TerminalInfo{ID: "alacritty", Name: "Alacritty", Path: path})
			found = true
			break
		}
	}
	if !found {
		if path, err := exec.LookPath("alacritty"); err == nil {
			terminals = append(terminals, TerminalInfo{ID: "alacritty", Name: "Alacritty", Path: path})
		}
	}

	// Kitty
	kittyPaths := []string{"/Applications/kitty.app", filepath.Join(homeDir, "Applications", "kitty.app")}
	found = false
	for _, path := range kittyPaths {
		if _, err := os.Stat(path); err == nil {
			terminals = append(terminals, TerminalInfo{ID: "kitty", Name: "Kitty", Path: path})
			found = true
			break
		}
	}
	if !found {
		if path, err := exec.LookPath("kitty"); err == nil {
			terminals = append(terminals, TerminalInfo{ID: "kitty", Name: "Kitty", Path: path})
		}
	}

	// WezTerm
	weztermPaths := []string{"/Applications/WezTerm.app", filepath.Join(homeDir, "Applications", "WezTerm.app")}
	found = false
	for _, path := range weztermPaths {
		if _, err := os.Stat(path); err == nil {
			terminals = append(terminals, TerminalInfo{ID: "wezterm", Name: "WezTerm", Path: path})
			found = true
			break
		}
	}
	if !found {
		if path, err := exec.LookPath("wezterm"); err == nil {
			terminals = append(terminals, TerminalInfo{ID: "wezterm", Name: "WezTerm", Path: path})
		}
	}

	return terminals
}
