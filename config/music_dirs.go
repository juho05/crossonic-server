package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type MusicDir struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Path  string   `json:"path"`
	Users []string `json:"users"`
}

func (c Config) GetMusicDirs() ([]MusicDir, error) {
	if c.musicDir != "" {
		stat, err := os.Stat(c.musicDir)
		if err != nil {
			return nil, fmt.Errorf("cannot access music dir: %s: %w", c.musicDir, err)
		}
		if !stat.IsDir() {
			return nil, fmt.Errorf("music dir not a directory: %s", c.musicDir)
		}
		return []MusicDir{
			{
				ID:    1,
				Name:  "Default",
				Path:  c.musicDir,
				Users: nil,
			},
		}, nil
	}

	configFile, err := os.Open(c.musicDirConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open music dir config file: %w", err)
	}
	defer configFile.Close()

	var musicDirs []MusicDir
	err = json.NewDecoder(configFile).Decode(&musicDirs)
	if err != nil {
		return nil, fmt.Errorf("invalid music dir config file: %w", err)
	}

	for i, dir := range musicDirs {
		if dir.ID == 0 {
			return nil, fmt.Errorf("music dir with index %d has no id", i)
		}
		if dir.ID < 0 {
			return nil, fmt.Errorf("music dir ids must be positive")
		}
		if dir.Name == "" {
			return nil, fmt.Errorf("music dir %d does not have a name", dir.ID)
		}
		if dir.Path == "" {
			return nil, fmt.Errorf("music dir %d does not have a path", dir.ID)
		}
		stat, err := os.Stat(dir.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to open music dir %d: %w", dir.ID, err)
		}
		if !stat.IsDir() {
			return nil, fmt.Errorf("music dir %d does not point to a directory: %s", dir.ID, dir.Path)
		}
		abs, err := filepath.Abs(dir.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to make music dir path of %d absolute: %w", dir.ID, err)
		}
		musicDirs[i].Path = abs
	}

	return musicDirs, nil
}

func GenerateMusicDirConfigString(musicDirs []MusicDir) string {
	var sb strings.Builder
	for i, dir := range musicDirs {
		if i > 0 {
			sb.WriteString(";")
		}
		sb.WriteString(strconv.Itoa(dir.ID))
		sb.WriteString(":")
		sb.WriteString(dir.Path)
	}
	return sb.String()
}
