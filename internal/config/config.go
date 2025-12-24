// Package config provides configuration management for kallm.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration.
type Config struct {
	// Server settings
	Port    int    `json:"port"`
	Host    string `json:"host"`
	LogJSON bool   `json:"log_json"`

	// Embedding settings
	EmbeddingProvider string `json:"embedding_provider"` // "openai" or "ollama"
	EmbeddingModel    string `json:"embedding_model"`

	// OpenAI settings (when provider is "openai")
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIBaseURL string `json:"openai_base_url"`

	// Ollama settings (when provider is "ollama")
	OllamaBaseURL string `json:"ollama_base_url"`

	// Cache settings
	SimilarityThreshold float64       `json:"similarity_threshold"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	MaxCacheSize        int           `json:"max_cache_size"`

	// Metrics settings
	MetricsEnabled bool `json:"metrics_enabled"`
	MetricsPort    int  `json:"metrics_port"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Port:              8080,
		Host:              "0.0.0.0",
		LogJSON:           false,
		EmbeddingProvider: "ollama", // default to free local embeddings
		EmbeddingModel:    "nomic-embed-text",
		OpenAIAPIKey:      "",
		OpenAIBaseURL:     "https://api.openai.com/v1",
		OllamaBaseURL:     "http://localhost:11434",
		SimilarityThreshold: 0.95,
		CacheTTL:            time.Hour * 24,
		MaxCacheSize:        10000,
		MetricsEnabled:      true,
		MetricsPort:         9090,
	}
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	if port := os.Getenv("KALLM_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if host := os.Getenv("KALLM_HOST"); host != "" {
		cfg.Host = host
	}

	if logJSON := os.Getenv("KALLM_LOG_JSON"); logJSON == "true" {
		cfg.LogJSON = true
	}

	if provider := os.Getenv("KALLM_EMBEDDING_PROVIDER"); provider != "" {
		cfg.EmbeddingProvider = provider
	}

	if model := os.Getenv("KALLM_EMBEDDING_MODEL"); model != "" {
		cfg.EmbeddingModel = model
	}

	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.OpenAIAPIKey = apiKey
		// Auto-switch to OpenAI if API key is provided
		if os.Getenv("KALLM_EMBEDDING_PROVIDER") == "" {
			cfg.EmbeddingProvider = "openai"
			if os.Getenv("KALLM_EMBEDDING_MODEL") == "" {
				cfg.EmbeddingModel = "text-embedding-3-small"
			}
		}
	}

	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		cfg.OpenAIBaseURL = baseURL
	}

	if ollamaURL := os.Getenv("OLLAMA_BASE_URL"); ollamaURL != "" {
		cfg.OllamaBaseURL = ollamaURL
	}

	if threshold := os.Getenv("KALLM_SIMILARITY_THRESHOLD"); threshold != "" {
		if t, err := strconv.ParseFloat(threshold, 64); err == nil {
			cfg.SimilarityThreshold = t
		}
	}

	if ttl := os.Getenv("KALLM_CACHE_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			cfg.CacheTTL = d
		}
	}

	if maxSize := os.Getenv("KALLM_MAX_CACHE_SIZE"); maxSize != "" {
		if s, err := strconv.Atoi(maxSize); err == nil {
			cfg.MaxCacheSize = s
		}
	}

	if metricsEnabled := os.Getenv("KALLM_METRICS_ENABLED"); metricsEnabled == "false" {
		cfg.MetricsEnabled = false
	}

	if metricsPort := os.Getenv("KALLM_METRICS_PORT"); metricsPort != "" {
		if p, err := strconv.Atoi(metricsPort); err == nil {
			cfg.MetricsPort = p
		}
	}

	return cfg
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.EmbeddingProvider != "openai" && c.EmbeddingProvider != "ollama" {
		return &ConfigError{Field: "KALLM_EMBEDDING_PROVIDER", Message: "must be 'openai' or 'ollama'"}
	}
	if c.EmbeddingProvider == "openai" && c.OpenAIAPIKey == "" {
		return &ConfigError{Field: "OPENAI_API_KEY", Message: "required when using OpenAI provider"}
	}
	if c.SimilarityThreshold < 0 || c.SimilarityThreshold > 1 {
		return &ConfigError{Field: "KALLM_SIMILARITY_THRESHOLD", Message: "must be between 0 and 1"}
	}
	if c.MaxCacheSize < 1 {
		return &ConfigError{Field: "KALLM_MAX_CACHE_SIZE", Message: "must be at least 1"}
	}
	return nil
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + " " + e.Message
}
