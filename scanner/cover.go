package scanner

import (
	"context"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/juho05/crossonic-server/audiotags"
	"github.com/juho05/log"
)

type albumCover struct {
	id       string
	cover    *string
	songPath string
}

func (s *Scanner) runSetAlbumCoverLoop(ctx context.Context) error {
	err := s.createCoverDir()
	if err != nil {
		return fmt.Errorf("create cover dir: %w", err)
	}
	var waitGroup sync.WaitGroup
	var setCoverErr error
	for range setAlbumCoversWorkerCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			for album := range s.setAlbumCover {
				if setCoverErr != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
				if album.cover != nil {
					err := s.saveCoverFromPath(*album.cover, album.id)
					if err != nil {
						if setCoverErr == nil {
							setCoverErr = fmt.Errorf("save cover from path: %w", err)
						}
						return
					}
					continue
				}

				stat, err := os.Stat(album.songPath)
				if err != nil {
					if setCoverErr == nil {
						setCoverErr = fmt.Errorf("stat song file: %w", err)
					}
					return
				}

				err = s.saveCoverFromEmbeddedCover(stat.ModTime(), album.songPath, album.id)
				if errors.Is(err, errNoEmbeddedCover) {
					err = s.removeCover(album.id)
					if err != nil {
						if setCoverErr == nil {
							setCoverErr = fmt.Errorf("remove cover: %w", err)
						}
						return
					}
					continue
				}
				if err != nil {
					if setCoverErr == nil {
						setCoverErr = fmt.Errorf("save cover from path: %w", err)
					}
					return
				}
			}
		}()
	}
	waitGroup.Wait()
	return setCoverErr
}

func (s *Scanner) createCoverDir() error {
	// delete old cover dirs
	_ = os.RemoveAll(filepath.Join(s.coverDir, "playlists"))
	_ = os.RemoveAll(filepath.Join(s.coverDir, "artists"))
	_ = os.RemoveAll(filepath.Join(s.coverDir, "albums"))
	_ = os.RemoveAll(filepath.Join(s.coverDir, "songs"))

	// create cover dir if it doesn't exist
	err := os.MkdirAll(s.coverDir, 0755)
	if err != nil {
		return fmt.Errorf("mkdirall: %w", err)
	}
	return nil
}

func (s *Scanner) saveCoverFromPath(originalPath, id string) error {
	originalStat, err := os.Stat(originalPath)
	if err != nil {
		return fmt.Errorf("stat original path: %w", err)
	}
	coverStat, err := os.Stat(s.idToCoverPath(id))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat existing cover: %w", err)
	}
	// abort if file was not modified
	if !errors.Is(err, os.ErrNotExist) && !originalStat.ModTime().After(coverStat.ModTime()) {
		return nil
	}

	old, err := os.Open(originalPath)
	if err != nil {
		return fmt.Errorf("open original path: %w", err)
	}
	defer old.Close()

	newFile, err := os.Create(s.idToCoverPath(id))
	if err != nil {
		return fmt.Errorf("create cover file: %w", err)
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, old)
	if err != nil {
		return fmt.Errorf("copy original path to cover file: %w", err)
	}

	s.invalidateCoverCache(id)

	return nil
}

var errNoEmbeddedCover = errors.New("no embedded cover")

func (s *Scanner) saveCoverFromEmbeddedCover(lastModified time.Time, songPath, id string) error {
	coverStat, err := os.Stat(s.idToCoverPath(id))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat existing cover: %w", err)
	}
	// abort if file was not modified
	if !errors.Is(err, os.ErrNotExist) && !lastModified.After(coverStat.ModTime()) {
		return nil
	}

	newFile, err := os.Create(s.idToCoverPath(id))
	if err != nil {
		return fmt.Errorf("create cover file: %w", err)
	}
	defer newFile.Close()

	file, err := audiotags.Open(songPath)
	if err != nil {
		return fmt.Errorf("open song file: %w", err)
	}
	defer file.Close()

	img, err := file.ReadImage()
	if img == nil {
		return errNoEmbeddedCover
	}
	if err != nil {
		return fmt.Errorf("read embedded song cover: %w", err)
	}

	err = jpeg.Encode(newFile, img, nil)
	if err != nil {
		return fmt.Errorf("encode cover image into cover file: %w", err)
	}

	s.invalidateCoverCache(id)

	return nil
}

func (s *Scanner) removeCover(id string) error {
	err := os.Remove(s.idToCoverPath(id))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete cover: %w", err)
	}
	s.invalidateCoverCache(id)
	return nil
}

func (s *Scanner) invalidateCoverCache(id string) {
	for _, key := range s.coverCache.Keys() {
		if strings.HasPrefix(key, id) {
			err := s.coverCache.DeleteObject(key)
			if err != nil {
				log.Errorf("failed to invalidate cover cache for %s: %s", id, err)
			}
		}
	}
}

func (s *Scanner) idToCoverPath(id string) string {
	return filepath.Join(s.coverDir, id)
}
