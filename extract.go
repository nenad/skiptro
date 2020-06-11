package skiptro

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os/exec"
	"time"

	"github.com/corona10/goimagehash"
)

var (
	soi = []byte{0xff, 0xd8}
	eoi = []byte{0xff, 0xd9}
)

type HashExtractor struct {
	HashFunc  func(mage image.Image) (*goimagehash.ImageHash, error)
	FrameStep int
}

// ExtractHashes returns images from a video
func (h *HashExtractor) ExtractHashes(filename string, at time.Duration, duration time.Duration) ([]*goimagehash.ImageHash, error) {
	// TODO Maybe extract streaming from output?
	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.0f", at.Seconds()),
		"-i", filename,
		"-an",           // Disable audio stream
		"-c:v", "mjpeg", // Set encoder to mjpeg
		"-f", "image2pipe", // Set output to image2pipe
		"-vf", fmt.Sprintf("fps=%d", h.FrameStep),
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

	off := 0
	buf := out.Bytes()

	var hashes []*goimagehash.ImageHash
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

		frame, err := jpeg.Decode(bytes.NewBuffer(tail[start:end]))
		if err != nil {
			return nil, fmt.Errorf("could not decode JPEG at bytes %d to %d: %w", start+off, end+off, err)
		}

		// TODO Maybe parallelize?
		hash, err := h.HashFunc(frame)
		if err != nil {
			return nil, fmt.Errorf("could not hash frame: %w", err)
		}

		hashes = append(hashes, hash)

		// Advance the offset after the last image found
		off += end
	}

	return hashes, nil
}
