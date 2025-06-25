package responses

import (
	"os"
	"path/filepath"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
)

func HasCoverArt(id string, conf config.Config) bool {
	idType, ok := crossonic.GetIDType(id)
	if !ok {
		return false
	}
	coverDir := filepath.Join(conf.DataDir, "covers")
	info, err := os.Stat(filepath.Join(coverDir, id))
	if err != nil {
		return idType == crossonic.IDTypeArtist
	}
	return info.Size() != 0
}
