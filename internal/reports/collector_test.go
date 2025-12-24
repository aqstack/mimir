package reports

import (
	"strings"
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	if c == nil {
		t.Fatal("expected non-nil collector")
	}
	if c.totalRequests != 0 {
		t.Error("expected zero initial requests")
	}
}

func TestRecordRequest(t *testing.T) {
	c := NewCollector()

	// Record a cache hit
	c.RecordRequest(true, 0.98, 5, 500)

	if c.totalRequests != 1 {
		t.Errorf("expected totalRequests=1, got %d", c.totalRequests)
	}
	if c.totalHits != 1 {
		t.Errorf("expected totalHits=1, got %d", c.totalHits)
	}
	if c.totalMisses != 0 {
		t.Errorf("expected totalMisses=0, got %d", c.totalMisses)
	}

	// Record a cache miss
	c.RecordRequest(false, 0, 100, 0)

	if c.totalRequests != 2 {
		t.Errorf("expected totalRequests=2, got %d", c.totalRequests)
	}
	if c.totalMisses != 1 {
		t.Errorf("expected totalMisses=1, got %d", c.totalMisses)
	}
}

func TestGetReport(t *testing.T) {
	c := NewCollector()

	// Record some requests
	c.RecordRequest(true, 0.99, 5, 500)
	c.RecordRequest(true, 0.97, 10, 600)
	c.RecordRequest(false, 0, 150, 0)
	c.RecordRequest(false, 0, 200, 0)

	report := c.GetReport()

	if report.TotalRequests != 4 {
		t.Errorf("expected TotalRequests=4, got %d", report.TotalRequests)
	}
	if report.TotalHits != 2 {
		t.Errorf("expected TotalHits=2, got %d", report.TotalHits)
	}
	if report.TotalMisses != 2 {
		t.Errorf("expected TotalMisses=2, got %d", report.TotalMisses)
	}
	if report.HitRate != 50.0 {
		t.Errorf("expected HitRate=50.0, got %f", report.HitRate)
	}
	// Avg latency = (5+10+150+200)/4 = 91.25
	if report.AvgLatencyMs != 91.25 {
		t.Errorf("expected AvgLatencyMs=91.25, got %f", report.AvgLatencyMs)
	}
	if report.TotalSavingsUSD <= 0 {
		t.Error("expected positive savings for cache hits")
	}
}

func TestLatencyDistribution(t *testing.T) {
	c := NewCollector()

	// Record requests in different latency buckets
	c.RecordRequest(false, 0, 5, 0)    // 0-10ms
	c.RecordRequest(false, 0, 25, 0)   // 10-50ms
	c.RecordRequest(false, 0, 75, 0)   // 50-100ms
	c.RecordRequest(false, 0, 200, 0)  // 100-500ms
	c.RecordRequest(false, 0, 1000, 0) // 500ms+

	report := c.GetReport()

	expected := map[string]int{
		"0-10ms":    1,
		"10-50ms":   1,
		"50-100ms":  1,
		"100-500ms": 1,
		"500ms+":    1,
	}

	for _, bucket := range report.LatencyDistribution {
		if expected[bucket.Bucket] != bucket.Count {
			t.Errorf("bucket %s: expected %d, got %d", bucket.Bucket, expected[bucket.Bucket], bucket.Count)
		}
	}
}

func TestSimilarityDistribution(t *testing.T) {
	c := NewCollector()

	// Record cache hits with different similarities
	c.RecordRequest(true, 1.0, 5, 100)   // 0.99-1.0
	c.RecordRequest(true, 0.98, 5, 100)  // 0.97-0.99
	c.RecordRequest(true, 0.96, 5, 100)  // 0.95-0.97
	c.RecordRequest(true, 0.92, 5, 100)  // 0.90-0.95
	c.RecordRequest(true, 0.85, 5, 100)  // <0.90
	c.RecordRequest(false, 0, 100, 0)    // miss - should not be counted

	report := c.GetReport()

	expected := map[string]int{
		"0.99-1.0":  1,
		"0.97-0.99": 1,
		"0.95-0.97": 1,
		"0.90-0.95": 1,
		"<0.90":     1,
	}

	for _, bucket := range report.SimilarityDistribution {
		if expected[bucket.Bucket] != bucket.Count {
			t.Errorf("bucket %s: expected %d, got %d", bucket.Bucket, expected[bucket.Bucket], bucket.Count)
		}
	}
}

func TestRecentRequests(t *testing.T) {
	c := NewCollector()

	// Record 60 requests
	for i := 0; i < 60; i++ {
		c.RecordRequest(i%2 == 0, 0.95, int64(i), 100)
	}

	report := c.GetReport()

	// Should only return last 50
	if len(report.RecentRequests) != 50 {
		t.Errorf("expected 50 recent requests, got %d", len(report.RecentRequests))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Minute, "30m"},
		{90 * time.Minute, "1h 30m"},
		{25 * time.Hour, "1d 1h 0m"},
		{50 * time.Hour, "2d 2h 0m"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v): expected %q, got %q", tt.duration, tt.expected, result)
		}
	}
}

func TestAppendWithLimit(t *testing.T) {
	slice := make([]DataPoint, 0, 3)

	// Add up to limit
	for i := 0; i < 3; i++ {
		slice = appendWithLimit(slice, DataPoint{Value: float64(i)}, 3)
	}

	if len(slice) != 3 {
		t.Errorf("expected len=3, got %d", len(slice))
	}

	// Add one more - should rotate
	slice = appendWithLimit(slice, DataPoint{Value: 99}, 3)

	if len(slice) != 3 {
		t.Errorf("expected len=3 after limit, got %d", len(slice))
	}
	if slice[2].Value != 99 {
		t.Errorf("expected last value=99, got %f", slice[2].Value)
	}
	if slice[0].Value != 1 {
		t.Errorf("expected first value=1 after rotation, got %f", slice[0].Value)
	}
}

func TestDashboardHTML(t *testing.T) {
	html := DashboardHTML()

	if len(html) == 0 {
		t.Error("expected non-empty dashboard HTML")
	}

	// Check for key elements
	if !strings.Contains(html, "<title>") {
		t.Error("expected HTML to contain title tag")
	}
	if !strings.Contains(html, "chart.js") {
		t.Error("expected HTML to contain Chart.js reference")
	}
	if !strings.Contains(html, "/reports/data") {
		t.Error("expected HTML to fetch from /reports/data")
	}
}
