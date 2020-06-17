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

// Hashes returns hashed images from a video file
func (h *HashExtractor) Hashes(filename string, at time.Duration, duration time.Duration) ([]*goimagehash.ImageHash, error) {
	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.0f", at.Seconds()),
		"-i", filename, // Set input file
		"-an",           // Disable audio stream
		"-c:v", "mjpeg", // Set encoder to mjpeg
		"-f", "image2pipe", // Set output to image2pipe
		"-vf", fmt.Sprintf("fps=%d%s", h.FPS, h.ffmpegScale),
		"-pix_fmt", "yuvj422p", // Set a common pixel format for output
		"-q", "1",
		"-to", fmt.Sprintf("%.0f", duration.Seconds()), // Duration
		"pipe:1", // Pipe to file descriptor 1 (stdout)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(h.WorkerCount)
	workCh := make(chan imageData, len(imagesData))
	for i, imgBytes := range imagesData {
		workCh <- imageData{index: i, bytes: imgBytes}
	}
	close(workCh)

	resultCh := make(chan hashResult, len(imagesData))
	for i := 0; i < h.WorkerCount; i++ {
		go h.startWorker(wg, ctx, workCh, resultCh)
	}

	errCh := make(chan error, h.WorkerCount)
	doneCh := make(chan []*goimagehash.ImageHash, 1)
	defer close(doneCh)
	go func() {
		hashes := make([]*goimagehash.ImageHash, len(imagesData))
		for res := range resultCh {
			if res.err != nil {
				errCh <- res.err
				cancel()
				return
			}
			hashes[res.index] = res.hash
		}
		doneCh <- hashes
	}()

	wg.Wait()
	close(resultCh)
	// deferred close of errCh so that there are no nil values in the channel
	defer close(errCh)

	select {
	case err := <-errCh:
		return nil, fmt.Errorf("error while extracting frames: %w", err)
	case hashes := <-doneCh:
		return hashes, nil
	}
}

func (h *HashExtractor) startWorker(wg *sync.WaitGroup, ctx context.Context, input <-chan imageData, output chan<- hashResult) {
	defer wg.Done()
	for data := range input {
		r := NewReader(ctx, bytes.NewBuffer(data.bytes))
		frame, err := jpeg.Decode(r)
		if err != nil {
			output <- hashResult{err: fmt.Errorf("could not decode image: %w", err)}
			break
		}

		hash, err := h.HashFunc(frame)
		output <- hashResult{
			hash:  hash,
			err:   err,
			index: data.index,
		}
	}
}
