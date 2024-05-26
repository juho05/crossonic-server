package ffmpeg

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/log"
)

type format struct {
	outFormat       string
	mime            string
	encoder         string
	minBitRateK     int
	defaultBitRateK int
	maxBitRateK     int
}

var formats = map[string]format{
	"mp3": {
		outFormat:       "mp3",
		mime:            "audio/mpeg",
		encoder:         "libmp3lame",
		minBitRateK:     64,
		defaultBitRateK: 192,
		maxBitRateK:     320,
	},
	"opus": {
		outFormat:       "ogg",
		mime:            "audio/ogg",
		encoder:         "libopus",
		minBitRateK:     32,
		defaultBitRateK: 192,
		maxBitRateK:     512,
	},
	"ogg": {
		outFormat:       "ogg",
		mime:            "audio/ogg",
		encoder:         "libopus",
		minBitRateK:     32,
		defaultBitRateK: 192,
		maxBitRateK:     500,
	},
	"vorbis": {
		outFormat:       "ogg",
		mime:            "audio/ogg",
		encoder:         "libvorbis",
		minBitRateK:     96,
		defaultBitRateK: 192,
		maxBitRateK:     500,
	},
	"aac": {
		outFormat:       "adts",
		mime:            "audio/aac",
		encoder:         "aac",
		minBitRateK:     64,
		defaultBitRateK: 192,
		maxBitRateK:     500,
	},
}

type Transcoder struct {
}

func NewTranscoder() (*Transcoder, error) {
	err := initialize()
	if err != nil {
		return nil, fmt.Errorf("new transcoder: %w", err)
	}
	cleanSeekRawCache()
	return &Transcoder{}, nil
}

func (t *Transcoder) Transcode(path string, format string, maxBitRateK int, timeOffset time.Duration) (out io.Reader, bitRate int, err error) {
	format = strings.ToLower(format)
	f, ok := formats[format]
	if !ok {
		if format != "" {
			log.Warnf("Requested transcoding format %s not supported. Falling back to opus...")
		}
		f = formats["opus"]
	}
	if maxBitRateK == 0 {
		maxBitRateK = f.defaultBitRateK
	}
	maxBitRateK = min(f.maxBitRateK, maxBitRateK)
	maxBitRateK = max(f.minBitRateK, maxBitRateK)
	args := []string{"-v", "0", "-ss", fmt.Sprintf("%dus", timeOffset.Microseconds()), "-i", path, "-map", "0:a:0", "-vn", "-b:a", fmt.Sprintf("%dk", maxBitRateK), "-c:a", f.encoder, "-f", f.outFormat, "-"}

	pipeR, pipeW := io.Pipe()

	cmd := exec.Command(ffmpegPath, args...)
	cmd.Stdout = pipeW

	err = cmd.Start()
	if err != nil {
		return nil, 0, fmt.Errorf("ffmpeg: transcode: %w", err)
	}
	go func() {
		cmd.Wait()
		pipeW.Close()
	}()
	return pipeR, maxBitRateK, nil
}

func (t *Transcoder) SeekRaw(path string, timeOffset time.Duration) (string, error) {
	err := os.MkdirAll(filepath.Join(config.CacheDir(), "seek-raw"), 0755)
	if err != nil {
		return "", fmt.Errorf("ffmpeg: seek raw: %w", err)
	}

	ext := filepath.Ext(path)
	name := strings.TrimSuffix(filepath.Base(path), ext)
	cachePath := filepath.Join(config.CacheDir(), fmt.Sprintf("%s-%s%s", name, crossonic.GenID(), ext))

	cmd := exec.Command(ffmpegPath, "-v", "0", "-ss", fmt.Sprintf("%dus", timeOffset.Microseconds()), "-i", path, "-map", "0:a:0", "-vn", cachePath)
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ffmpeg: seek raw: %w", err)
	}
	return cachePath, nil
}

func cleanSeekRawCache() {
	err := os.RemoveAll(filepath.Join(config.CacheDir(), "seek-raw"))
	if err != nil {
		log.Errorf("failed to clean seek-raw cache dir: %w", err)
	}
}

func GetContentTypeFromFormatString(format string, maxBitRateK int) (string, int) {
	format = strings.ToLower(format)
	f, ok := formats[format]
	if !ok {
		f = formats["opus"]
	}
	if maxBitRateK == 0 {
		maxBitRateK = f.defaultBitRateK
	}
	maxBitRateK = min(f.maxBitRateK, maxBitRateK)
	maxBitRateK = max(f.minBitRateK, maxBitRateK)
	return f.mime, maxBitRateK
}
