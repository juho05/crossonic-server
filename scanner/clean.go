package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/juho05/crossonic-server/repos"
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

	err = s.tx.Album().DeleteIfNoTracks(ctx)
	if err != nil {
		return fmt.Errorf("delete orphaned albums: %w", err)
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
