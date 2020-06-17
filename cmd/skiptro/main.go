package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"sync"

	"github.com/corona10/goimagehash"
	"github.com/nenad/skiptro"
)

// TODO Save longest batch of hashes to a file to quickly compare it to many episodes
// TODO No source/target only targets (multiple files)
// TODO Targets + saved file with intro hashes

func main() {

	cfg := skiptro.Config{}
	err := cfg.Parse()
	if err != nil {
		log.Fatal(err)
	}

	extractor := skiptro.NewExtractor(cfg.HashFunc, cfg.FPS, cfg.Workers)

	if cfg.Debug {
		s1, s2, err := skiptro.ProfileAndTrace()
		if err != nil {
			log.Fatal("could not start profiling: ", err)
		}
		defer s1()
		defer s2()
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	var sourceHashes, targetHashes []*goimagehash.ImageHash
	go func() {
		defer wg.Done()
		hashes, err := extractor.Hashes(cfg.Source, 0, cfg.Duration)
		if err != nil {
			panic(err)
		}
		sourceHashes = hashes
	}()

	go func() {
		defer wg.Done()
		hashes, err := extractor.Hashes(cfg.Target, 0, cfg.Duration)
		if err != nil {
			panic(err)
		}
		targetHashes = hashes
	}()
	wg.Wait()

	similar, err := skiptro.FindSimilarFrames(sourceHashes, targetHashes, cfg.Tolerance)
	if err != nil {
		log.Fatal("could not find similar frames: ", err)
	}

	if cfg.Debug {
		skiptro.DebugImage(similar, cfg.FPS)
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
	deltaDur := float64(cfg.Duration.Milliseconds()) / float64(len(similar))
	sSource := deltaDur * float64(bi) / 1000
	sTarget := deltaDur * float64(bj) / 1000
	end := deltaDur * float64(maxFrames) / 1000

	if cfg.EDL {
		edlPath := strings.TrimSuffix(cfg.Target, path.Ext(cfg.Target)) + ".edl"
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
