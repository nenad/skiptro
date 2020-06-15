package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
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
	workers   = flag.Int("workers", runtime.NumCPU(), "How many workers to spin up for parallel processing (default is number of processors)")
	debug     = flag.Bool("debug", false, "Prints debug statements")
	fps       = flag.Int("fps", 2, "How many frames samples should be taken in one second")
	edl       = flag.Bool("edl", false, "Should a EDL file be produced as an output for the target")
	profile   = flag.String("profile", "", "Writes a CPU profile to the disk")
)

func main() {
	flag.Parse()

	var hashFunc *skiptro.HashFunc
	switch strings.ToLower(*hashType) {
	case "difference":
		hashFunc = &skiptro.HashDifference
	case "perception":
		hashFunc = &skiptro.HashPerception
	case "average":
		hashFunc = &skiptro.HashAverage
	default:
		log.Fatal("-hashtype must be 'difference', 'perception', or 'average'")
	}

	if *source == "" {
		log.Fatal("-source option is empty or not provided")
	}

	if *target == "" {
		log.Fatal("-target option is empty or not provided")
	}

	extractor := skiptro.NewExtractor(hashFunc, *fps, *workers)

	fmt.Println("Goroutines start: ", runtime.NumGoroutine())

	if *profile != "" {
		ftrace, err := os.Create(*profile + ".trace")
		if err != nil {
			panic(err)
		}

		if err := trace.Start(ftrace); err != nil {
			panic(err)
		}
		defer trace.Stop()

		f, err := os.Create(*profile)
		if err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
	}

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
		skiptro.DebugImage("debug", similar, *fps)
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
