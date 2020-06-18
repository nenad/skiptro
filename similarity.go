package skiptro

import (
	"fmt"
	"time"

	"github.com/corona10/goimagehash"
)

type (
	Similarity struct {
		Matrix     [][]bool
		StartIndex int
		EndIndex   int
	}

	Scene struct {
		Start      time.Duration
		End        time.Duration
		Similarity Similarity
	}
)

// FindLongestScene returns the longest matching scene between two
// TODO Return hash frames of the intro in the scene
func FindLongestScene(source, target []*goimagehash.ImageHash, tolerance int, duration time.Duration) (Scene, error) {
	similarityMatrix := make([][]bool, len(source))
	for i, h1 := range source {
		similarityMatrix[i] = make([]bool, len(target))
		for j, h2 := range target {
			dist, err := h1.Distance(h2)
			if err != nil {
				return Scene{}, fmt.Errorf("could not calculate distance: %w", err)
			}

			if dist < tolerance {
				similarityMatrix[i][j] = true
			}
		}
	}

	// TODO Maybe have a threshold for seconds of dissimilarity for intros with different scenes (i.e. The Office)
	rows := len(similarityMatrix)
	cols := len(similarityMatrix[0])

	// TODO Configurable
	skipTolerance := 3
	curSkip := 0
	inFlow := false

	// Right/top side of diagonal
	targetFrame, max := 0, 0
	for i := 0; i < rows; i++ {
		j := 0
		diagSimilar := 0
		k := 0
		for i+k < rows && j+k < cols {
			similar := similarityMatrix[i+k][j+k] ||
				clamp(i+k-1, j+k+1, similarityMatrix) ||
				clamp(i+k+1, j+k-1, similarityMatrix) ||
				clamp(i+k, j+k-1, similarityMatrix) ||
				clamp(i+k-1, j+k, similarityMatrix)

			if similar {
				diagSimilar++
				inFlow = true
			} else if inFlow {
				curSkip++
				if curSkip == skipTolerance {
					curSkip = 0
					inFlow = false
				}

				if inFlow {
					diagSimilar++
				}
			}

			if diagSimilar > max {
				max = diagSimilar
				targetFrame = j + k
			}

			k++
		}
	}

	// Left/bottom side of diagonal
	for j := 1; j < cols; j++ {
		i := 0
		diagSimilar := 0
		k := 0
		for i+k < rows && j+k < cols {
			similar := similarityMatrix[i+k][j+k] ||
				clamp(i+k-1, j+k+1, similarityMatrix) ||
				clamp(i+k+1, j+k-1, similarityMatrix) ||
				clamp(i+k, j+k-1, similarityMatrix) ||
				clamp(i+k-1, j+k, similarityMatrix)

			if similar {
				diagSimilar++
				inFlow = true
			} else if inFlow {
				curSkip++
				if curSkip == skipTolerance {
					curSkip = 0
					inFlow = false
				}

				if inFlow {
					diagSimilar++
				}
			}

			if diagSimilar > max {
				max = diagSimilar
				targetFrame = j + k
			}

			k++
		}
	}

	targetBegin := targetFrame - max

	frameDuration := float64(duration.Milliseconds()) / float64(len(similarityMatrix))

	start, err := time.ParseDuration(fmt.Sprintf("%ds", int((frameDuration*float64(targetBegin))/1000)))
	if err != nil {
		return Scene{}, fmt.Errorf("could not parse starting time: %w", err)
	}

	end, err := time.ParseDuration(fmt.Sprintf("%ds", int((frameDuration*float64(max))/1000)))
	if err != nil {
		return Scene{}, fmt.Errorf("could not parse starting time: %w", err)
	}

	return Scene{
		Start: start,
		End:   start + end,
		Similarity: Similarity{
			Matrix:     similarityMatrix,
			StartIndex: targetBegin,
			EndIndex:   targetBegin + max,
		},
	}, nil
}

func clamp(i, j int, matrix [][]bool) bool {
	if i < 0 || i >= len(matrix) {
		return false
	}

	if j < 0 || j >= len(matrix[0]) {
		return false
	}

	return matrix[i][j]
}
