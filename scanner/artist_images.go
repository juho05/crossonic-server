package scanner

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

type artistImageScanner struct {
	lastScan time.Time
	images   map[string]string
	conf     config.Config
	fullScan bool
}

func (s *Scanner) findArtistImages(ctx context.Context) (map[string]string, error) {
	scanner := &artistImageScanner{
		lastScan: s.lastScan,
		images:   make(map[string]string),
		conf:     s.conf,
		fullScan: s.fullScan,
	}

	err := scanner.scanDir(ctx, s.mediaDir)
	if err != nil {
		return nil, fmt.Errorf("scan dir: %w", err)
	}
	return scanner.images, nil
}

func (a *artistImageScanner) scanDir(ctx context.Context, mediaDir string) error {
	err := filepath.WalkDir(mediaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		if d.IsDir() {
			return nil
		}

		fileName := filepath.Base(path)
		ext := filepath.Ext(fileName)
		fileType := mime.TypeByExtension(ext)
		if fileType == "image/jpeg" || fileType == "image/png" {
			for _, pattern := range a.conf.ArtistImagePriority {
				match, err := filepath.Match(pattern, fileName)
				if err != nil {
					return fmt.Errorf("invalid artist image priority pattern %s: %w", pattern, err)
				}
				if !match {
					continue
				}
				if !a.fullScan {
					info, err := os.Stat(path)
					if err != nil {
						return fmt.Errorf("stat: %w", err)
					}
					if info.ModTime().Before(a.lastScan) {
						return nil
					}
				}
				dirName := filepath.Base(filepath.Dir(path))
				if dirName == "." || dirName == string(filepath.Separator) {
					log.Warnf("found artist image but could not determine parent directory name: %s", path)
					return nil
				}
				a.registerArtistImage(path, dirName)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}
	return nil
}

func (a *artistImageScanner) registerArtistImage(path, artistName string) {
	a.images[artistName] = path
}

func (s *Scanner) saveArtistImages(ctx context.Context, foundImages map[string]string) error {
	artists, err := s.tx.Artist().FindByNames(ctx, util.MapKeys(foundImages), repos.IncludeArtistInfoBare())
	if err != nil {
		return fmt.Errorf("find artists by names: %w", err)
	}

	type artistImage struct {
		artistID string
		path     string
	}

	imageChannel := make(chan artistImage, saveArtistCoversWorkerCount)

	var waitGroup sync.WaitGroup

	var saveImageErr error
	for range saveArtistCoversWorkerCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for image := range imageChannel {
				if saveImageErr != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
				err = s.saveArtistImage(image.path, image.artistID)
				if err != nil {
					if saveImageErr == nil {
						saveImageErr = fmt.Errorf("save artist image: %w", err)
					}
					return
				}
			}
		}()
	}

	for _, a := range artists {
		imageChannel <- artistImage{
			artistID: a.ID,
			path:     foundImages[a.Name],
		}
	}
	close(imageChannel)
	waitGroup.Wait()

	if saveImageErr != nil {
		return saveImageErr
	}

	return nil
}

func (s *Scanner) saveArtistImage(path, artistID string) error {
	old, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open original path: %w", err)
	}
	defer old.Close()

	newFile, err := os.Create(s.idToCoverPath(artistID))
	if err != nil {
		return fmt.Errorf("create artist image file: %w", err)
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, old)
	if err != nil {
		return fmt.Errorf("copy original path to artist image file: %w", err)
	}

	s.invalidateCoverCache(artistID)

	return nil
}
