package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MusicDir struct {
	ID    int      `json:"-"`
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
		musicDirs[i].ID = i + 1
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
		musicDirs[i].Path = filepath.Clean(strings.TrimSpace(abs))
	}

	err = checkIfPathsSubdirOfEachOther(musicDirs)
	if err != nil {
		return nil, err
	}

	return musicDirs, nil
}

func GenerateMusicDirConfigString(musicDirs []MusicDir) string {
	var sb strings.Builder
	for i, dir := range musicDirs {
		if i > 0 {
			sb.WriteString(";")
		}
		sb.WriteString(dir.Path)
		for _, u := range dir.Users {
			sb.WriteString(":")
			sb.WriteString(u)
		}
	}
	return sb.String()
}

func checkIfPathsSubdirOfEachOther(musicDirs []MusicDir) error {
	for _, a := range musicDirs {
		for _, b := range musicDirs {
			if a.ID == b.ID {
				continue
			}
			if strings.HasPrefix(a.Path, b.Path) {
				if a.Path == b.Path {
					return fmt.Errorf("music dir %d has the same path as music dir %d", a.ID, b.ID)
				}
				return fmt.Errorf("music dir %d is a subdirectory of music dir %d", a.ID, b.ID)
			}
		}
	}
	return nil
}
