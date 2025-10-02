package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

func (s *Scanner) deleteOrphaned(ctx context.Context) error {
	if s.fullScan {
		err := s.tx.Song().DeleteLastUpdatedBefore(ctx, s.scanStart)
		if err != nil {
			return fmt.Errorf("delete orphaned songs (by last updated): %w", err)
		}
	} else {
		err := s.deleteOrphanedSongsByPath(ctx)
		if err != nil {
			return fmt.Errorf("delete orphaned songs (by path): %w", err)
		}
	}

	err := s.tx.Genre().DeleteIfNoSongs(ctx)
	if err != nil {
		return fmt.Errorf("delete orphaned genres: %w", err)
	}

	err = s.cleanAlbums(ctx)
	if err != nil {
		return fmt.Errorf("clean albums: %w", err)
	}

	err = s.cleanArtists(ctx)
	if err != nil {
		return fmt.Errorf("clean artists: %w", err)
	}

	return nil
}

func (s *Scanner) cleanAlbums(ctx context.Context) error {
	ids, err := s.tx.Album().FindAlbumIDsToMigrate(ctx, s.scanStart)
	if err != nil {
		return fmt.Errorf("find albums to migrate: %w", err)
	}

	for _, id := range ids {
		log.Tracef("migrating album annotations from %s to %s", id.OldID, id.NewID)
		err := s.tx.Album().MigrateAnnotations(ctx, id.OldID, id.NewID)
		if err != nil {
			log.Errorf("failed to migrate album annotations from %s to %s: %v", id.OldID, id.NewID, err)
		}
	}

	err = s.tx.Album().DeleteIfNoTracks(ctx)
	if err != nil {
		return fmt.Errorf("delete orphaned albums: %w", err)
	}
	return nil
}

func (s *Scanner) cleanArtists(ctx context.Context) error {
	ids, err := s.tx.Artist().FindArtistIDsToMigrate(ctx, s.scanStart)
	if err != nil {
		return fmt.Errorf("find artists to migrate: %w", err)
	}

	for _, id := range ids {
		log.Tracef("migrating artist annotations from %s to %s", id.OldID, id.NewID)
		err := s.tx.Artist().MigrateAnnotations(ctx, id.OldID, id.NewID)
		if err != nil {
			log.Errorf("failed to migrate artist annotations from %s to %s: %v", id.OldID, id.NewID, err)
		}
	}

	err = s.tx.Artist().DeleteIfNoAlbumsAndNoSongs(ctx)
	if err != nil {
		return fmt.Errorf("delete orphaned artists: %w", err)
	}

	return nil
}

func (s *Scanner) deleteOrphanedSongsByPath(ctx context.Context) error {
	limit := deleteOrphanedSongsByPathBatchSize
	var waitGroup sync.WaitGroup
	removePaths := make([]string, 0, deleteOrphanedSongsByPathBatchSize)
	var foundCount atomic.Int32
	for i := 0; ; i += deleteOrphanedSongsByPathBatchSize {
		paths, err := s.tx.Song().FindPaths(ctx, s.scanStart, repos.Paginate{
			Offset: int(foundCount.Load()),
			Limit:  &limit,
		})
		if err != nil {
			return fmt.Errorf("find song paths: %w", err)
		}
		if len(paths) == 0 {
			// done
			break
		}

		removePathsChan := make(chan string, deleteOrphanedSongsByPathBatchSize)
		checkPathsChan := make(chan string, deleteOrphanedSongsByPathBatchSize)

		// find missing paths
		for range deleteOrphanedSongsByPathWorkerCount {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				for path := range checkPathsChan {
					_, err := os.Stat(path)
					if errors.Is(err, os.ErrNotExist) {
						removePathsChan <- path
					} else {
						foundCount.Add(1)
					}
				}
			}()
		}

		for _, p := range paths {
			checkPathsChan <- p
		}
		close(checkPathsChan)
		waitGroup.Wait()
		close(removePathsChan)

		for p := range removePathsChan {
			removePaths = append(removePaths, p)
		}
		if len(removePaths) > 0 {
			err := s.tx.Song().DeleteByPaths(ctx, removePaths)
			if err != nil {
				return fmt.Errorf("delete songs by paths: %w", err)
			}
			removePaths = removePaths[:0]
		}
	}

	return nil
}
