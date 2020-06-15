package skiptro

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os/exec"
	"sync"
	"time"

	"github.com/corona10/goimagehash"
)

var (
	soi                     = []byte{0xff, 0xd8}
	eoi                     = []byte{0xff, 0xd9}
	HashDifference HashFunc = goimagehash.DifferenceHash
	HashPerception HashFunc = goimagehash.PerceptionHash
	HashAverage    HashFunc = goimagehash.AverageHash
)

type (
	hashResult struct {
		hash  *goimagehash.ImageHash
		err   error
		index int
	}
	imageData struct {
		index int
		bytes []byte
	}

	HashFunc func(image.Image) (*goimagehash.ImageHash, error)

	HashExtractor struct {
		HashFunc    HashFunc
		FPS         int
		WorkerCount int

		ffmpegScale string
	}
)

func NewExtractor(f *HashFunc, fps int, workers int) *HashExtractor {
	scale := ""

	switch f {
	case &HashDifference:
		scale = ",scale=9:8"
	case &HashAverage:
		scale = ",scale=8:8"
	case &HashPerception:
		scale = ",scale=64:64"
	}

	return &HashExtractor{
		HashFunc:    *f,
		FPS:         fps,
		WorkerCount: workers,
		ffmpegScale: scale,
	}
}

// ExtractHashes returns images from a video
func (h *HashExtractor) Hashes(filename string, at time.Duration, duration time.Duration) ([]*goimagehash.ImageHash, error) {
	// TODO Set up the scale depending on the hashing algorithm used. Probably better in a constructor
	// Maybe storing the allowed function pointers in variable and do comparison

	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.0f", at.Seconds()),
		"-i", filename,
		"-an",           // Disable audio stream
		"-c:v", "mjpeg", // Set encoder to mjpeg
		"-f", "image2pipe", // Set output to image2pipe
		"-vf", fmt.Sprintf("fps=%d%s", h.FPS, h.ffmpegScale),
		"-pix_fmt", "yuvj422p",
		"-q", "1",
		"-to", fmt.Sprintf("%.0f", duration.Seconds()),
		"pipe:1",
	)

	out := bytes.Buffer{}
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	buf := out.Bytes()

	var imagesData [][]byte
	off := 0
	index := 0
	for {
		if off >= len(buf) {
			break
		}
		tail := buf[off:]

		start := bytes.Index(tail, soi)
		if start == -1 {
			break
		}
		end := bytes.Index(tail, eoi)
		if end == -1 {
			break
		}
		// Account for the two bytes which are searched at the end
		end += 2
		// Advance the offset after the last image found
		off += end

		imagesData = append(imagesData, tail[start:end])
		index++
	}

	// TODO Make parent with timeout?
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(h.WorkerCount)
	workCh := make(chan imageData, len(imagesData))
	for i, imgBytes := range imagesData {
		workCh <- imageData{index: i, bytes: imgBytes}
	}
	close(workCh)

	resultCh := make(chan hashResult, len(imagesData))
	for i := 0; i < h.WorkerCount; i++ {
		go h.startWorker(&wg, ctx, workCh, resultCh)
	}

	hashes := make([]*goimagehash.ImageHash, len(imagesData))
	errCh := make(chan error, h.WorkerCount)
	go func() {
		for res := range resultCh {
			if res.err != nil {
				errCh <- res.err
				cancel()
				return
			}
			hashes[res.index] = res.hash
		}
	}()

	wg.Wait()
	close(resultCh)
	defer close(errCh) // deferring close of errCh so that there are no nil values in the channel before the select

	select {
	case err := <-errCh:
		return nil, fmt.Errorf("error while extracting frames: %w", err)
	default:
		return hashes, nil
	}
}

func (h *HashExtractor) startWorker(wg *sync.WaitGroup, ctx context.Context, imgData <-chan imageData, resultCh chan<- hashResult) {
	defer wg.Done()
	for data := range imgData {
		r := NewReader(ctx, bytes.NewBuffer(data.bytes))
		frame, err := jpeg.Decode(r)
		if err != nil {
			resultCh <- hashResult{err: fmt.Errorf("could not decode image: %w", err)}
			break
		}

		hash, err := h.HashFunc(frame)
		resultCh <- hashResult{
			hash:  hash,
			err:   err,
			index: data.index,
		}
	}
}
