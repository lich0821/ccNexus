package storage

import "time"

type Endpoint struct {
	ID          int64
	Name        string
	APIUrl      string
	APIKey      string
	Enabled     bool
	Transformer string
	Model       string
	Remark      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DailyStat struct {
	ID           int64
	EndpointName string
	Date         string
	Requests     int
	Errors       int
	InputTokens  int
	OutputTokens int
	DeviceID     string
	CreatedAt    time.Time
}

type EndpointStats struct {
	Requests     int
	Errors       int
	InputTokens  int64
	OutputTokens int64
}

type Storage interface {
	// Endpoints
	GetEndpoints() ([]Endpoint, error)
	SaveEndpoint(ep *Endpoint) error
	UpdateEndpoint(ep *Endpoint) error
	DeleteEndpoint(name string) error

	// Stats
	RecordDailyStat(stat *DailyStat) error
	GetDailyStats(endpointName, startDate, endDate string) ([]DailyStat, error)
	GetAllStats() (map[string][]DailyStat, error)
	GetTotalStats() (int, map[string]*EndpointStats, error)
	GetEndpointTotalStats(endpointName string) (*EndpointStats, error)

	// Config
	GetConfig(key string) (string, error)
	SetConfig(key, value string) error

	// Close
	Close() error
}
