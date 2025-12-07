package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
)

// UpdateInfo represents update information
type UpdateInfo struct {
	HasUpdate      bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseDate    string `json:"releaseDate"`
	Changelog      string `json:"changelog"`
	DownloadURL    string `json:"downloadUrl"`
	FileSize       int64  `json:"fileSize"`
}

// InstallResult represents installation result
type InstallResult struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Updater manages application updates
type Updater struct {
	currentVersion string
	downloader     *Downloader
	downloadPath   string
}

// New creates a new updater
func New(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		downloader:     NewDownloader(),
		downloadPath:   getDownloadsDir(),
	}
}

// CheckForUpdates checks if a new version is available
func (u *Updater) CheckForUpdates() (*UpdateInfo, error) {
	release, err := GetLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to check updates: %w", err)
	}

	info := &UpdateInfo{
		CurrentVersion: u.currentVersion,
		LatestVersion:  release.TagName,
		ReleaseDate:    release.PublishedAt.Format("2006-01-02"),
		Changelog:      release.Body,
	}

	hasUpdate, err := IsNewerVersion(u.currentVersion, release.TagName)
	if err != nil {
		return nil, fmt.Errorf("failed to compare versions: %w", err)
	}

	info.HasUpdate = hasUpdate

	if hasUpdate {
		asset, err := release.GetAssetForPlatform()
		if err != nil {
			return nil, fmt.Errorf("failed to find asset: %w", err)
		}
		info.DownloadURL = asset.DownloadURL
		info.FileSize = asset.Size
	}

	return info, nil
}

// DownloadUpdate downloads the update file in background
func (u *Updater) DownloadUpdate(url, filename string) error {
	destPath := filepath.Join(u.downloadPath, filename)
	go u.downloader.Download(url, destPath)
	return nil
}

// GetDownloadProgress returns current download progress
func (u *Updater) GetDownloadProgress() DownloadProgress {
	return u.downloader.GetProgress()
}

// CancelDownload cancels the current download
func (u *Updater) CancelDownload() {
	u.downloader.Cancel()
}

// InstallUpdate installs the downloaded update
func (u *Updater) InstallUpdate(filePath string) (*InstallResult, error) {
	// Extract version from filename (e.g., ccNexus-v3.4.1-darwin-arm64.zip -> v3.4.1)
	filename := filepath.Base(filePath)
	parts := strings.Split(filename, "-")
	version := "latest"
	if len(parts) >= 2 {
		version = parts[1]
	}

	switch runtime.GOOS {
	case "windows":
		return u.installWindows(filePath, version)
	case "darwin":
		return u.installMacOS(filePath, version)
	case "linux":
		return u.installLinux(filePath, version)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// openFileManager opens the file manager to show the specified path
func openFileManager(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", "/select,", filepath.Clean(path))
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "linux":
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// getDownloadsDir returns the Downloads directory path
func getDownloadsDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Downloads")
}

// installWindows installs update on Windows
func (u *Updater) installWindows(filePath, version string) (*InstallResult, error) {
	extractDir := filepath.Join(getDownloadsDir(), fmt.Sprintf("ccNexus-%s", version))

	if err := unzip(filePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract: %w", err)
	}

	exePath := filepath.Join(extractDir, "ccNexus.exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("executable not found in archive")
	}

	openFileManager(exePath)

	return &InstallResult{
		Success: true,
		Path:    extractDir,
		Message: "install_instructions_windows",
	}, nil
}

// installMacOS installs update on macOS
func (u *Updater) installMacOS(filePath, version string) (*InstallResult, error) {
	extractDir := filepath.Join(getDownloadsDir(), fmt.Sprintf("ccNexus-%s", version))

	if err := unzip(filePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract: %w", err)
	}

	appPath := filepath.Join(extractDir, "ccNexus.app")
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("app bundle not found in archive")
	}

	openFileManager(appPath)

	return &InstallResult{
		Success: true,
		Path:    extractDir,
		Message: "install_instructions_macos",
	}, nil
}

// installLinux installs update on Linux
func (u *Updater) installLinux(filePath, version string) (*InstallResult, error) {
	extractDir := filepath.Join(getDownloadsDir(), fmt.Sprintf("ccNexus-%s", version))

	if err := untar(filePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract: %w", err)
	}

	exePath := filepath.Join(extractDir, "ccNexus")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("executable not found in archive")
	}

	if err := os.Chmod(exePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to set executable permission: %w", err)
	}

	openFileManager(exePath)

	return &InstallResult{
		Success: true,
		Path:    extractDir,
		Message: "install_instructions_linux",
	}, nil
}

// unzip extracts a zip archive to destination directory
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// untar extracts a tar.gz archive to destination directory
func untar(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

// CleanupOldDownloads removes old download files
func (u *Updater) CleanupOldDownloads(maxAge time.Duration) error {
	entries, err := os.ReadDir(u.downloadPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			filePath := filepath.Join(u.downloadPath, entry.Name())
			os.Remove(filePath)
		}
	}

	return nil
}

// SendNotification sends a system notification
func SendNotification(title, message string) error {
	if runtime.GOOS == "darwin" {
		script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "default"`, message, title)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()
	}
	return beeep.Notify(title, message, "")
}
