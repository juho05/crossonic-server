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
	if selection != "all" {
		var err error
		instanceID, err = db.System().InstanceID(context.Background())
		if err != nil {
			return fmt.Errorf("remove crossonic id in %s: get instance ID: %w", path, err)
		}
	}

	fmt.Printf("Removing crossonic tags in %s...\n", path)
	var counter int
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, _ error) error {
		ext := filepath.Ext(path)
		if !strings.HasPrefix(mime.TypeByExtension(ext), "audio/") {
			return nil
		}
		if counter%5 == 0 {
			fmt.Print("\rProcessed: ", counter)
		}
		file, err := audiotags.Open(path)
		if err != nil {
			log.Errorf("remove crossonic id in %s: %s", path, err)
			return nil
		}
		defer file.Close()
		if !file.HasMedia() {
			log.Errorf("remove crossonic id in %s: unsupported format", path)
			return nil
		}
		tags := file.ReadTags()
		var changed bool
		if selection == "current" {
			_, changed = tags["crossonic_id_"+instanceID]
			delete(tags, "crossonic_id_"+instanceID)
		} else {
			for k := range tags {
				if strings.HasPrefix(k, "crossonic_") {
					changed = true
					delete(tags, k)
				}
			}
		}
		if changed {
			if !file.WriteTags(tags) {
				log.Errorf("remove crossonic id in %s: write failed", path)
				return nil
			}
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
