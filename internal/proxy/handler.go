// Package proxy provides HTTP proxy handling for kallm.
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aqstack/kallm/internal/cache"
	"github.com/aqstack/kallm/internal/config"
	"github.com/aqstack/kallm/internal/embedding"
	"github.com/aqstack/kallm/internal/logger"
	"github.com/aqstack/kallm/internal/reports"
	"github.com/aqstack/kallm/pkg/api"
)

// Handler handles proxied requests with semantic caching.
type Handler struct {
	cfg       *config.Config
	cache     cache.Cache
	embedder  embedding.Embedder
	client    *http.Client
	logger    *logger.Logger
	collector *reports.Collector
}

// NewHandler creates a new proxy handler.
func NewHandler(cfg *config.Config, c cache.Cache, e embedding.Embedder, log *logger.Logger) *Handler {
	return &Handler{
		cfg:      cfg,
		cache:    c,
		embedder: e,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
		logger:    log,
		collector: reports.NewCollector(),
	}
}

// ServeHTTP handles incoming requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/health":
		h.handleHealth(w, r)
	case r.URL.Path == "/stats":
		h.handleStats(w, r)
	case r.URL.Path == "/reports" || r.URL.Path == "/reports/":
		h.handleDashboard(w, r)
	case r.URL.Path == "/reports/data":
		h.handleReportsData(w, r)
	case r.URL.Path == "/reports/logs":
		h.handleLogs(w, r)
	case r.URL.Path == "/reports/logs/clear":
		h.handleClearLogs(w, r)
	case r.URL.Path == "/v1/chat/completions":
		h.handleChatCompletions(w, r)
	case strings.HasPrefix(r.URL.Path, "/v1/"):
		// Pass through other OpenAI endpoints
		h.handlePassthrough(w, r)
	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// handleHealth handles health check requests.
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleStats handles cache statistics requests.
func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := h.cache.Stats(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleChatCompletions handles chat completion requests with caching.
func (h *Handler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Parse request
	var req api.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Skip caching for streaming requests
	if req.Stream {
		h.logger.Debug("skipping cache for streaming request")
		h.forwardRequest(w, r, body)
		return
	}

	// Generate cache key from messages
	cacheKey := h.generateCacheKey(req)

	// Get embedding for cache lookup
	emb, err := h.embedder.Embed(ctx, cacheKey)
	if err != nil {
		h.logger.Warn("failed to generate embedding, forwarding request", "error", err)
		h.forwardRequest(w, r, body)
		return
	}

	// Check cache
	if entry, similarity, found := h.cache.Get(ctx, emb, h.cfg.SimilarityThreshold); found {
		latencyMs := time.Since(startTime).Milliseconds()
		h.logger.Info("cache hit",
			"similarity", fmt.Sprintf("%.4f", similarity),
			"latency_ms", latencyMs,
		)

		// Record metrics - estimate tokens saved based on response
		tokensSaved := entry.Response.Usage.TotalTokens
		h.collector.RecordRequest(true, similarity, latencyMs, tokensSaved, cacheKey)
		h.collector.AddLog("hit", fmt.Sprintf("[HIT] %.2f%% sim, %dms - %s", similarity*100, latencyMs, truncatePrompt(cacheKey, 80)))

		// Return cached response with cache header
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Kallm-Cache", "HIT")
		w.Header().Set("X-Kallm-Similarity", fmt.Sprintf("%.4f", similarity))
		json.NewEncoder(w).Encode(entry.Response)
		return
	}

	// Cache miss - forward to OpenAI
	h.logger.Debug("cache miss, forwarding to upstream")

	resp, respBody, err := h.doUpstreamRequest(ctx, r, body)
	if err != nil {
		h.logger.Error("upstream request failed", "error", err)
		h.writeError(w, "Upstream request failed", http.StatusBadGateway)
		return
	}

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.Header().Set("X-Kallm-Cache", "MISS")

	// If successful, cache the response
	if resp.StatusCode == http.StatusOK {
		var chatResp api.ChatCompletionResponse
		if err := json.Unmarshal(respBody, &chatResp); err == nil {
			entry := &api.CacheEntry{
				Request:   req,
				Response:  chatResp,
				Embedding: emb,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(h.cfg.CacheTTL),
				HitCount:  0,
				LastHitAt: time.Now(),
			}
			if err := h.cache.Set(ctx, entry); err != nil {
				h.logger.Warn("failed to cache response", "error", err)
			} else {
				h.logger.Debug("cached response", "model", chatResp.Model)
			}
		}
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	latencyMs := time.Since(startTime).Milliseconds()

	// Record cache miss metric
	h.collector.RecordRequest(false, 0, latencyMs, 0, cacheKey)
	h.collector.AddLog("miss", fmt.Sprintf("[MISS] %dms - %s", latencyMs, truncatePrompt(cacheKey, 80)))

	h.logger.Info("upstream request completed",
		"status", resp.StatusCode,
		"latency_ms", latencyMs,
	)
}

// generateCacheKey creates a cache key from the request messages.
func (h *Handler) generateCacheKey(req api.ChatCompletionRequest) string {
	var sb strings.Builder

	for _, msg := range req.Messages {
		sb.WriteString(msg.Role)
		sb.WriteString(": ")

		switch content := msg.Content.(type) {
		case string:
			sb.WriteString(content)
		case []interface{}:
			// Handle multimodal content
			for _, part := range content {
				if p, ok := part.(map[string]interface{}); ok {
					if text, ok := p["text"].(string); ok {
						sb.WriteString(text)
					}
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// forwardRequest forwards a request to the upstream without caching.
func (h *Handler) forwardRequest(w http.ResponseWriter, r *http.Request, body []byte) {
	resp, respBody, err := h.doUpstreamRequest(r.Context(), r, body)
	if err != nil {
		h.writeError(w, "Upstream request failed", http.StatusBadGateway)
		return
	}

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// doUpstreamRequest sends a request to the upstream OpenAI API.
func (h *Handler) doUpstreamRequest(ctx context.Context, r *http.Request, body []byte) (*http.Response, []byte, error) {
	upstreamURL := h.cfg.OpenAIBaseURL + r.URL.Path

	req, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	// Copy headers
	for k, v := range r.Header {
		req.Header[k] = v
	}

	// Use configured API key if not provided in request
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.OpenAIAPIKey)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, respBody, nil
}

// handlePassthrough passes requests directly to upstream.
func (h *Handler) handlePassthrough(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	h.forwardRequest(w, r, body)
}

// writeError writes an error response.
func (h *Handler) writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(api.ErrorResponse{
		Error: api.APIError{
			Message: message,
			Type:    "kallm_error",
		},
	})
}

// handleDashboard serves the performance dashboard HTML.
func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(reports.DashboardHTML()))
}

// handleReportsData serves the performance report data as JSON.
func (h *Handler) handleReportsData(w http.ResponseWriter, r *http.Request) {
	report := h.collector.GetReport()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// handleLogs serves the recent logs as JSON.
func (h *Handler) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs := h.collector.GetLogs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleClearLogs clears the log buffer.
func (h *Handler) handleClearLogs(w http.ResponseWriter, r *http.Request) {
	h.collector.ClearLogs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
}

// truncatePrompt truncates a prompt for display.
func truncatePrompt(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
