package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	githubAPIURL = "https://api.github.com/repos/lich0821/ccNexus/releases/latest"
	httpTimeout  = 30 * time.Second
)

// ReleaseInfo represents GitHub release information
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// GetLatestRelease fetches the latest release from GitHub
func GetLatestRelease() (*ReleaseInfo, error) {
	client := &http.Client{Timeout: httpTimeout}

	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ccNexus-Updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API rate limit exceeded. Please try again later. Details: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}

	return &release, nil
}

// GetAssetForPlatform finds the appropriate asset for current platform
func (r *ReleaseInfo) GetAssetForPlatform() (*Asset, error) {
	pattern := getPlatformPattern()

	for _, asset := range r.Assets {
		if strings.Contains(asset.Name, pattern) {
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("no asset found for platform: %s", pattern)
}

// getPlatformPattern returns the file pattern for current platform
func getPlatformPattern() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "windows":
		if goarch == "arm64" {
			return "windows-arm64.zip"
		}
		return "windows-amd64.zip"
	case "darwin":
		if goarch == "arm64" {
			return "darwin-arm64.zip"
		}
		return "darwin-amd64.zip"
	case "linux":
		return "linux-amd64.tar.gz"
	default:
		return fmt.Sprintf("%s-%s", goos, goarch)
	}
}
