package skiptro

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	Duration  time.Duration
	HashFunc  *HashFunc
	Source    string
	Target    string
	Tolerance int
	Workers   int
	FPS       int
	EDL       bool
	Debug     bool
}

func (c *Config) Parse() error {
	set := flag.NewFlagSet("config", flag.ExitOnError)
	duration := set.Duration("duration", time.Second*20, "How long should it look for the intro")
	hashType := set.String("hashtype", "difference", "Which hash type should be used")
	source := set.String("source", "", "File which contains the intro")
	target := set.String("target", "", "File in which we are looking for the intro")
	tolerance := set.Int("tolerance", 13, "How similar should the images be. Lower values means more similar.")
	workers := set.Int("workers", runtime.NumCPU(), "How many workers to spin up for parallel processing (default is number of processors)")
	debug := set.Bool("debug", false, "Prints debug statements")
	fps := set.Int("fps", 3, "How many frames samples should be taken in one second")
	edl := set.Bool("edl", false, "Should a EDL file be produced as an output for the target")

	if err := set.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("error while parsing flags: %w", err)
	}

	if duration.Milliseconds() <= 0 {
		return fmt.Errorf("duration must be a positive number")
	}

	var hashFunc *HashFunc
	switch strings.ToLower(*hashType) {
	case "difference":
		hashFunc = &HashDifference
	case "perception":
		hashFunc = &HashPerception
	case "average":
		hashFunc = &HashAverage
	default:
		return fmt.Errorf("hashtype must be one of 'difference', 'perception', or 'average'")
	}

	if _, err := os.Stat(*source); err != nil {
		return fmt.Errorf("could not find source file: %w", err)
	}

	if _, err := os.Stat(*target); err != nil {
		return fmt.Errorf("could not find target file: %w", err)
	}

	if *tolerance <= 0 {
		return fmt.Errorf("tolerance must be greater than 0")
	}

	if *workers <= 0 {
		return fmt.Errorf("workers must be greater than 0")
	}

	if *fps <= 0 {
		return fmt.Errorf("fps must be greater than 0")
	}

	c.Duration = *duration
	c.HashFunc = hashFunc
	c.Source = *source
	c.Target = *target
	c.Tolerance = *tolerance
	c.Workers = *workers
	c.FPS = *fps
	c.EDL = *edl
	c.Debug = *debug

	return nil
}
