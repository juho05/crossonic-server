package responses

import (
	"os"
	"path/filepath"

	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
)

func hasCoverArt(id string) bool {
	idType, ok := crossonic.GetIDType(id)
	if !ok {
		return false
	}
	var path string
	switch idType {
	case crossonic.IDTypeSong:
		path = filepath.Join(config.DataDir(), "covers", "songs")
	case crossonic.IDTypeAlbum:
		path = filepath.Join(config.DataDir(), "covers", "albums")
	case crossonic.IDTypeArtist:
		path = filepath.Join(config.DataDir(), "covers", "artists")
	case crossonic.IDTypePlaylist:
		path = filepath.Join(config.DataDir(), "covers", "playlists")
	default:
		return false
	}
	info, err := os.Stat(filepath.Join(path, id))
	if err != nil {
		return idType == crossonic.IDTypeArtist
	}
	return info.Size() != 0
}
