package crossonic

import (
	"embed"
	"io/fs"
	"log"

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

func init() {
	var err error
	MigrationsFS, err = fs.Sub(migrationsFS, "db/migrations")
	if err != nil {
		log.Fatal(err)
	}
	GenID, err = nanoid.CustomUnicode("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-~", 12)
	if err != nil {
		panic(err)
	}
}
