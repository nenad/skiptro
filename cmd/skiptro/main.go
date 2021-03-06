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

	scene, err := skiptro.FindLongestScene(sourceHashes, targetHashes, cfg.Tolerance, cfg.Duration)
	if err != nil {
		log.Fatal("could not find longest scene: ", err)
	}

	if cfg.Debug {
		skiptro.DebugImage(scene, cfg.FPS)
	}

	if cfg.SkipFile {
		edlPath := strings.TrimSuffix(cfg.Target, path.Ext(cfg.Target)) + ".edl"
		err := ioutil.WriteFile(edlPath, skiptro.EDL(scene), 0644)
		if err != nil {
			log.Fatalf("could not write .edl file: %s", err)
		}

		m3uPath := strings.TrimSuffix(cfg.Target, path.Ext(cfg.Target)) + ".m3u"
		err = ioutil.WriteFile(m3uPath, skiptro.M3U(scene, cfg.Target), 0644)
		if err != nil {
			log.Fatalf("could not write .m3u file: %s", err)
		}
	}

	duration := scene.End - scene.Start

	fmt.Printf("Video %q stats:\n- Start: %s\n- End: %s\n- Duration: %s\n", cfg.Target, scene.Start.String(), scene.End.String(), duration.String())
}
