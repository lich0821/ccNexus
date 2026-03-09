package storage

import "time"

type Endpoint struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	APIUrl      string    `json:"apiUrl"`
	APIKey      string    `json:"apiKey"`
	AuthMode    string    `json:"authMode"`
	Enabled     bool      `json:"enabled"`
	Transformer string    `json:"transformer"`
	Model       string    `json:"model"`
	Remark      string    `json:"remark"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type EndpointCredential struct {
	ID            int64      `json:"id"`
	EndpointName  string     `json:"endpointName"`
	ProviderType  string     `json:"providerType"`
	AccountID     string     `json:"accountId,omitempty"`
	Email         string     `json:"email,omitempty"`
	AccessToken   string     `json:"accessToken,omitempty"`
	RefreshToken  string     `json:"refreshToken,omitempty"`
	IDToken       string     `json:"idToken,omitempty"`
	LastRefresh   *time.Time `json:"lastRefresh,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	Status        string     `json:"status"`
	Enabled       bool       `json:"enabled"`
	FailureCount  int        `json:"failureCount"`
	CooldownUntil *time.Time `json:"cooldownUntil,omitempty"`
	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`
	LastUsedAt    *time.Time `json:"lastUsedAt,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
	Remark        string     `json:"remark,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type TokenPoolStats struct {
	Total       int `json:"total"`
	Active      int `json:"active"`
	Expiring    int `json:"expiring"`
	Expired     int `json:"expired"`
	Invalid     int `json:"invalid"`
	Cooldown    int `json:"cooldown"`
	Disabled    int `json:"disabled"`
	NeedRefresh int `json:"needRefresh"`
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
	GetEndpointCredentials(endpointName string) ([]EndpointCredential, error)
	GetCredentialByID(id int64) (*EndpointCredential, error)
	SaveEndpointCredential(cred *EndpointCredential) error
	UpdateEndpointCredential(cred *EndpointCredential) error
	DeleteEndpointCredential(endpointName string, id int64) error
	GetTokenPoolStats(endpointName string) (TokenPoolStats, error)
	GetAllTokenPoolStats() (map[string]TokenPoolStats, error)

	// Stats
	RecordDailyStat(stat *DailyStat) error
	GetDailyStats(endpointName, startDate, endDate string) ([]DailyStat, error)
	GetAllStats() (map[string][]DailyStat, error)
	GetTotalStats() (int, map[string]*EndpointStats, error)
	GetEndpointTotalStats(endpointName string) (*EndpointStats, error)
	GetPeriodStatsAggregated(startDate, endDate string) (map[string]*EndpointStats, error)

	// Config
	GetConfig(key string) (string, error)
	SetConfig(key, value string) error

	// Close
	Close() error
}
