package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 8080 {
		t.Errorf("expected Port=8080, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected Host=0.0.0.0, got %s", cfg.Host)
	}
	if cfg.EmbeddingProvider != "ollama" {
		t.Errorf("expected EmbeddingProvider=ollama, got %s", cfg.EmbeddingProvider)
	}
	if cfg.EmbeddingModel != "nomic-embed-text" {
		t.Errorf("expected EmbeddingModel=nomic-embed-text, got %s", cfg.EmbeddingModel)
	}
	if cfg.OllamaBaseURL != "http://localhost:11434" {
		t.Errorf("expected OllamaBaseURL=http://localhost:11434, got %s", cfg.OllamaBaseURL)
	}
	if cfg.SimilarityThreshold != 0.95 {
		t.Errorf("expected SimilarityThreshold=0.95, got %f", cfg.SimilarityThreshold)
	}
	if cfg.CacheTTL != 24*time.Hour {
		t.Errorf("expected CacheTTL=24h, got %v", cfg.CacheTTL)
	}
	if cfg.MaxCacheSize != 10000 {
		t.Errorf("expected MaxCacheSize=10000, got %d", cfg.MaxCacheSize)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Save original env
	origEnv := map[string]string{
		"KALLM_PORT":                 os.Getenv("KALLM_PORT"),
		"KALLM_HOST":                 os.Getenv("KALLM_HOST"),
		"KALLM_EMBEDDING_PROVIDER":   os.Getenv("KALLM_EMBEDDING_PROVIDER"),
		"KALLM_EMBEDDING_MODEL":      os.Getenv("KALLM_EMBEDDING_MODEL"),
		"OLLAMA_BASE_URL":            os.Getenv("OLLAMA_BASE_URL"),
		"KALLM_SIMILARITY_THRESHOLD": os.Getenv("KALLM_SIMILARITY_THRESHOLD"),
		"KALLM_CACHE_TTL":            os.Getenv("KALLM_CACHE_TTL"),
		"KALLM_MAX_CACHE_SIZE":       os.Getenv("KALLM_MAX_CACHE_SIZE"),
		"OPENAI_API_KEY":             os.Getenv("OPENAI_API_KEY"),
	}

	// Restore env after test
	defer func() {
		for k, v := range origEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Clear env
	for k := range origEnv {
		os.Unsetenv(k)
	}

	t.Run("custom values", func(t *testing.T) {
		os.Setenv("KALLM_PORT", "9090")
		os.Setenv("KALLM_HOST", "127.0.0.1")
		os.Setenv("KALLM_EMBEDDING_PROVIDER", "ollama")
		os.Setenv("KALLM_EMBEDDING_MODEL", "all-minilm")
		os.Setenv("OLLAMA_BASE_URL", "http://ollama:11434")
		os.Setenv("KALLM_SIMILARITY_THRESHOLD", "0.90")
		os.Setenv("KALLM_CACHE_TTL", "1h")
		os.Setenv("KALLM_MAX_CACHE_SIZE", "5000")

		cfg := LoadFromEnv()

		if cfg.Port != 9090 {
			t.Errorf("expected Port=9090, got %d", cfg.Port)
		}
		if cfg.Host != "127.0.0.1" {
			t.Errorf("expected Host=127.0.0.1, got %s", cfg.Host)
		}
		if cfg.EmbeddingProvider != "ollama" {
			t.Errorf("expected EmbeddingProvider=ollama, got %s", cfg.EmbeddingProvider)
		}
		if cfg.EmbeddingModel != "all-minilm" {
			t.Errorf("expected EmbeddingModel=all-minilm, got %s", cfg.EmbeddingModel)
		}
		if cfg.OllamaBaseURL != "http://ollama:11434" {
			t.Errorf("expected OllamaBaseURL=http://ollama:11434, got %s", cfg.OllamaBaseURL)
		}
		if cfg.SimilarityThreshold != 0.90 {
			t.Errorf("expected SimilarityThreshold=0.90, got %f", cfg.SimilarityThreshold)
		}
		if cfg.CacheTTL != time.Hour {
			t.Errorf("expected CacheTTL=1h, got %v", cfg.CacheTTL)
		}
		if cfg.MaxCacheSize != 5000 {
			t.Errorf("expected MaxCacheSize=5000, got %d", cfg.MaxCacheSize)
		}
	})

	t.Run("auto-switch to OpenAI when API key provided", func(t *testing.T) {
		// Clear previous env
		for k := range origEnv {
			os.Unsetenv(k)
		}

		os.Setenv("OPENAI_API_KEY", "sk-test-key")

		cfg := LoadFromEnv()

		if cfg.EmbeddingProvider != "openai" {
			t.Errorf("expected EmbeddingProvider=openai when API key set, got %s", cfg.EmbeddingProvider)
		}
		if cfg.EmbeddingModel != "text-embedding-3-small" {
			t.Errorf("expected EmbeddingModel=text-embedding-3-small, got %s", cfg.EmbeddingModel)
		}
	})

	t.Run("explicit provider overrides auto-switch", func(t *testing.T) {
		for k := range origEnv {
			os.Unsetenv(k)
		}

		os.Setenv("OPENAI_API_KEY", "sk-test-key")
		os.Setenv("KALLM_EMBEDDING_PROVIDER", "ollama")

		cfg := LoadFromEnv()

		if cfg.EmbeddingProvider != "ollama" {
			t.Errorf("expected explicit provider to be respected, got %s", cfg.EmbeddingProvider)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid ollama config",
			cfg: &Config{
				EmbeddingProvider:   "ollama",
				SimilarityThreshold: 0.95,
				MaxCacheSize:        1000,
			},
			wantErr: false,
		},
		{
			name: "valid openai config",
			cfg: &Config{
				EmbeddingProvider:   "openai",
				OpenAIAPIKey:        "sk-test",
				SimilarityThreshold: 0.95,
				MaxCacheSize:        1000,
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			cfg: &Config{
				EmbeddingProvider:   "invalid",
				SimilarityThreshold: 0.95,
				MaxCacheSize:        1000,
			},
			wantErr: true,
			errMsg:  "KALLM_EMBEDDING_PROVIDER",
		},
		{
			name: "openai without api key",
			cfg: &Config{
				EmbeddingProvider:   "openai",
				OpenAIAPIKey:        "",
				SimilarityThreshold: 0.95,
				MaxCacheSize:        1000,
			},
			wantErr: true,
			errMsg:  "OPENAI_API_KEY",
		},
		{
			name: "similarity threshold too high",
			cfg: &Config{
				EmbeddingProvider:   "ollama",
				SimilarityThreshold: 1.5,
				MaxCacheSize:        1000,
			},
			wantErr: true,
			errMsg:  "KALLM_SIMILARITY_THRESHOLD",
		},
		{
			name: "similarity threshold negative",
			cfg: &Config{
				EmbeddingProvider:   "ollama",
				SimilarityThreshold: -0.1,
				MaxCacheSize:        1000,
			},
			wantErr: true,
			errMsg:  "KALLM_SIMILARITY_THRESHOLD",
		},
		{
			name: "max cache size zero",
			cfg: &Config{
				EmbeddingProvider:   "ollama",
				SimilarityThreshold: 0.95,
				MaxCacheSize:        0,
			},
			wantErr: true,
			errMsg:  "KALLM_MAX_CACHE_SIZE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" {
					cfgErr, ok := err.(*ConfigError)
					if !ok {
						t.Errorf("expected *ConfigError, got %T", err)
					} else if cfgErr.Field != tt.errMsg {
						t.Errorf("expected error field %s, got %s", tt.errMsg, cfgErr.Field)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{Field: "TEST_FIELD", Message: "test message"}
	expected := "config error: TEST_FIELD test message"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
