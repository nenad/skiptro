package main

import (
	"fmt"
	"os"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/nenad/skiptro"
)

func main() {
	filename := os.Args[1]
	filename2 := os.Args[2]

	dur := time.Second * 20

	// TODO Can be done in goroutines
	images, err := skiptro.ExtractImages(filename, time.Second*50, dur)
	if err != nil {
		panic(err)
	}

	images2, err := skiptro.ExtractImages(filename2, time.Second, dur)
	if err != nil {
		panic(err)
	}

	similar := make([][]bool, len(images))

	deltaDur := float64(dur.Milliseconds()) / float64(len(images))

	// TODO Can be done in goroutines
	imgHashes1 := make([]*goimagehash.ImageHash, len(images))
	for i, img := range images {
		hash, _ := goimagehash.DifferenceHash(img)
		imgHashes1[i] = hash
	}

	imgHashes2 := make([]*goimagehash.ImageHash, len(images2))
	for i, img := range images2 {
		hash, _ := goimagehash.DifferenceHash(img)
		imgHashes2[i] = hash
	}

	for i, h1 := range imgHashes1 {
		similar[i] = make([]bool, len(imgHashes2))
		for j, h2 := range imgHashes2 {
			dist, _ := h1.Distance(h2)
			if dist < 10 {
				similar[i][j] = true
			}
		}
	}

	fmt.Println()

	// Print matrix for visual debugging
	for i := 0; i < len(similar); i++ {
		for j := 0; j < len(similar[i]); j++ {
			if similar[i][j] {
				fmt.Printf("O ")
			} else {
				fmt.Printf(". ")
			}
		}
		fmt.Println()
	}

	// Finds longest diagonal with similar values
	rows := len(similar)
	cols := len(similar[0])
	bi, bj, maxFrames := 0, 0, 0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			diagSimilar := 0
			k := 0
			for i+k < rows && j+k < cols {
				if similar[i+k][j+k] {
					diagSimilar++
				}

				if diagSimilar > maxFrames {
					bi = i
					bj = j
					maxFrames = diagSimilar
				}

				k++
			}
		}
	}

	fmt.Printf("i: %d (first clip starts at %.2fms); j: %d (second clip starts at %.2fms); maxFrames: %d\n", bi, deltaDur*float64(bi), bj, deltaDur*float64(bj), maxFrames)
}
