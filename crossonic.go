package crossonic

import (
	"embed"
	"io/fs"
	"log"
)

var (
	ServerName        = "crossonic-server"
	Version    string = "dev"
)

//go:embed all:db/migrations
var migrationsFS embed.FS
var MigrationsFS fs.FS

func init() {
	var err error
	MigrationsFS, err = fs.Sub(migrationsFS, "db/migrations")
	if err != nil {
		log.Fatal(err)
	}
}
