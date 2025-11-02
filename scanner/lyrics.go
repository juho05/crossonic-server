package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/juho05/crossonic-server/audiotags"
)

func (s *Scanner) findLyricsSidecar(songPath string) (path *string, modified bool) {
	ext := filepath.Ext(songPath)
	basePath := strings.TrimSuffix(songPath, ext)

	lrcSideCarPath := basePath + ".lrc"
	lrcFileInfo, err := os.Stat(lrcSideCarPath)
	if err == nil {
		return &lrcSideCarPath, s.lastScan.IsZero() || lrcFileInfo.ModTime().After(s.lastScan)
	}

	txtSideCarPath := basePath + ".txt"
	txtFileInfo, err := os.Stat(txtSideCarPath)
	if err == nil {
		return &txtSideCarPath, s.lastScan.IsZero() || txtFileInfo.ModTime().After(s.lastScan)
	}

	return nil, false
}

func (s *Scanner) scanLyrics(sideCarPath *string, tags audiotags.KeyMap) *string {
	if sideCarPath != nil {
		content, err := os.ReadFile(*sideCarPath)
		if err == nil {
			str := string(content)
			return &str
		}
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
