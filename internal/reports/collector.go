// Package reports provides performance reporting and visualization for kallm.
package reports

import (
	"fmt"
	"sync"
	"time"
)

// DataPoint represents a single metric data point.
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// RequestMetric represents metrics for a single request.
type RequestMetric struct {
	Timestamp   time.Time `json:"timestamp"`
	CacheHit    bool      `json:"cache_hit"`
	Similarity  float64   `json:"similarity"`
	LatencyMs   int64     `json:"latency_ms"`
	TokensSaved int       `json:"tokens_saved"`
	Prompt      string    `json:"prompt,omitempty"`
}

// LogEntry represents a log entry.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// Collector collects and aggregates performance metrics over time.
type Collector struct {
	mu sync.RWMutex

	// Raw request metrics (ring buffer)
	requests    []RequestMetric
	maxRequests int
	requestIdx  int

	// Log buffer (ring buffer)
	logs    []LogEntry
	maxLogs int

	// Aggregated time-series data (per minute)
	hitRateHistory    []DataPoint
	latencyHistory    []DataPoint
	savingsHistory    []DataPoint
	throughputHistory []DataPoint

	// Current window stats
	windowStart   time.Time
	windowHits    int64
	windowMisses  int64
	windowLatency int64
	windowSavings float64

	// Lifetime stats
	totalRequests  int64
	totalHits      int64
	totalMisses    int64
	totalLatencyMs int64
	totalSavings   float64
	startTime      time.Time
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	now := time.Now()
	return &Collector{
		requests:          make([]RequestMetric, 0, 1000),
		maxRequests:       1000,
		logs:              make([]LogEntry, 0, 100),
		maxLogs:           100,
		hitRateHistory:    make([]DataPoint, 0, 60),   // 1 hour at 1-min resolution
		latencyHistory:    make([]DataPoint, 0, 60),
		savingsHistory:    make([]DataPoint, 0, 60),
		throughputHistory: make([]DataPoint, 0, 60),
		windowStart:       now,
		startTime:         now,
	}
}

// RecordRequest records metrics for a single request.
func (c *Collector) RecordRequest(cacheHit bool, similarity float64, latencyMs int64, tokensSaved int, prompt string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Check if we need to rotate the window (every minute)
	if now.Sub(c.windowStart) >= time.Minute {
		c.rotateWindow(now)
	}

	// Truncate prompt for storage
	if len(prompt) > 100 {
		prompt = prompt[:97] + "..."
	}

	// Record raw metric
	metric := RequestMetric{
		Timestamp:   now,
		CacheHit:    cacheHit,
		Similarity:  similarity,
		LatencyMs:   latencyMs,
		TokensSaved: tokensSaved,
		Prompt:      prompt,
	}

	if len(c.requests) < c.maxRequests {
		c.requests = append(c.requests, metric)
	} else {
		c.requests[c.requestIdx] = metric
		c.requestIdx = (c.requestIdx + 1) % c.maxRequests
	}

	// Update window stats
	if cacheHit {
		c.windowHits++
		c.totalHits++
	} else {
		c.windowMisses++
		c.totalMisses++
	}
	c.windowLatency += latencyMs
	c.totalLatencyMs += latencyMs
	c.totalRequests++

	// Estimate cost savings ($0.002 per 1K tokens for GPT-4)
	if cacheHit && tokensSaved > 0 {
		savings := float64(tokensSaved) * 0.000002
		c.windowSavings += savings
		c.totalSavings += savings
	}
}

// rotateWindow aggregates current window and starts a new one.
func (c *Collector) rotateWindow(now time.Time) {
	total := c.windowHits + c.windowMisses
	if total > 0 {
		hitRate := float64(c.windowHits) / float64(total)
		avgLatency := float64(c.windowLatency) / float64(total)

		c.hitRateHistory = appendWithLimit(c.hitRateHistory, DataPoint{
			Timestamp: c.windowStart,
			Value:     hitRate * 100,
		}, 60)

		c.latencyHistory = appendWithLimit(c.latencyHistory, DataPoint{
			Timestamp: c.windowStart,
			Value:     avgLatency,
		}, 60)

		c.savingsHistory = appendWithLimit(c.savingsHistory, DataPoint{
			Timestamp: c.windowStart,
			Value:     c.windowSavings,
		}, 60)

		c.throughputHistory = appendWithLimit(c.throughputHistory, DataPoint{
			Timestamp: c.windowStart,
			Value:     float64(total),
		}, 60)
	}

	// Reset window
	c.windowStart = now
	c.windowHits = 0
	c.windowMisses = 0
	c.windowLatency = 0
	c.windowSavings = 0
}

func appendWithLimit(slice []DataPoint, point DataPoint, limit int) []DataPoint {
	if len(slice) >= limit {
		copy(slice, slice[1:])
		slice[len(slice)-1] = point
		return slice
	}
	return append(slice, point)
}

// Report represents the full performance report.
type Report struct {
	// Summary stats
	Uptime         string  `json:"uptime"`
	TotalRequests  int64   `json:"total_requests"`
	TotalHits      int64   `json:"total_hits"`
	TotalMisses    int64   `json:"total_misses"`
	HitRate        float64 `json:"hit_rate"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	TotalSavingsUSD float64 `json:"total_savings_usd"`
	RequestsPerMin float64 `json:"requests_per_min"`

	// Time series for charts
	HitRateHistory    []DataPoint `json:"hit_rate_history"`
	LatencyHistory    []DataPoint `json:"latency_history"`
	SavingsHistory    []DataPoint `json:"savings_history"`
	ThroughputHistory []DataPoint `json:"throughput_history"`

	// Recent requests for table
	RecentRequests []RequestMetric `json:"recent_requests"`

	// Distribution data
	LatencyDistribution  []BucketCount `json:"latency_distribution"`
	SimilarityDistribution []BucketCount `json:"similarity_distribution"`
}

// BucketCount represents a histogram bucket.
type BucketCount struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// GetReport generates the current performance report.
func (c *Collector) GetReport() *Report {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	uptime := now.Sub(c.startTime)

	var hitRate, avgLatency, reqPerMin float64
	if c.totalRequests > 0 {
		hitRate = float64(c.totalHits) / float64(c.totalRequests) * 100
		avgLatency = float64(c.totalLatencyMs) / float64(c.totalRequests)
	}
	if uptime.Minutes() > 0 {
		reqPerMin = float64(c.totalRequests) / uptime.Minutes()
	}

	// Get recent requests (last 50)
	recentRequests := make([]RequestMetric, 0, 50)
	for i := len(c.requests) - 1; i >= 0 && len(recentRequests) < 50; i-- {
		recentRequests = append(recentRequests, c.requests[i])
	}

	// Calculate distributions
	latencyDist := c.calculateLatencyDistribution()
	similarityDist := c.calculateSimilarityDistribution()

	return &Report{
		Uptime:               formatDuration(uptime),
		TotalRequests:        c.totalRequests,
		TotalHits:            c.totalHits,
		TotalMisses:          c.totalMisses,
		HitRate:              hitRate,
		AvgLatencyMs:         avgLatency,
		TotalSavingsUSD:      c.totalSavings,
		RequestsPerMin:       reqPerMin,
		HitRateHistory:       c.hitRateHistory,
		LatencyHistory:       c.latencyHistory,
		SavingsHistory:       c.savingsHistory,
		ThroughputHistory:    c.throughputHistory,
		RecentRequests:       recentRequests,
		LatencyDistribution:  latencyDist,
		SimilarityDistribution: similarityDist,
	}
}

func (c *Collector) calculateLatencyDistribution() []BucketCount {
	buckets := map[string]int{
		"0-10ms":   0,
		"10-50ms":  0,
		"50-100ms": 0,
		"100-500ms": 0,
		"500ms+":   0,
	}

	for _, req := range c.requests {
		switch {
		case req.LatencyMs < 10:
			buckets["0-10ms"]++
		case req.LatencyMs < 50:
			buckets["10-50ms"]++
		case req.LatencyMs < 100:
			buckets["50-100ms"]++
		case req.LatencyMs < 500:
			buckets["100-500ms"]++
		default:
			buckets["500ms+"]++
		}
	}

	return []BucketCount{
		{Bucket: "0-10ms", Count: buckets["0-10ms"]},
		{Bucket: "10-50ms", Count: buckets["10-50ms"]},
		{Bucket: "50-100ms", Count: buckets["50-100ms"]},
		{Bucket: "100-500ms", Count: buckets["100-500ms"]},
		{Bucket: "500ms+", Count: buckets["500ms+"]},
	}
}

func (c *Collector) calculateSimilarityDistribution() []BucketCount {
	buckets := map[string]int{
		"0.99-1.0":  0,
		"0.97-0.99": 0,
		"0.95-0.97": 0,
		"0.90-0.95": 0,
		"<0.90":     0,
	}

	for _, req := range c.requests {
		if !req.CacheHit {
			continue
		}
		switch {
		case req.Similarity >= 0.99:
			buckets["0.99-1.0"]++
		case req.Similarity >= 0.97:
			buckets["0.97-0.99"]++
		case req.Similarity >= 0.95:
			buckets["0.95-0.97"]++
		case req.Similarity >= 0.90:
			buckets["0.90-0.95"]++
		default:
			buckets["<0.90"]++
		}
	}

	return []BucketCount{
		{Bucket: "0.99-1.0", Count: buckets["0.99-1.0"]},
		{Bucket: "0.97-0.99", Count: buckets["0.97-0.99"]},
		{Bucket: "0.95-0.97", Count: buckets["0.95-0.97"]},
		{Bucket: "0.90-0.95", Count: buckets["0.90-0.95"]},
		{Bucket: "<0.90", Count: buckets["<0.90"]},
	}
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// AddLog adds a log entry to the buffer.
func (c *Collector) AddLog(level, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	if len(c.logs) >= c.maxLogs {
		c.logs = c.logs[1:]
	}
	c.logs = append(c.logs, entry)
}

// GetLogs returns recent log entries.
func (c *Collector) GetLogs() []LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]LogEntry, len(c.logs))
	copy(result, c.logs)
	return result
}

// ClearLogs clears all log entries.
func (c *Collector) ClearLogs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs = make([]LogEntry, 0, c.maxLogs)
}

