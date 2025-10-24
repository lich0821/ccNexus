package proxy

import (
	"sync"
	"time"
)

// EndpointStats represents statistics for a single endpoint
type EndpointStats struct {
	Requests int       `json:"requests"`
	Errors   int       `json:"errors"`
	LastUsed time.Time `json:"lastUsed"`
}

// Stats represents overall proxy statistics
type Stats struct {
	TotalRequests  int                      `json:"totalRequests"`
	EndpointStats  map[string]*EndpointStats `json:"endpointStats"`
	mu             sync.RWMutex
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		EndpointStats: make(map[string]*EndpointStats),
	}
}

// RecordRequest records a request for an endpoint
func (s *Stats) RecordRequest(endpointName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests++

	if _, exists := s.EndpointStats[endpointName]; !exists {
		s.EndpointStats[endpointName] = &EndpointStats{}
	}

	stats := s.EndpointStats[endpointName]
	stats.Requests++
	stats.LastUsed = time.Now()
}

// RecordError records an error for an endpoint
func (s *Stats) RecordError(endpointName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.EndpointStats[endpointName]; !exists {
		s.EndpointStats[endpointName] = &EndpointStats{}
	}

	s.EndpointStats[endpointName].Errors++
}

// GetStats returns a copy of current statistics (thread-safe)
func (s *Stats) GetStats() (int, map[string]*EndpointStats) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy
	statsCopy := make(map[string]*EndpointStats)
	for name, stats := range s.EndpointStats {
		statsCopy[name] = &EndpointStats{
			Requests: stats.Requests,
			Errors:   stats.Errors,
			LastUsed: stats.LastUsed,
		}
	}

	return s.TotalRequests, statsCopy
}

// Reset resets all statistics
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests = 0
	s.EndpointStats = make(map[string]*EndpointStats)
}
