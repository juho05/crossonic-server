package mockdb

import (
	"context"
	"github.com/juho05/crossonic-server/repos"
	"time"
)

type ScrobbleRepository struct {
	CreateMultipleMock            func(ctx context.Context, params []repos.CreateScrobbleParams) error
	FindPossibleConflictsMock     func(ctx context.Context, user string, songIDs []string, times []time.Time) ([]*repos.Scrobble, error)
	DeleteNowPlayingMock          func(ctx context.Context, user string) error
	GetNowPlayingMock             func(ctx context.Context, user string) (*repos.Scrobble, error)
	GetNowPlayingSongsMock        func(ctx context.Context, include repos.IncludeSongInfo) ([]*repos.NowPlayingSong, error)
	FindUnsubmittedLBScrobbleMock func(ctx context.Context) ([]*repos.Scrobble, error)
	SetLBSubmittedByUsersMock     func(ctx context.Context, users []string) error
	GetDurationSumMock            func(ctx context.Context, user string, start, end time.Time) (repos.DurationMS, error)
	GetDistinctSongCountMock      func(ctx context.Context, user string, start, end time.Time) (int, error)
	GetDistinctAlbumCountMock     func(ctx context.Context, user string, start, end time.Time) (int, error)
	GetDistinctArtistCountMock    func(ctx context.Context, user string, start, end time.Time) (int, error)
	GetTopSongsByDurationMock     func(ctx context.Context, user string, start, end time.Time, offset, limit int, include repos.IncludeSongInfo) ([]*repos.ScrobbleTopSong, error)
}

func (s ScrobbleRepository) CreateMultiple(ctx context.Context, params []repos.CreateScrobbleParams) error {
	if s.CreateMultipleMock != nil {
		return s.CreateMultipleMock(ctx, params)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) FindPossibleConflicts(ctx context.Context, user string, songIDs []string, times []time.Time) ([]*repos.Scrobble, error) {
	if s.FindPossibleConflictsMock != nil {
		return s.FindPossibleConflictsMock(ctx, user, songIDs, times)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) DeleteNowPlaying(ctx context.Context, user string) error {
	if s.DeleteNowPlayingMock != nil {
		return s.DeleteNowPlayingMock(ctx, user)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) GetNowPlaying(ctx context.Context, user string) (*repos.Scrobble, error) {
	if s.GetNowPlayingMock != nil {
		return s.GetNowPlayingMock(ctx, user)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) GetNowPlayingSongs(ctx context.Context, include repos.IncludeSongInfo) ([]*repos.NowPlayingSong, error) {
	if s.GetNowPlayingSongsMock != nil {
		return s.GetNowPlayingSongsMock(ctx, include)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) FindUnsubmittedLBScrobbles(ctx context.Context) ([]*repos.Scrobble, error) {
	if s.FindUnsubmittedLBScrobbleMock != nil {
		return s.FindUnsubmittedLBScrobbleMock(ctx)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) SetLBSubmittedByUsers(ctx context.Context, users []string) error {
	if s.SetLBSubmittedByUsersMock != nil {
		return s.SetLBSubmittedByUsersMock(ctx, users)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) GetDurationSum(ctx context.Context, user string, start, end time.Time) (repos.DurationMS, error) {
	if s.GetDurationSumMock != nil {
		return s.GetDurationSumMock(ctx, user, start, end)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) GetDistinctSongCount(ctx context.Context, user string, start, end time.Time) (int, error) {
	if s.GetDistinctSongCountMock != nil {
		return s.GetDistinctSongCountMock(ctx, user, start, end)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) GetDistinctAlbumCount(ctx context.Context, user string, start, end time.Time) (int, error) {
	if s.GetDistinctAlbumCountMock != nil {
		return s.GetDistinctAlbumCountMock(ctx, user, start, end)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) GetDistinctArtistCount(ctx context.Context, user string, start, end time.Time) (int, error) {
	if s.GetDistinctArtistCountMock != nil {
		return s.GetDistinctArtistCountMock(ctx, user, start, end)
	}
	panic("not implemented")
}

func (s ScrobbleRepository) GetTopSongsByDuration(ctx context.Context, user string, start, end time.Time, offset, limit int, include repos.IncludeSongInfo) ([]*repos.ScrobbleTopSong, error) {
	if s.GetTopSongsByDurationMock != nil {
		return s.GetTopSongsByDurationMock(ctx, user, start, end, offset, limit, include)
	}
	panic("not implemented")
}
