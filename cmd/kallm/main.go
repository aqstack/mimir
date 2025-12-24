// kallm - Kubernetes-native LLM Semantic Cache
//
// A drop-in proxy that caches LLM API responses using semantic similarity,
// reducing costs and latency for repeated or similar queries.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aqstack/kallm/internal/cache"
	"github.com/aqstack/kallm/internal/config"
	"github.com/aqstack/kallm/internal/embedding"
	"github.com/aqstack/kallm/internal/logger"
	"github.com/aqstack/kallm/internal/proxy"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse flags
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("kallm %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Load configuration
	cfg := config.LoadFromEnv()

	// Setup logger
	log := logger.New(cfg.LogJSON)

	log.Info("starting kallm",
		"version", version,
		"port", cfg.Port,
		"similarity_threshold", cfg.SimilarityThreshold,
		"cache_ttl", cfg.CacheTTL.String(),
	)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Initialize embedder based on provider
	var embedder embedding.Embedder
	switch cfg.EmbeddingProvider {
	case "ollama":
		embedder = embedding.NewOllamaEmbedder(&embedding.OllamaConfig{
			BaseURL: cfg.OllamaBaseURL,
			Model:   cfg.EmbeddingModel,
		})
		log.Info("initialized Ollama embedder",
			"base_url", cfg.OllamaBaseURL,
			"model", embedder.Model(),
			"dimensions", embedder.Dimensions(),
		)
	case "openai":
		embedder = embedding.NewOpenAIEmbedder(&embedding.OpenAIConfig{
			APIKey:  cfg.OpenAIAPIKey,
			BaseURL: cfg.OpenAIBaseURL,
			Model:   cfg.EmbeddingModel,
		})
		log.Info("initialized OpenAI embedder",
			"model", embedder.Model(),
			"dimensions", embedder.Dimensions(),
		)
	}

	// Initialize cache
	semanticCache := cache.NewMemoryCache(&cache.Options{
		MaxSize:             cfg.MaxCacheSize,
		DefaultTTL:          cfg.CacheTTL,
		CleanupInterval:     5 * time.Minute,
		SimilarityThreshold: cfg.SimilarityThreshold,
	})

	log.Info("initialized cache",
		"max_size", cfg.MaxCacheSize,
		"ttl", cfg.CacheTTL.String(),
	)

	// Create handler
	handler := proxy.NewHandler(cfg, semanticCache, embedder, log)

	// Apply middleware
	var h http.Handler = handler
	h = proxy.CORSMiddleware(h)
	h = proxy.LoggingMiddleware(log)(h)
	h = proxy.RecoveryMiddleware(log)(h)

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      h,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// Print final stats
	stats := semanticCache.Stats(context.Background())
	log.Info("final cache stats",
		"total_entries", stats.TotalEntries,
		"total_hits", stats.TotalHits,
		"total_misses", stats.TotalMisses,
		"hit_rate", fmt.Sprintf("%.2f%%", stats.HitRate*100),
		"estimated_saved_usd", fmt.Sprintf("$%.4f", stats.EstimatedSaved),
	)

	log.Info("server stopped")
}
