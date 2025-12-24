package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewOllamaEmbedder(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		embedder := NewOllamaEmbedder(&OllamaConfig{})

		if embedder.baseURL != "http://localhost:11434" {
			t.Errorf("expected default baseURL, got %s", embedder.baseURL)
		}
		if embedder.model != "nomic-embed-text" {
			t.Errorf("expected default model nomic-embed-text, got %s", embedder.model)
		}
		if embedder.dimensions != 768 {
			t.Errorf("expected dimensions=768 for nomic-embed-text, got %d", embedder.dimensions)
		}
	})

	t.Run("custom values", func(t *testing.T) {
		embedder := NewOllamaEmbedder(&OllamaConfig{
			BaseURL: "http://ollama:11434",
			Model:   "mxbai-embed-large",
			Timeout: time.Minute,
		})

		if embedder.baseURL != "http://ollama:11434" {
			t.Errorf("expected custom baseURL, got %s", embedder.baseURL)
		}
		if embedder.model != "mxbai-embed-large" {
			t.Errorf("expected model mxbai-embed-large, got %s", embedder.model)
		}
		if embedder.dimensions != 1024 {
			t.Errorf("expected dimensions=1024 for mxbai-embed-large, got %d", embedder.dimensions)
		}
	})

	t.Run("model dimensions mapping", func(t *testing.T) {
		tests := []struct {
			model      string
			dimensions int
		}{
			{"nomic-embed-text", 768},
			{"mxbai-embed-large", 1024},
			{"all-minilm", 384},
			{"unknown-model", 768}, // default
		}

		for _, tt := range tests {
			embedder := NewOllamaEmbedder(&OllamaConfig{Model: tt.model})
			if embedder.dimensions != tt.dimensions {
				t.Errorf("model %s: expected dimensions=%d, got %d", tt.model, tt.dimensions, embedder.dimensions)
			}
		}
	})
}

func TestOllamaEmbedderEmbed(t *testing.T) {
	t.Run("successful embed", func(t *testing.T) {
		expectedEmbedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/api/embeddings" {
				t.Errorf("expected /api/embeddings, got %s", r.URL.Path)
			}

			var req ollamaRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}

			if req.Model != "nomic-embed-text" {
				t.Errorf("expected model nomic-embed-text, got %s", req.Model)
			}
			if req.Prompt != "test text" {
				t.Errorf("expected prompt 'test text', got %s", req.Prompt)
			}

			resp := ollamaResponse{Embedding: expectedEmbedding}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		embedder := NewOllamaEmbedder(&OllamaConfig{
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

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		embedder := NewOllamaEmbedder(&OllamaConfig{BaseURL: server.URL})
		_, err := embedder.Embed(context.Background(), "test")
		if err == nil {
			t.Error("expected error on server error")
		}
	})

	t.Run("empty embedding response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ollamaResponse{Embedding: []float64{}}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		embedder := NewOllamaEmbedder(&OllamaConfig{BaseURL: server.URL})
		_, err := embedder.Embed(context.Background(), "test")
		if err == nil {
			t.Error("expected error on empty embedding")
		}
	})

	t.Run("connection error", func(t *testing.T) {
		embedder := NewOllamaEmbedder(&OllamaConfig{
			BaseURL: "http://localhost:99999",
			Timeout: 100 * time.Millisecond,
		})
		_, err := embedder.Embed(context.Background(), "test")
		if err == nil {
			t.Error("expected error on connection failure")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second)
		}))
		defer server.Close()

		embedder := NewOllamaEmbedder(&OllamaConfig{BaseURL: server.URL})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := embedder.Embed(ctx, "test")
		if err == nil {
			t.Error("expected error on cancelled context")
		}
	})
}

func TestOllamaEmbedderEmbedBatch(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := ollamaResponse{Embedding: []float64{float64(callCount), 0.2, 0.3}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewOllamaEmbedder(&OllamaConfig{BaseURL: server.URL})

	texts := []string{"text1", "text2", "text3"}
	embeddings, err := embedder.EmbedBatch(context.Background(), texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(embeddings) != 3 {
		t.Fatalf("expected 3 embeddings, got %d", len(embeddings))
	}

	// Verify each embedding is different (based on call count)
	for i, emb := range embeddings {
		if emb[0] != float64(i+1) {
			t.Errorf("embedding %d: expected first value %f, got %f", i, float64(i+1), emb[0])
		}
	}

	if callCount != 3 {
		t.Errorf("expected 3 API calls, got %d", callCount)
	}
}

func TestOllamaEmbedderMethods(t *testing.T) {
	embedder := NewOllamaEmbedder(&OllamaConfig{
		Model: "all-minilm",
	})

	if embedder.Model() != "all-minilm" {
		t.Errorf("expected Model()=all-minilm, got %s", embedder.Model())
	}

	if embedder.Dimensions() != 384 {
		t.Errorf("expected Dimensions()=384, got %d", embedder.Dimensions())
	}
}
