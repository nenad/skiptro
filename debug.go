package skiptro

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"runtime/pprof"
	"runtime/trace"
)

func DebugImage(out string, similarityMatrix [][]bool, fps int) {
	img := image.NewRGBA(image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: len(similarityMatrix), Y: len(similarityMatrix[0])}})
	// Draw matrix for visual debugging
	for x := 0; x < len(similarityMatrix); x++ {
		for y := 0; y < len(similarityMatrix[x]); y++ {
			// Draw checkerboard
			if (x+y)%2 == 0 {
				img.Set(x, y, color.RGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xaa})
			} else {
				img.Set(x, y, color.White)
			}

			// Mark 5sec and 30sec with different colors
			if x%(5*fps) == 0 || y%(5*fps) == 0 {
				//  #ffae98
				img.Set(x, y, color.RGBA{R: 0xff, G: 0xae, B: 0x98, A: 0xff})
			}
			if x%(30*fps) == 0 || y%(30*fps) == 0 {
				//  #98bcff
				img.Set(x, y, color.RGBA{R: 0x98, G: 0xbc, B: 0xff, A: 0xff})
			}

			// Override if there is a similar pixel
			if similarityMatrix[x][y] {
				//  #a41caf
				img.Set(x, y, color.RGBA{R: 0xa4, G: 0x1c, B: 0xaf, A: 0xff})
			}
		}
	}

	imgOut, err := os.Create(out + ".png")
	if err != nil {
		log.Fatal(err)
	}

	if err := png.Encode(imgOut, img); err != nil {
		log.Fatal(err)
	}
}

func ProfileAndTrace(name string) (stopCpuFunc func(), stopTraceFunc func(), err error) {
	ftrace, err := os.Create(name + ".trace")
	if err != nil {
		return nil, nil, fmt.Errorf("could not create file: %w", err)
	}

	if err := trace.Start(ftrace); err != nil {
		return nil, nil, fmt.Errorf("could not start trace: %w", err)
	}

	f, err := os.Create(name + ".profile")
	if err != nil {
		return nil, nil, fmt.Errorf("could not create file: %w", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		return nil, nil, fmt.Errorf("could not start cpu profiler: %w", err)
	}

	return pprof.StopCPUProfile, trace.Stop, nil
}
