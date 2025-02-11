package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

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
	var waitGroup sync.WaitGroup

	checkPathsChan := make(chan string, deleteOrphanedSongsByPathBatchSize)

	removePathsChan := make(chan string, deleteOrphanedSongsByPathBatchSize)

	// find missing paths
	for range deleteOrphanedSongsByPathWorkerCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for path := range checkPathsChan {
				_, err := os.Stat(path)
				if errors.Is(err, os.ErrNotExist) {
					removePathsChan <- path
				}
			}
		}()
	}

	// delete songs by paths
	deleteSongsDone := make(chan error)
	go func() {
		defer close(deleteSongsDone)

		removePaths := make([]string, 0, deleteOrphanedSongsByPathBatchSize)
		for path := range removePathsChan {
			removePaths = append(removePaths, path)
			if len(removePaths) >= deleteOrphanedSongsByPathBatchSize {
				err := s.tx.Song().DeleteByPaths(ctx, removePaths)
				if err != nil {
					deleteSongsDone <- fmt.Errorf("delete songs by paths: %w", err)
					return
				}
				removePaths = removePaths[:0]
			}
		}
		if len(removePaths) > 0 {
			err := s.tx.Song().DeleteByPaths(ctx, removePaths)
			if err != nil {
				deleteSongsDone <- fmt.Errorf("delete songs by paths: %w", err)
				return
			}
		}
	}()

	// find all song paths
	limit := deleteOrphanedSongsByPathBatchSize
	for i := 0; ; i += deleteOrphanedSongsByPathBatchSize {
		paths, err := s.tx.Song().FindPaths(ctx, s.lastScan, repos.Paginate{
			Offset: i,
			Limit:  &limit,
		})
		if err != nil {
			close(checkPathsChan)
			waitGroup.Wait()
			close(removePathsChan)
			<-deleteSongsDone
			return fmt.Errorf("find song paths: %w", err)
		}
		if len(paths) == 0 {
			break
		}
		for _, p := range paths {
			select {
			case <-ctx.Done():
				break
			default:
			}
			checkPathsChan <- p
		}
	}

	// finish path checker
	close(checkPathsChan)
	waitGroup.Wait()

	// finish song remover
	close(removePathsChan)
	err := <-deleteSongsDone
	if err != nil {
		return err
	}

	return nil
}
