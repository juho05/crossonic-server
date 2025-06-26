package ffmpeg

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTranscoder_SelectFormat(t1 *testing.T) {
	tests := []struct {
		name            string
		formatName      string
		channels        int
		maxBitRateK     int
		wantFormat      Format
		wantMaxBitrateK int
	}{
		{"raw should result in an empty format", "raw", 2, 10, Format{}, 0},
		{"select mp3", "mp3", 2, 320, formats["mp3"], 320},
		{"mp3 bitrate too large", "mp3", 2, 500, formats["mp3"], 320},
		{"mp3 bitrate too small", "mp3", 2, 10, formats["mp3"], 64},
		{"select opus", "opus", 2, 512, formats["opus"], 512},
		{"select opus mono", "opus", 1, 512, formats["opus"], 256},
		{"opus bitrate too large", "opus", 4, 1024, formats["opus"], 512},
		{"select ogg (bitrate too small)", "ogg", 2, 10, formats["ogg"], 96},
		{"select vorbis default bitrate", "vorbis", 2, 0, formats["vorbis"], 192},
		{"select unknown format", "asdf", 2, 500, formats["mp3"], 320},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transcoder{} // constructor isn't used because SelectFormat does not need ffmpeg
			f, b := t.SelectFormat(tt.formatName, tt.channels, tt.maxBitRateK)
			assert.Equal(t1, tt.wantFormat, f)
			assert.Equal(t1, tt.wantMaxBitrateK, b)
		})
	}
}
