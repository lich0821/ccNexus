package openai

import (
	"sync/atomic"
	"time"
)

// StreamMetrics tracks V3 stream processing metrics
type StreamMetrics struct {
	EventsProcessed  uint64
	EventsGenerated  uint64
	ErrorCount       uint64
	TotalDuration    int64 // nanoseconds
	ProcessingCount  uint64
}

var globalMetrics StreamMetrics

// RecordEvent records a processed event
func (m *StreamMetrics) RecordEvent(inputCount, outputCount int, duration time.Duration) {
	atomic.AddUint64(&m.EventsProcessed, uint64(inputCount))
	atomic.AddUint64(&m.EventsGenerated, uint64(outputCount))
	atomic.AddInt64(&m.TotalDuration, int64(duration))
	atomic.AddUint64(&m.ProcessingCount, 1)
}

// RecordError records an error
func (m *StreamMetrics) RecordError() {
	atomic.AddUint64(&m.ErrorCount, 1)
}

// GetMetrics returns current metrics snapshot
func (m *StreamMetrics) GetMetrics() (processed, generated, errors uint64, avgDuration time.Duration) {
	processed = atomic.LoadUint64(&m.EventsProcessed)
	generated = atomic.LoadUint64(&m.EventsGenerated)
	errors = atomic.LoadUint64(&m.ErrorCount)

	totalDur := atomic.LoadInt64(&m.TotalDuration)
	count := atomic.LoadUint64(&m.ProcessingCount)

	if count > 0 {
		avgDuration = time.Duration(totalDur / int64(count))
	}

	return
}

// GetGlobalMetrics returns the global metrics instance
func GetGlobalMetrics() *StreamMetrics {
	return &globalMetrics
}
