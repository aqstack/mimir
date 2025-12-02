package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aqstack/kallm/pkg/api"
)

// OpenAIEmbedder generates embeddings using the OpenAI API.
type OpenAIEmbedder struct {
	apiKey     string
	baseURL    string
	model      string
	dimensions int
	client     *http.Client
}

// OpenAIConfig configures the OpenAI embedder.
type OpenAIConfig struct {
	APIKey   string
	BaseURL  string
	Model    string
	Timeout  time.Duration
}

// NewOpenAIEmbedder creates a new OpenAI embedder.
func NewOpenAIEmbedder(cfg *OpenAIConfig) *OpenAIEmbedder {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "text-embedding-3-small"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Determine dimensions based on model
	dimensions := 1536 // default for text-embedding-3-small
	switch cfg.Model {
	case "text-embedding-3-large":
		dimensions = 3072
	case "text-embedding-ada-002":
		dimensions = 1536
	}

	return &OpenAIEmbedder{
		apiKey:     cfg.APIKey,
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		dimensions: dimensions,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Embed generates an embedding for the given text.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := api.EmbeddingRequest{
		Input: texts,
		Model: e.model,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp api.ErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			return nil, fmt.Errorf("API error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var embResp api.EmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([][]float64, len(embResp.Data))
	for _, d := range embResp.Data {
		result[d.Index] = d.Embedding
	}

	return result, nil
}

// Dimensions returns the dimensionality of the embeddings.
func (e *OpenAIEmbedder) Dimensions() int {
	return e.dimensions
}

// Model returns the model name used for embeddings.
func (e *OpenAIEmbedder) Model() string {
	return e.model
}
