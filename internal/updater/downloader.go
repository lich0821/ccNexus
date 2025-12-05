package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DownloadProgress represents download progress
type DownloadProgress struct {
	Status     string  `json:"status"`
	Progress   float64 `json:"progress"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Speed      int64   `json:"speed"`
	FilePath   string  `json:"filePath"`
	Error      string  `json:"error"`
}

// Downloader handles file downloads
type Downloader struct {
	progress DownloadProgress
	mu       sync.RWMutex
}

// NewDownloader creates a new downloader
func NewDownloader() *Downloader {
	return &Downloader{
		progress: DownloadProgress{Status: "idle"},
	}
}

// Download downloads a file from URL to destination
func (d *Downloader) Download(url, destPath string) error {
	d.mu.Lock()
	d.progress = DownloadProgress{
		Status:   "downloading",
		FilePath: destPath,
	}
	d.mu.Unlock()

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		d.setError(fmt.Sprintf("failed to create directory: %v", err))
		return err
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		d.setError(fmt.Sprintf("failed to create file: %v", err))
		return err
	}
	defer out.Close()

	// Download file
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		d.setError(fmt.Sprintf("failed to download: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		d.setError(fmt.Sprintf("download failed: HTTP %d", resp.StatusCode))
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Update total size
	d.mu.Lock()
	d.progress.Total = resp.ContentLength
	d.mu.Unlock()

	// Download with progress tracking
	startTime := time.Now()
	buffer := make([]byte, 32*1024)
	var downloaded int64

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := out.Write(buffer[:n]); writeErr != nil {
				d.setError(fmt.Sprintf("failed to write file: %v", writeErr))
				return writeErr
			}
			downloaded += int64(n)

			// Update progress
			elapsed := time.Since(startTime).Seconds()
			speed := int64(0)
			if elapsed > 0 {
				speed = int64(float64(downloaded) / elapsed)
			}

			progress := float64(0)
			if resp.ContentLength > 0 {
				progress = float64(downloaded) / float64(resp.ContentLength) * 100
			}

			d.mu.Lock()
			d.progress.Downloaded = downloaded
			d.progress.Progress = progress
			d.progress.Speed = speed
			d.mu.Unlock()
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			d.setError(fmt.Sprintf("download error: %v", err))
			return err
		}
	}

	d.mu.Lock()
	d.progress.Status = "completed"
	d.progress.Progress = 100
	d.mu.Unlock()

	return nil
}

// GetProgress returns current download progress
func (d *Downloader) GetProgress() DownloadProgress {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.progress
}

// setError sets error status
func (d *Downloader) setError(errMsg string) {
	d.mu.Lock()
	d.progress.Status = "failed"
	d.progress.Error = errMsg
	d.mu.Unlock()
}
