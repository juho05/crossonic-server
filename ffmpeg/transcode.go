package ffmpeg

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
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
		Name:                  "opus",
		outFormat:             "ogg",
		Mime:                  "audio/ogg",
		encoder:               "libopus",
		minBitRateK:           32,
		defaultBitRateK:       192,
		maxBitRateK:           512,
		maxBitRatePerChannelK: 256,
	},
	"vorbis": {
		Name:                  "vorbis",
		outFormat:             "ogg",
		Mime:                  "audio/ogg",
		encoder:               "libvorbis",
		minBitRateK:           96,
		defaultBitRateK:       192,
		maxBitRateK:           480,
		maxBitRatePerChannelK: 240,
	},
	"aac": {
		Name:                  "aac",
		outFormat:             "adts",
		Mime:                  "audio/aac",
		encoder:               "aac",
		minBitRateK:           64,
		defaultBitRateK:       192,
		maxBitRateK:           500,
		maxBitRatePerChannelK: 500,
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

func (t *Transcoder) SelectFormat(name string, maxBitRateK int) (Format, int) {
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
	maxBitRateK = min(f.maxBitRateK, maxBitRateK)
	maxBitRateK = max(f.minBitRateK, maxBitRateK)
	return f, maxBitRateK
}

func (t *Transcoder) Transcode(path string, channels int, format Format, maxBitRateK int, timeOffset time.Duration, w io.Writer, onDone func()) (bitRate int, err error) {
	if maxBitRateK == 0 {
		maxBitRateK = format.defaultBitRateK
	}
	maxBitRateK = min(format.maxBitRateK, maxBitRateK)
	maxBitRateK = max(format.minBitRateK, maxBitRateK)
	maxBitRateK = min(format.maxBitRatePerChannelK*channels, maxBitRateK)
	args := []string{"-v", "0", "-ss", fmt.Sprintf("%dus", timeOffset.Microseconds()), "-i", path, "-map", "0:a:0", "-vn", "-b:a", fmt.Sprintf("%dk", maxBitRateK), "-c:a", format.encoder, "-f", format.outFormat, "-"}

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
	cmd := exec.Command(ffmpegPath, "-v", "0", "-ss", fmt.Sprintf("%dus", timeOffset.Microseconds()), "-i", path, "-map", "0:a:0", "-vn", "-")

	stderr := new(bytes.Buffer)
	err := cmd.Start()
	cmd.Stdout = w
	cmd.Stderr = stderr
	if err != nil {
		return fmt.Errorf("ffmpeg: seek raw: %w", err)
	}
	go func() {
		err = cmd.Wait()
		if err != nil {
			if stderr != nil {
				log.Errorf("ffmpeg: seek raw: %s\n%s", err, stderr.String())
			} else {
				log.Errorf("ffmpeg: seek raw: %s", err)
			}
			return
		}
		if onDone != nil {
			onDone()
		}
	}()
	return nil
}
