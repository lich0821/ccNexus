package storage

import (
	"encoding/json"
	"os"
)

type LegacyEndpoint struct {
	Name        string `json:"name"`
	APIUrl      string `json:"apiUrl"`
	APIKey      string `json:"apiKey"`
	Enabled     bool   `json:"enabled"`
	Transformer string `json:"transformer,omitempty"`
	Model       string `json:"model,omitempty"`
	Remark      string `json:"remark,omitempty"`
}

type LegacyWebDAVConfig struct {
	URL        string `json:"url"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	ConfigPath string `json:"configPath"`
	StatsPath  string `json:"statsPath"`
}

type LegacyConfig struct {
	Port         int                     `json:"port"`
	Endpoints    []LegacyEndpoint        `json:"endpoints"`
	LogLevel     int                     `json:"logLevel"`
	Language     string                  `json:"language"`
	WindowWidth  int                     `json:"windowWidth"`
	WindowHeight int                     `json:"windowHeight"`
	WebDAV       *LegacyWebDAVConfig     `json:"webdav,omitempty"`
}

type LegacyDailyStats struct {
	Date         string `json:"date"`
	Requests     int    `json:"requests"`
	Errors       int    `json:"errors"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

type LegacyEndpointStats struct {
	Requests     int                            `json:"requests"`
	Errors       int                            `json:"errors"`
	InputTokens  int                            `json:"inputTokens"`
	OutputTokens int                            `json:"outputTokens"`
	DailyHistory map[string]*LegacyDailyStats   `json:"dailyHistory"`
}

type LegacyStats struct {
	TotalRequests int                                `json:"totalRequests"`
	EndpointStats map[string]*LegacyEndpointStats    `json:"endpointStats"`
}

func LoadLegacyConfig(path string) (*LegacyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config LegacyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func LoadLegacyStats(path string) (*LegacyStats, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var stats LegacyStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}
