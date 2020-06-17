package skiptro

import (
	"fmt"

	"github.com/corona10/goimagehash"
)

func FindSimilarFrames(source, target []*goimagehash.ImageHash, tolerance int) ([][]bool, error) {
	similarityMatrix := make([][]bool, len(source))
	for i, h1 := range source {
		similarityMatrix[i] = make([]bool, len(target))
		for j, h2 := range target {
			dist, err := h1.Distance(h2)
			if err != nil {
				return nil, fmt.Errorf("could not calculate distance: %w", err)
			}

			if dist < tolerance {
				similarityMatrix[i][j] = true
			}
		}
	}

	return similarityMatrix, nil
}
