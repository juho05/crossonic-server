package scanner

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

func (s *Scanner) LoadMusicDirs(tx repos.Transaction) (changed bool, err error) {
	ctx := context.Background()

	log.Trace("loading music dir config")

	musicDirs, err := s.conf.GetMusicDirs()
	if err != nil {
		return false, fmt.Errorf("get music dir config: %w", err)
	}
	s.musicDirs = musicDirs

	changed = false
	lastMusicDirConfig, err := tx.System().MusicDirConfig(ctx)
	if errors.Is(err, repos.ErrNotFound) {
		changed = true
	} else if err == nil {
		var sb strings.Builder
		for i, dir := range musicDirs {
			if i > 0 {
				sb.WriteString(";")
			}
			sb.WriteString(strconv.Itoa(dir.ID))
			sb.WriteString(":")
			sb.WriteString(dir.Path)
		}
		currentMusicDirConfig := sb.String()
		if lastMusicDirConfig != currentMusicDirConfig {
			changed = true
			err = tx.System().SetMusicDirConfig(ctx, currentMusicDirConfig)
			if err != nil {
				return false, fmt.Errorf("set system music dir config: %w", err)
			}
		}
	} else {
		return false, fmt.Errorf("get system music dir config: %w", err)
	}

	if !changed {
		log.Tracef("music dir config hasn't changed, skipping db update")
		return false, nil
	}

	log.Tracef("music dir config changed, updating database")

	err = tx.MusicFolder().DeleteAllUserAssociations(ctx)
	if err != nil {
		return false, fmt.Errorf("delete user associations: %w", err)
	}

	err = tx.MusicFolder().CreateOrUpdate(ctx, util.Map(musicDirs, func(dir config.MusicDir) repos.MusicFolder {
		return repos.MusicFolder{
			ID:   dir.ID,
			Name: dir.Name,
			Path: dir.Path,
		}
	}))
	if err != nil {
		return false, fmt.Errorf("create or update music folders in db: %w", err)
	}

	users, err := tx.User().FindAll(ctx)
	if err != nil {
		return false, fmt.Errorf("get all users")
	}
	userNames := util.Map(users, func(u *repos.User) string {
		return u.Name
	})
	for _, dir := range musicDirs {
		names := userNames

		if len(dir.Users) > 0 {
			names = make([]string, len(dir.Users))
			for _, u := range dir.Users {
				if !slices.Contains(userNames, u) {
					return false, fmt.Errorf("unknown user for music dir %d: %s", dir.ID, u)
				}
				names = append(names, u)
			}
		}

		err = tx.MusicFolder().CreateUserAssociations(ctx, dir.ID, names)
		if err != nil {
			return false, fmt.Errorf("create user associations for music folder %d: %w", dir.ID, err)
		}
	}

	err = tx.MusicFolder().DeleteMusicFoldersNotIn(ctx, util.Map(musicDirs, func(dir config.MusicDir) int {
		return dir.ID
	}))
	if err != nil {
		return false, fmt.Errorf("delete music folders that no longer exist: %w", err)
	}

	return changed, nil
}
