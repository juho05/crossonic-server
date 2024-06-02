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

//go:embed all:db/migrations
var migrationsFS embed.FS
var MigrationsFS fs.FS

var GenID func() string
var IDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-~"
var IDRegex = regexp.MustCompile(fmt.Sprintf("^(tr)|(al)|(ar)_[%s]{12}$", strings.ReplaceAll(IDAlphabet, "-", "\\-")))

func init() {
	var err error
	MigrationsFS, err = fs.Sub(migrationsFS, "db/migrations")
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
	IDTypeSong   IDType = "tr"
	IDTypeAlbum  IDType = "al"
	IDTypeArtist IDType = "ar"
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

func GetIDType(id string) (IDType, bool) {
	parts := strings.Split(id, "_")
	if len(parts) != 2 {
		return "", false
	}
	types := []IDType{
		IDTypeSong, IDTypeAlbum, IDTypeArtist,
	}
	if !slices.Contains(types, IDType(parts[0])) {
		return "", false
	}
	return IDType(parts[0]), true
}
