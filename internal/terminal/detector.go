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
	if runtime.GOOS == "windows" {
		return detectWindowsTerminals()
	}
	return []TerminalInfo{}
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
