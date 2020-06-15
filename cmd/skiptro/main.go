package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/nenad/skiptro"
)

// TODO Save longest batch of hashes to a file to quickly compare it to many episodes
// TODO No source/target only targets (multiple files)
// TODO Targets + saved file with intro hashes

var (
	duration  = flag.Duration("duration", time.Second*20, "How long should it look for the intro")
	hashType  = flag.String("hashtype", "difference", "Which hash type should be used")
	source    = flag.String("source", "", "File which contains the intro")
	target    = flag.String("target", "", "File in which we are looking for the intro")
	tolerance = flag.Int("tolerance", 10, "How similar should the images be. Lower values means more similar.")
	debug     = flag.Bool("debug", false, "Prints debug statements")
	fps       = flag.Int("fps", 2, "How many frames samples should be taken in one second")
	edl       = flag.Bool("edl", false, "Should a EDL file be produced as an output for the target")
	profile   = flag.String("profile", "", "Writes a CPU profile to the disk")
)

func main() {
	flag.Parse()

	var hashFunc func(image.Image) (*goimagehash.ImageHash, error)
	switch strings.ToLower(*hashType) {
	case "difference":
		hashFunc = goimagehash.DifferenceHash
	case "perception":
		hashFunc = goimagehash.PerceptionHash
	case "average":
		hashFunc = goimagehash.AverageHash
	}

	if *source == "" {
		log.Fatal("-source option is empty or not provided")
	}

	if *target == "" {
		log.Fatal("-target option is empty or not provided")
	}

	extractor := skiptro.HashExtractor{
		HashFunc: hashFunc,
		FPS:      *fps,
	}

	fmt.Println("Goroutines start: ", runtime.NumGoroutine())

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
	}

	// TODO Can be done in goroutines
	wg := sync.WaitGroup{}
	wg.Add(2)
	var sHashes, tHashes []*goimagehash.ImageHash
	go func() {
		defer wg.Done()
		hashes, err := extractor.Hashes(*source, 0, *duration)
		if err != nil {
			panic(err)
		}
		sHashes = hashes
	}()

	go func() {
		defer wg.Done()
		hashes, err := extractor.Hashes(*target, 0, *duration)
		if err != nil {
			panic(err)
		}
		tHashes = hashes
	}()
	wg.Wait()

	fmt.Println("Goroutines end: ", runtime.NumGoroutine())

	fmt.Printf("Len1: %d, Len2: %d\n", len(sHashes), len(tHashes))

	similar := make([][]bool, len(sHashes))

	deltaDur := float64(duration.Milliseconds()) / float64(len(sHashes))

	for i, h1 := range sHashes {
		similar[i] = make([]bool, len(tHashes))
		for j, h2 := range tHashes {
			dist, err := h1.Distance(h2)
			if err != nil {
				panic(fmt.Errorf("error on distance: %w", err))
			}

			if dist < *tolerance {
				similar[i][j] = true
			}
		}
	}

	if *debug {
		img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: len(similar), Y: len(similar[0])}})
		// Draw matrix for visual debugging
		for x := 0; x < len(similar); x++ {
			for y := 0; y < len(similar[x]); y++ {
				// Draw checkerboard
				if (x+y)%2 == 0 {
					img.Set(x, y, color.RGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xaa})
				} else {
					img.Set(x, y, color.White)
				}

				// Mark 5sec and 30sec with different colors
				if x%(5**fps) == 0 || y%(5**fps) == 0 {
					//  #ffae98
					img.Set(x, y, color.RGBA{R: 0xff, G: 0xae, B: 0x98, A: 0xff})
				}
				if x%(30**fps) == 0 || y%(30**fps) == 0 {
					//  #98bcff
					img.Set(x, y, color.RGBA{R: 0x98, G: 0xbc, B: 0xff, A: 0xff})
				}

				// Override if there is a similar pixel
				if similar[x][y] {
					//  #a41caf
					img.Set(x, y, color.RGBA{R: 0xa4, G: 0x1c, B: 0xaf, A: 0xff})
				}
			}
		}

		imgOut, err := os.Create("debug.png")
		if err != nil {
			log.Fatal(err)
		}

		if err := png.Encode(imgOut, img); err != nil {
			log.Fatal(err)
		}
	}

	// Finds longest diagonal with similar values
	// TODO Should have a threshold for seconds of dissimilarity for intros with different scenes (i.e. The Office)
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

	// Convert to seconds
	sSource := deltaDur * float64(bi) / 1000
	sTarget := deltaDur * float64(bj) / 1000
	end := deltaDur * float64(maxFrames) / 1000

	if *edl {
		edlPath := strings.TrimSuffix(*target, path.Ext(*target)) + ".edl"
		err := ioutil.WriteFile(edlPath, []byte(fmt.Sprintf("%.2f %.2f 3\n", sTarget, sTarget+end)), 0644)
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf(
		"i: %d (source frame at %.2fs, end at %.2fs);\n"+
			"j: %d (target frame at %.2fs, end at %.2fs);\n"+
			"maxFrames: %d; duration: %.2f\n",
		bi, sSource, sSource+end,
		bj, sTarget, sTarget+end,
		maxFrames, end)
}
