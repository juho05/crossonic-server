package scanner

import (
	"context"
	"fmt"
)

func (s *Scanner) deleteOrphaned(ctx context.Context) error {
	err := s.tx.Song().DeleteLastUpdatedBefore(ctx, s.scanStart)
	if err != nil {
		return fmt.Errorf("delete orphaned songs: %w", err)
	}

	err = s.tx.Genre().DeleteIfNoSongs(ctx)
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
