package utils

import (
	"fmt"
	"math"
)

// dotProduct calculates the dot product of two vectors.
func dotProduct(vec1, vec2 []float32) (float32, error) {
	if len(vec1) != len(vec2) {
		return 0, fmt.Errorf("vectors must have the same dimension")
	}
	var product float32
	for i := range vec1 {
		product += vec1[i] * vec2[i]
	}
	return product, nil
}

// magnitude calculates the L2 norm (magnitude) of a vector.
func magnitude(vec []float32) float32 {
	var sumOfSquares float32
	for _, val := range vec {
		sumOfSquares += val * val
	}
	return float32(math.Sqrt(float64(sumOfSquares)))
}

// CosineSimilarity calculates the cosine similarity between two vectors.
func CosineSimilarity(vec1, vec2 []float32) (float32, error) {
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}
	dotProduct, err := dotProduct(vec1, vec2)
	if err != nil {
		return 0, err
	}

	mag1 := magnitude(vec1)
	mag2 := magnitude(vec2)

	if mag1 == 0 || mag2 == 0 {
		return 0, nil
	}

	return dotProduct / (mag1 * mag2), nil
}
