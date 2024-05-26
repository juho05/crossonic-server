package ffmpeg

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/juho05/log"
)

var ffmpegPath string

func initialize() error {
	if ffmpegPath != "" {
		return nil
	}
	var err error
	ffmpegPath, err = exec.LookPath("ffmpeg")
	if err != nil && !errors.Is(err, exec.ErrDot) {
		return fmt.Errorf("initialize ffmpeg: %w", err)
	}
	log.Tracef("FFmpeg path: %s", ffmpegPath)
	return nil
}
