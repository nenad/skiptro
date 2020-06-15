package skiptro

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type (
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
