package crossonic

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"slices"
	"strings"

	"github.com/jaevor/go-nanoid"
)

var (
	ServerName        = "crossonic-server"
	Version    string = "dev"
)

//go:embed all:repos/migrations
var migrationsFS embed.FS
var MigrationsFS fs.FS

var GenID func() string
var IDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-~"
var IDRegex = regexp.MustCompile(fmt.Sprintf("^(tr)|(al)|(ar)|(pl)_[%s]{12}$", strings.ReplaceAll(IDAlphabet, "-", "\\-")))

func init() {
	var err error
	MigrationsFS, err = fs.Sub(migrationsFS, "repos/migrations")
	if err != nil {
		log.Fatal(err)
	}
	GenID, err = nanoid.CustomUnicode(IDAlphabet, 12)
	if err != nil {
		panic(err)
	}
}

type IDType string

const (
	IDTypeSong     IDType = "tr"
	IDTypeAlbum    IDType = "al"
	IDTypeArtist   IDType = "ar"
	IDTypePlaylist IDType = "pl"
)

func GenIDSong() string {
	return string(IDTypeSong) + "_" + GenID()
}

func GenIDAlbum() string {
	return string(IDTypeAlbum) + "_" + GenID()
}

func GenIDArtist() string {
	return string(IDTypeArtist) + "_" + GenID()
}

func GenIDPlaylist() string {
	return string(IDTypePlaylist) + "_" + GenID()
}

func GetIDType(id string) (IDType, bool) {
	parts := strings.Split(id, "_")
	if len(parts) != 2 {
		return "", false
	}
	types := []IDType{
		IDTypeSong, IDTypeAlbum, IDTypeArtist, IDTypePlaylist,
	}
	if !slices.Contains(types, IDType(parts[0])) {
		return "", false
	}
	return IDType(parts[0]), true
}

func IsIDType(id string, idType IDType) bool {
	typ, ok := GetIDType(id)
	return ok && idType == typ
}
