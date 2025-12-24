package cache

import (
	"context"
	"testing"
	"time"

	"github.com/aqstack/kallm/pkg/api"
)

func newTestEntry(embedding []float64, ttl time.Duration) *api.CacheEntry {
	now := time.Now()
	return &api.CacheEntry{
		Request: api.ChatCompletionRequest{
			Model:    "test-model",
			Messages: []api.Message{{Role: "user", Content: "test"}},
		},
		Response: api.ChatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: now.Unix(),
			Model:   "test-model",
			Choices: []api.Choice{{
				Index:        0,
				Message:      api.Message{Role: "assistant", Content: "test response"},
				FinishReason: "stop",
			}},
		},
		Embedding: embedding,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		LastHitAt: now,
	}
}

func TestNewMemoryCache(t *testing.T) {
	t.Run("with nil options uses defaults", func(t *testing.T) {
		cache := NewMemoryCache(nil)
		if cache == nil {
			t.Fatal("expected non-nil cache")
		}
		if cache.opts.MaxSize != 10000 {
			t.Errorf("expected MaxSize=10000, got %d", cache.opts.MaxSize)
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		opts := &Options{
			MaxSize:             100,
			DefaultTTL:          time.Hour,
			CleanupInterval:     time.Minute,
			SimilarityThreshold: 0.9,
		}
		cache := NewMemoryCache(opts)
		if cache.opts.MaxSize != 100 {
			t.Errorf("expected MaxSize=100, got %d", cache.opts.MaxSize)
		}
	})
}

func TestMemoryCacheSetAndGet(t *testing.T) {
	cache := NewMemoryCache(&Options{
		MaxSize:         100,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour, // Don't run cleanup during test
	})
	ctx := context.Background()

	t.Run("set and get exact match", func(t *testing.T) {
		embedding := []float64{1, 0, 0}
		entry := newTestEntry(embedding, time.Hour)

		err := cache.Set(ctx, entry)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		result, similarity, found := cache.Get(ctx, embedding, 0.99)
		if !found {
			t.Fatal("expected to find cached entry")
		}
		if similarity < 0.99 {
			t.Errorf("expected similarity >= 0.99, got %f", similarity)
		}
		if result.Response.ID != entry.Response.ID {
			t.Error("returned entry doesn't match stored entry")
		}
	})

	t.Run("get similar vector", func(t *testing.T) {
		// Clear cache
		cache.Clear(ctx)

		embedding := []float64{1, 0, 0}
		entry := newTestEntry(embedding, time.Hour)
		cache.Set(ctx, entry)

		// Slightly different vector
		queryEmbedding := []float64{0.99, 0.1, 0}
		result, similarity, found := cache.Get(ctx, queryEmbedding, 0.9)
		if !found {
			t.Fatal("expected to find similar cached entry")
		}
		if similarity < 0.9 {
			t.Errorf("expected similarity >= 0.9, got %f", similarity)
		}
		if result == nil {
			t.Error("expected non-nil result")
		}
	})

	t.Run("miss when below threshold", func(t *testing.T) {
		cache.Clear(ctx)

		embedding := []float64{1, 0, 0}
		entry := newTestEntry(embedding, time.Hour)
		cache.Set(ctx, entry)

		// Very different vector
		queryEmbedding := []float64{0, 1, 0}
		_, _, found := cache.Get(ctx, queryEmbedding, 0.9)
		if found {
			t.Error("expected cache miss for dissimilar vector")
		}
	})

	t.Run("expired entries not returned", func(t *testing.T) {
		cache.Clear(ctx)

		embedding := []float64{1, 0, 0}
		entry := newTestEntry(embedding, -time.Hour) // Already expired
		cache.Set(ctx, entry)

		_, _, found := cache.Get(ctx, embedding, 0.9)
		if found {
			t.Error("expected cache miss for expired entry")
		}
	})
}

func TestMemoryCacheStats(t *testing.T) {
	cache := NewMemoryCache(&Options{
		MaxSize:         100,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	ctx := context.Background()

	embedding := []float64{1, 0, 0}
	entry := newTestEntry(embedding, time.Hour)
	cache.Set(ctx, entry)

	// Generate some hits and misses
	cache.Get(ctx, embedding, 0.9)                  // hit
	cache.Get(ctx, embedding, 0.9)                  // hit
	cache.Get(ctx, []float64{0, 1, 0}, 0.9)         // miss
	cache.Get(ctx, []float64{0, 0, 1}, 0.9)         // miss
	cache.Get(ctx, []float64{-1, 0, 0}, 0.9)        // miss

	// Allow async hit stats update
	time.Sleep(10 * time.Millisecond)

	stats := cache.Stats(ctx)

	if stats.TotalEntries != 1 {
		t.Errorf("expected TotalEntries=1, got %d", stats.TotalEntries)
	}
	if stats.TotalHits != 2 {
		t.Errorf("expected TotalHits=2, got %d", stats.TotalHits)
	}
	if stats.TotalMisses != 3 {
		t.Errorf("expected TotalMisses=3, got %d", stats.TotalMisses)
	}
	if stats.HitRate != 0.4 {
		t.Errorf("expected HitRate=0.4, got %f", stats.HitRate)
	}
}

func TestMemoryCacheDelete(t *testing.T) {
	cache := NewMemoryCache(&Options{
		MaxSize:         100,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	ctx := context.Background()

	embedding := []float64{1, 0, 0}
	entry := newTestEntry(embedding, time.Hour)
	cache.Set(ctx, entry)

	if cache.Size(ctx) != 1 {
		t.Fatalf("expected size=1, got %d", cache.Size(ctx))
	}

	err := cache.Delete(ctx, embedding)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if cache.Size(ctx) != 0 {
		t.Errorf("expected size=0 after delete, got %d", cache.Size(ctx))
	}
}

func TestMemoryCacheClear(t *testing.T) {
	cache := NewMemoryCache(&Options{
		MaxSize:         100,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 10; i++ {
		embedding := make([]float64, 3)
		embedding[i%3] = 1.0
		entry := newTestEntry(embedding, time.Hour)
		cache.Set(ctx, entry)
	}

	if cache.Size(ctx) == 0 {
		t.Fatal("expected non-zero size before clear")
	}

	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if cache.Size(ctx) != 0 {
		t.Errorf("expected size=0 after clear, got %d", cache.Size(ctx))
	}

	stats := cache.Stats(ctx)
	if stats.TotalHits != 0 || stats.TotalMisses != 0 {
		t.Error("expected stats to be reset after clear")
	}
}

func TestMemoryCacheEviction(t *testing.T) {
	cache := NewMemoryCache(&Options{
		MaxSize:         3,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	ctx := context.Background()

	// Add entries up to capacity
	embeddings := [][]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}

	for i, emb := range embeddings {
		entry := newTestEntry(emb, time.Hour)
		entry.Response.ID = string(rune('A' + i))
		cache.Set(ctx, entry)
		time.Sleep(10 * time.Millisecond) // Ensure different LastHitAt
	}

	if cache.Size(ctx) != 3 {
		t.Fatalf("expected size=3, got %d", cache.Size(ctx))
	}

	// Add one more, should evict oldest
	newEmb := []float64{1, 1, 0}
	newEntry := newTestEntry(newEmb, time.Hour)
	newEntry.Response.ID = "D"
	cache.Set(ctx, newEntry)

	if cache.Size(ctx) != 3 {
		t.Errorf("expected size=3 after eviction, got %d", cache.Size(ctx))
	}
}

func TestMemoryCacheCleanup(t *testing.T) {
	cache := NewMemoryCache(&Options{
		MaxSize:         100,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	ctx := context.Background()

	// Add mix of expired and valid entries
	validEmb := []float64{1, 0, 0}
	validEntry := newTestEntry(validEmb, time.Hour)
	cache.Set(ctx, validEntry)

	expiredEmb := []float64{0, 1, 0}
	expiredEntry := newTestEntry(expiredEmb, -time.Hour) // Already expired
	cache.Set(ctx, expiredEntry)

	if cache.Size(ctx) != 2 {
		t.Fatalf("expected size=2, got %d", cache.Size(ctx))
	}

	removed := cache.Cleanup(ctx)
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	if cache.Size(ctx) != 1 {
		t.Errorf("expected size=1 after cleanup, got %d", cache.Size(ctx))
	}
}

func TestMemoryCacheUpdateExisting(t *testing.T) {
	cache := NewMemoryCache(&Options{
		MaxSize:         100,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	ctx := context.Background()

	embedding := []float64{1, 0, 0}

	entry1 := newTestEntry(embedding, time.Hour)
	entry1.Response.Choices[0].Message.Content = "first response"
	cache.Set(ctx, entry1)

	entry2 := newTestEntry(embedding, time.Hour)
	entry2.Response.Choices[0].Message.Content = "second response"
	cache.Set(ctx, entry2)

	// Should still be size 1 (updated, not added)
	if cache.Size(ctx) != 1 {
		t.Errorf("expected size=1 after update, got %d", cache.Size(ctx))
	}

	// Should return updated value
	result, _, found := cache.Get(ctx, embedding, 0.99)
	if !found {
		t.Fatal("expected to find entry")
	}
	if result.Response.Choices[0].Message.Content != "second response" {
		t.Error("expected entry to be updated")
	}
}

func BenchmarkMemoryCacheGet(b *testing.B) {
	cache := NewMemoryCache(&Options{
		MaxSize:         10000,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Hour,
	})
	ctx := context.Background()

	// Pre-populate cache with 1000 entries
	for i := 0; i < 1000; i++ {
		embedding := make([]float64, 768)
		for j := range embedding {
			embedding[j] = float64(i*768+j) / 768000.0
		}
		entry := newTestEntry(embedding, time.Hour)
		cache.Set(ctx, entry)
	}

	queryEmb := make([]float64, 768)
	for i := range queryEmb {
		queryEmb[i] = float64(i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, queryEmb, 0.95)
	}
}
