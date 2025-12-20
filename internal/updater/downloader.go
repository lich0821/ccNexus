package updater

import (
	"fmt"
	"io"
	"net"
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
	progress   DownloadProgress
	mu         sync.RWMutex
	cancelChan chan struct{}
	proxyURL   string
}

// NewDownloader creates a new downloader
func NewDownloader() *Downloader {
	return &Downloader{
		progress: DownloadProgress{Status: "idle"},
	}
}

// SetProxyURL sets the proxy URL for downloads
func (d *Downloader) SetProxyURL(proxyURL string) {
	d.mu.Lock()
	d.proxyURL = proxyURL
	d.mu.Unlock()
}

// Download downloads a file from URL to destination
func (d *Downloader) Download(url, destPath string) error {
	d.mu.Lock()
	d.progress = DownloadProgress{
		Status:   "downloading",
		FilePath: destPath,
	}
	d.cancelChan = make(chan struct{})
	proxyURL := d.proxyURL
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

	// Download file with proper timeouts
	client := &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	// Apply proxy if configured
	if proxyURL != "" {
		if transport, err := createProxyTransport(proxyURL); err == nil {
			client.Transport = transport
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		d.setError(fmt.Sprintf("failed to create request: %v", err))
		return err
	}
	req.Header.Set("User-Agent", "ccNexus-Updater")

	resp, err := client.Do(req)
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
		// Check for cancellation
		select {
		case <-d.cancelChan:
			out.Close()
			os.Remove(destPath)
			d.mu.Lock()
			d.progress.Status = "cancelled"
			d.mu.Unlock()
			return fmt.Errorf("download cancelled")
		default:
		}

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

// Cancel cancels the current download
func (d *Downloader) Cancel() {
	d.mu.RLock()
	if d.cancelChan != nil && d.progress.Status == "downloading" {
		close(d.cancelChan)
	}
	d.mu.RUnlock()
}
