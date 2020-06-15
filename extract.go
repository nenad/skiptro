package skiptro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/corona10/goimagehash"
)

var (
	soi = []byte{0xff, 0xd8}
	eoi = []byte{0xff, 0xd9}
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

	HashExtractor struct {
		HashFunc func(mage image.Image) (*goimagehash.ImageHash, error)
		FPS      int
	}

	Metadata struct {
		Filename    string
		Width       int    `json:"width"`
		Height      int    `json:"height"`
		PixelFormat string `json:"pix_fmt"`
		FrameRate   float64
		Duration    time.Duration
	}
)

func ExtractMetadata(filename string) (Metadata, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error", // Output only when there are errors
		"-of", "json", // Output as JSON
		"-select_streams", "v:0", // Select only the first video stream, should work in most cases
		"-show_entries", "stream=width,height,pix_fmt,r_frame_rate,duration",
		filename,
	)
	buf := bytes.Buffer{}
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return Metadata{}, fmt.Errorf("could not run command: %w", err)
	}

	jsonOutput := struct {
		Streams []Metadata `json:"streams"`
	}{}

	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		return Metadata{}, fmt.Errorf("could not unmarshal output: %w", err)
	}

	meta := jsonOutput.Streams[0]
	meta.Filename = filename
	return meta, nil
}

func (m *Metadata) UnmarshalJSON(data []byte) error {
	type metaAlias Metadata
	aux := &struct {
		FrameRateRaw string `json:"r_frame_rate"`
		DurationRaw  string `json:"duration"`
		*metaAlias
	}{
		metaAlias: (*metaAlias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("could not unmarshal Metadata: %w", err)
	}

	rateStrings := strings.Split(aux.FrameRateRaw, "/")
	dividend, err := strconv.ParseFloat(rateStrings[0], 64)
	if err != nil {
		return fmt.Errorf("could not parse dividend: %w", err)
	}
	divisor, err := strconv.ParseFloat(rateStrings[1], 64)
	if err != nil {
		return fmt.Errorf("could not parse divisor: %w", err)
	}

	m.FrameRate = dividend / divisor
	m.Duration, err = time.ParseDuration(aux.DurationRaw + "s")
	if err != nil {
		return fmt.Errorf("could not parse video duration: %w", err)
	}

	return nil
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
		"-vf", fmt.Sprintf("fps=%d,scale=9:8", h.FPS), // Prepare for Difference hash
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

	n := runtime.NumCPU()

	wg := sync.WaitGroup{}
	wg.Add(n)
	workCh := make(chan imageData, len(imagesData))
	for i, imgBytes := range imagesData {
		workCh <- imageData{index: i, bytes: imgBytes}
	}
	close(workCh)

	resultCh := make(chan hashResult, len(imagesData))

	for i := 0; i < n; i++ {
		go h.startWorker(&wg, ctx, workCh, resultCh)
	}

	hashes := make([]*goimagehash.ImageHash, len(imagesData))

	var hashErr error
	go func() {
		for res := range resultCh {
			if res.err != nil {
				hashErr = res.err
				cancel()
				return
			}
			hashes[res.index] = res.hash
		}
	}()

	wg.Wait()
	close(resultCh)

	if hashErr != nil {
		return nil, hashErr
	}

	return hashes, nil
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
