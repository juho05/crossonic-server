package ffmpeg

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/juho05/log"
)

type Format struct {
	Name                  string
	Mime                  string
	outFormat             string
	encoder               string
	minBitRateK           int
	defaultBitRateK       int
	maxBitRateK           int
	maxBitRatePerChannelK int
}

var formats = map[string]Format{
	"mp3": {
		Name:                  "mp3",
		outFormat:             "mp3",
		Mime:                  "audio/mpeg",
		encoder:               "libmp3lame",
		minBitRateK:           64,
		defaultBitRateK:       192,
		maxBitRateK:           320,
		maxBitRatePerChannelK: 320,
	},
	"opus": {
		Name:                  "opus",
		outFormat:             "ogg",
		Mime:                  "audio/ogg",
		encoder:               "libopus",
		minBitRateK:           32,
		defaultBitRateK:       192,
		maxBitRateK:           512,
		maxBitRatePerChannelK: 256,
	},
	"ogg": {
		Name:                  "vorbis",
		outFormat:             "ogg",
		Mime:                  "audio/ogg",
		encoder:               "libvorbis",
		minBitRateK:           96,
		defaultBitRateK:       192,
		maxBitRateK:           480,
		maxBitRatePerChannelK: 240,
	},
	"vorbis": {
		Name:                  "vorbis",
		outFormat:             "ogg",
		Mime:                  "audio/ogg",
		encoder:               "libvorbis",
		minBitRateK:           80,
		defaultBitRateK:       192,
		maxBitRateK:           480,
		maxBitRatePerChannelK: 240,
	},
}

type Transcoder struct {
}

func NewTranscoder() (*Transcoder, error) {
	err := initialize()
	if err != nil {
		return nil, fmt.Errorf("new transcoder: %w", err)
	}
	return &Transcoder{}, nil
}

func (t *Transcoder) SelectFormat(name string, channels, maxBitRateK int) (Format, int) {
	if name == "raw" {
		return Format{}, 0
	}
	format := strings.ToLower(name)
	f, ok := formats[format]
	if !ok {
		if format != "" {
			log.Warnf("Requested transcoding format %s not supported. Falling back to mp3...")
		}
		f = formats["mp3"]
	}
	if maxBitRateK == 0 {
		return f, f.defaultBitRateK
	}
	maxBitRateK = min(f.maxBitRateK, channels*f.maxBitRatePerChannelK, maxBitRateK)
	maxBitRateK = max(f.minBitRateK, maxBitRateK)
	return f, maxBitRateK
}

func (t *Transcoder) Transcode(path string, channels int, format Format, maxBitRateK int, timeOffset time.Duration, w io.Writer, onDone func()) (bitRate int, err error) {
	if maxBitRateK == 0 {
		maxBitRateK = format.defaultBitRateK
	}
	maxBitRateK = min(format.maxBitRateK, channels*format.maxBitRatePerChannelK, maxBitRateK)
	maxBitRateK = max(format.minBitRateK, maxBitRateK)
	bitRateFlags := []string{"-b:a", fmt.Sprintf("%dk", maxBitRateK)}
	if format.encoder == "libvorbis" {
		// FIXME: the resulting bitrate seems to be a bit higher than requested in most cases
		bitRateFlags = []string{"-q:a"}
		if maxBitRateK <= 80 {
			bitRateFlags = append(bitRateFlags, "1")
		} else if maxBitRateK <= 96 {
			bitRateFlags = append(bitRateFlags, "2")
		} else if maxBitRateK <= 112 {
			bitRateFlags = append(bitRateFlags, "3")
		} else if maxBitRateK <= 128 {
			bitRateFlags = append(bitRateFlags, "4")
		} else if maxBitRateK <= 160 {
			bitRateFlags = append(bitRateFlags, "5")
		} else if maxBitRateK <= 192 {
			bitRateFlags = append(bitRateFlags, "6")
		} else if maxBitRateK <= 224 {
			bitRateFlags = append(bitRateFlags, "7")
		} else if maxBitRateK <= 256 {
			bitRateFlags = append(bitRateFlags, "8")
		} else if maxBitRateK <= 320 {
			bitRateFlags = append(bitRateFlags, "9")
		} else {
			bitRateFlags = append(bitRateFlags, "10")
		}
	}
	args := []string{"-v", "error", "-ss", fmt.Sprintf("%dus", timeOffset.Microseconds()), "-i", path, "-map", "0:a:0", "-vn"}
	args = append(args, bitRateFlags...)
	args = append(args, "-c:a", format.encoder, "-f", format.outFormat, "-")

	stderr := new(bytes.Buffer)
	cmd := exec.Command(ffmpegPath, args...)
	cmd.Stdout = w
	cmd.Stderr = stderr

	err = cmd.Start()
	if err != nil {
		return 0, fmt.Errorf("ffmpeg: transcode: %w", err)
	}
	go func() {
		err = cmd.Wait()
		if err != nil {
			if stderr != nil {
				log.Errorf("ffmpeg: transcode: %s\n%s", err, stderr.String())
			} else {
				log.Errorf("ffmpeg: transcode: %s", err)
			}
			return
		}
		if onDone != nil {
			onDone()
		}
	}()
	return maxBitRateK, nil
}

func (t *Transcoder) SeekRaw(path string, timeOffset time.Duration, w io.Writer, onDone func()) error {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	cmd := exec.Command(ffmpegPath, "-v", "error", "-ss", fmt.Sprintf("%dus", timeOffset.Microseconds()), "-i", path, "-map", "0:a:0", "-vn", "-c", "copy", "-f", ext, "-")

	stderr := new(bytes.Buffer)
	cmd.Stdout = w
	cmd.Stderr = stderr
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("ffmpeg: seek raw: start: %w", err)
	}
	go func() {
		err = cmd.Wait()
		if err != nil {
			if stderr != nil {
				log.Errorf("ffmpeg: seek raw: wait: %s\n%s", err, stderr.String())
			} else {
				log.Errorf("ffmpeg: seek raw: wait: %s", err)
			}
			return
		}
		if onDone != nil {
			onDone()
		}
	}()
	return nil
}
