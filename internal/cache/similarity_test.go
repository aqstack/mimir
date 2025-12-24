package cache

import (
	"math"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		delta    float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 2, 3},
			b:        []float64{1, 2, 3},
			expected: 1.0,
			delta:    0.0001,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{-1, 0, 0},
			expected: -1.0,
			delta:    0.0001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "similar vectors",
			a:        []float64{1, 2, 3},
			b:        []float64{1, 2, 4},
			expected: 0.9914,
			delta:    0.001,
		},
		{
			name:     "different length vectors",
			a:        []float64{1, 2},
			b:        []float64{1, 2, 3},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "zero vector a",
			a:        []float64{0, 0, 0},
			b:        []float64{1, 2, 3},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "zero vector b",
			a:        []float64{1, 2, 3},
			b:        []float64{0, 0, 0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "unit vectors 45 degrees",
			a:        []float64{1, 0},
			b:        []float64{math.Sqrt(2) / 2, math.Sqrt(2) / 2},
			expected: math.Sqrt(2) / 2,
			delta:    0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if math.Abs(result-tt.expected) > tt.delta {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestEuclideanDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		delta    float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 2, 3},
			b:        []float64{1, 2, 3},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "unit distance",
			a:        []float64{0, 0},
			b:        []float64{1, 0},
			expected: 1.0,
			delta:    0.0001,
		},
		{
			name:     "3-4-5 triangle",
			a:        []float64{0, 0},
			b:        []float64{3, 4},
			expected: 5.0,
			delta:    0.0001,
		},
		{
			name:     "different length vectors",
			a:        []float64{1, 2},
			b:        []float64{1, 2, 3},
			expected: math.Inf(1),
			delta:    0.0,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: math.Inf(1),
			delta:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EuclideanDistance(tt.a, tt.b)
			if math.IsInf(tt.expected, 1) {
				if !math.IsInf(result, 1) {
					t.Errorf("expected +Inf, got %f", result)
				}
			} else if math.Abs(result-tt.expected) > tt.delta {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestNormalizeVector(t *testing.T) {
	tests := []struct {
		name           string
		input          []float64
		expectedLength float64
		delta          float64
	}{
		{
			name:           "non-zero vector",
			input:          []float64{3, 4},
			expectedLength: 1.0,
			delta:          0.0001,
		},
		{
			name:           "unit vector",
			input:          []float64{1, 0, 0},
			expectedLength: 1.0,
			delta:          0.0001,
		},
		{
			name:           "larger vector",
			input:          []float64{1, 2, 3, 4, 5},
			expectedLength: 1.0,
			delta:          0.0001,
		},
		{
			name:           "zero vector",
			input:          []float64{0, 0, 0},
			expectedLength: 0.0,
			delta:          0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeVector(tt.input)

			// Calculate length of result
			var length float64
			for _, v := range result {
				length += v * v
			}
			length = math.Sqrt(length)

			if math.Abs(length-tt.expectedLength) > tt.delta {
				t.Errorf("expected length %f, got %f", tt.expectedLength, length)
			}
		})
	}

	// Test that direction is preserved
	t.Run("direction preserved", func(t *testing.T) {
		input := []float64{3, 4}
		result := NormalizeVector(input)

		// Result should be {0.6, 0.8}
		if math.Abs(result[0]-0.6) > 0.0001 {
			t.Errorf("expected result[0]=0.6, got %f", result[0])
		}
		if math.Abs(result[1]-0.8) > 0.0001 {
			t.Errorf("expected result[1]=0.8, got %f", result[1])
		}
	})
}

func BenchmarkCosineSimilarity(b *testing.B) {
	// Create 768-dimensional vectors (typical embedding size)
	a := make([]float64, 768)
	vecB := make([]float64, 768)
	for i := range a {
		a[i] = float64(i) / 768.0
		vecB[i] = float64(i+1) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(a, vecB)
	}
}
