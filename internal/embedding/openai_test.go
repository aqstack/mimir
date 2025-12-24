package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aqstack/kallm/pkg/api"
)

func TestNewOpenAIEmbedder(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		embedder := NewOpenAIEmbedder(&OpenAIConfig{
			APIKey: "test-key",
		})

		if embedder.baseURL != "https://api.openai.com/v1" {
			t.Errorf("expected default baseURL, got %s", embedder.baseURL)
		}
		if embedder.model != "text-embedding-3-small" {
			t.Errorf("expected default model, got %s", embedder.model)
		}
		if embedder.dimensions != 1536 {
			t.Errorf("expected dimensions=1536, got %d", embedder.dimensions)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		embedder := NewOpenAIEmbedder(&OpenAIConfig{
			APIKey:  "test-key",
			BaseURL: "https://custom.api.com/v1",
			Model:   "text-embedding-3-large",
			Timeout: time.Minute,
		})

		if embedder.baseURL != "https://custom.api.com/v1" {
			t.Errorf("expected custom baseURL, got %s", embedder.baseURL)
		}
		if embedder.model != "text-embedding-3-large" {
			t.Errorf("expected custom model, got %s", embedder.model)
		}
		if embedder.dimensions != 3072 {
			t.Errorf("expected dimensions=3072 for large model, got %d", embedder.dimensions)
		}
	})

	t.Run("model dimensions mapping", func(t *testing.T) {
		tests := []struct {
			model      string
			dimensions int
		}{
			{"text-embedding-3-small", 1536},
			{"text-embedding-3-large", 3072},
			{"text-embedding-ada-002", 1536},
			{"unknown-model", 1536}, // default
		}

		for _, tt := range tests {
			embedder := NewOpenAIEmbedder(&OpenAIConfig{
				APIKey: "test",
				Model:  tt.model,
			})
			if embedder.dimensions != tt.dimensions {
				t.Errorf("model %s: expected dimensions=%d, got %d", tt.model, tt.dimensions, embedder.dimensions)
			}
		}
	})
}

func TestOpenAIEmbedderEmbed(t *testing.T) {
	t.Run("successful embed", func(t *testing.T) {
		expectedEmbedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/embeddings" {
				t.Errorf("expected /embeddings, got %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("expected Bearer auth header, got %s", r.Header.Get("Authorization"))
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			}

			var req api.EmbeddingRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}

			resp := api.EmbeddingResponse{
				Object: "list",
				Data: []api.EmbeddingData{
					{
						Object:    "embedding",
						Embedding: expectedEmbedding,
						Index:     0,
					},
				},
				Model: "text-embedding-3-small",
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		embedder := NewOpenAIEmbedder(&OpenAIConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
		})

		embedding, err := embedder.Embed(context.Background(), "test text")
		if err != nil {
			t.Fatalf("Embed failed: %v", err)
		}

		if len(embedding) != len(expectedEmbedding) {
			t.Fatalf("expected %d dimensions, got %d", len(expectedEmbedding), len(embedding))
		}
		for i, v := range expectedEmbedding {
			if embedding[i] != v {
				t.Errorf("embedding[%d]: expected %f, got %f", i, v, embedding[i])
			}
		}
	})

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			resp := api.ErrorResponse{
				Error: api.APIError{
					Message: "Invalid API key",
					Type:    "invalid_request_error",
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		embedder := NewOpenAIEmbedder(&OpenAIConfig{
			APIKey:  "invalid-key",
			BaseURL: server.URL,
		})
		_, err := embedder.Embed(context.Background(), "test")
		if err == nil {
			t.Error("expected error on API error")
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		embedder := NewOpenAIEmbedder(&OpenAIConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
		})
		_, err := embedder.Embed(context.Background(), "test")
		if err == nil {
			t.Error("expected error on server error")
		}
	})

	t.Run("empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := api.EmbeddingResponse{
				Object: "list",
				Data:   []api.EmbeddingData{},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		embedder := NewOpenAIEmbedder(&OpenAIConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
		})
		_, err := embedder.Embed(context.Background(), "test")
		if err == nil {
			t.Error("expected error on empty response")
		}
	})
}

func TestOpenAIEmbedderEmbedBatch(t *testing.T) {
	t.Run("successful batch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req api.EmbeddingRequest
			json.NewDecoder(r.Body).Decode(&req)

			// Return embeddings for each input
			inputs := req.Input.([]interface{})
			data := make([]api.EmbeddingData, len(inputs))
			for i := range inputs {
				data[i] = api.EmbeddingData{
					Object:    "embedding",
					Embedding: []float64{float64(i + 1), 0.2, 0.3},
					Index:     i,
				}
			}

			resp := api.EmbeddingResponse{
				Object: "list",
				Data:   data,
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		embedder := NewOpenAIEmbedder(&OpenAIConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
		})

		texts := []string{"text1", "text2", "text3"}
		embeddings, err := embedder.EmbedBatch(context.Background(), texts)
		if err != nil {
			t.Fatalf("EmbedBatch failed: %v", err)
		}

		if len(embeddings) != 3 {
			t.Fatalf("expected 3 embeddings, got %d", len(embeddings))
		}

		for i, emb := range embeddings {
			if emb[0] != float64(i+1) {
				t.Errorf("embedding %d: expected first value %f, got %f", i, float64(i+1), emb[0])
			}
		}
	})

	t.Run("empty input", func(t *testing.T) {
		embedder := NewOpenAIEmbedder(&OpenAIConfig{APIKey: "test"})
		embeddings, err := embedder.EmbedBatch(context.Background(), []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if embeddings != nil {
			t.Error("expected nil for empty input")
		}
	})
}

func TestOpenAIEmbedderMethods(t *testing.T) {
	embedder := NewOpenAIEmbedder(&OpenAIConfig{
		APIKey: "test",
		Model:  "text-embedding-3-large",
	})

	if embedder.Model() != "text-embedding-3-large" {
		t.Errorf("expected Model()=text-embedding-3-large, got %s", embedder.Model())
	}

	if embedder.Dimensions() != 3072 {
		t.Errorf("expected Dimensions()=3072, got %d", embedder.Dimensions())
	}
}
