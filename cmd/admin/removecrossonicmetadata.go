package main

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/juho05/crossonic-server/audiotags"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

func removeCrossonicMetadata(args []string, db repos.DB, conf config.Config) error {
	if len(args) < 3 {
		fmt.Println("USAGE:", args[0], "remove-crossonic-metadata <selection> <path?>\n\nSELECTION:\n  current (delete metadata created by this crossonic instance\n  all (delete metadata created by any crossonic instance)")
		os.Exit(1)
	}
	var selection string
	switch args[2] {
	case "all":
		selection = "all"
	case "current":
		selection = "current"
	default:
		fmt.Println("USAGE:", args[0], "remove-crossonic-metadata <selection> <path?>\n\nSELECTION:\n  current (delete metadata created by this crossonic instance\n  all (delete metadata created by any crossonic instance)")
		os.Exit(1)
	}
	path := conf.MusicDir
	if len(args) == 4 {
		path = args[3]
	}

	var instanceID string
	var err error
	if selection != "all" {
		instanceID, err = db.System().InstanceID(context.Background())
		if err != nil {
			return fmt.Errorf("remove crossonic id in %s: get instance ID: %w", path, err)
		}
	}

	fmt.Printf("Removing crossonic tags in %s...\n", path)
	var counter int
	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, _ error) error {
		ext := filepath.Ext(path)
		if !strings.HasPrefix(mime.TypeByExtension(ext), "audio/") {
			return nil
		}
		if counter%10 == 0 {
			fmt.Print("\rProcessed: ", counter)
		}
		if !audiotags.RemoveCrossonicTag(path, instanceID) {
			log.Errorf("remove crossonic id in %s: write failed", path)
			return nil
		}
		counter++
		return nil
	})
	if err != nil {
		return fmt.Errorf("remove crossonic metadata: %w", err)
	}
	fmt.Println("\rProcessed: ", counter)
	fmt.Println("Done.")
	return nil
}
