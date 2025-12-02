// Package embedding provides embedding generation functionality.
package embedding

import "context"

// Embedder defines the interface for generating embeddings.
type Embedder interface {
	// Embed generates an embedding for the given text.
	Embed(ctx context.Context, text string) ([]float64, error)

	// EmbedBatch generates embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)

	// Dimensions returns the dimensionality of the embeddings.
	Dimensions() int

	// Model returns the model name used for embeddings.
	Model() string
}
