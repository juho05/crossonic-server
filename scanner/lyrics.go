package scanner

import (
	"os"
	"strings"

	"github.com/juho05/crossonic-server/audiotags"
)

func (s *Scanner) scanLyrics(sideCarPath string, tags audiotags.KeyMap) *string {
	content, err := os.ReadFile(sideCarPath)
	if err == nil {
		str := string(content)
		return &str
	}

	var lyrics *string
	if l, ok := tags["lyrics"]; ok {
		ly := strings.Join(l, "\n")
		lyrics = &ly
	} else if l, ok := tags["unsyncedlyrics"]; ok {
		ly := strings.Join(l, "\n")
		lyrics = &ly
	}
	return lyrics
}
