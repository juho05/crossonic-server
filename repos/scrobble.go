package repos

import (
	"context"
	"time"
)

// models

type Scrobble struct {
	User                    string         `db:"user_name"`
	SongID                  string         `db:"song_id"`
	AlbumID                 *string        `db:"album_id"`
	Time                    time.Time      `db:"time"`
	SongDuration            DurationMS     `db:"song_duration_ms"`
	Duration                NullDurationMS `db:"duration_ms"`
	SubmittedToListenBrainz bool           `db:"submitted_to_listenbrainz"`
	NowPlaying              bool           `db:"now_playing"`
}

type NowPlayingSong struct {
	*CompleteSong
	User string    `db:"user_name"`
	Time time.Time `db:"time"`
}

type ScrobbleTopSong struct {
	*CompleteSong
	TotalDuration DurationMS `db:"total_duration_ms"`
}

// params

type CreateScrobbleParams struct {
	User                    string
	SongID                  string
	AlbumID                 *string
	Time                    time.Time
	SongDuration            DurationMS
	Duration                NullDurationMS
	SubmittedToListenBrainz bool
	NowPlaying              bool
}

type ScrobbleRepository interface {
	CreateMultiple(ctx context.Context, params []CreateScrobbleParams) error
	FindPossibleConflicts(ctx context.Context, user string, songIDs []string, times []time.Time) ([]*Scrobble, error)

	DeleteNowPlaying(ctx context.Context, user string) error
	GetNowPlaying(ctx context.Context, user string) (*Scrobble, error)
	GetNowPlayingSongs(ctx context.Context, include IncludeSongInfo) ([]*NowPlayingSong, error)

	FindUnsubmittedLBScrobbles(ctx context.Context) ([]*Scrobble, error)
	SetLBSubmittedByUsers(ctx context.Context, users []string) error

	GetDurationSum(ctx context.Context, user string, start, end time.Time) (DurationMS, error)
	GetDistinctSongCount(ctx context.Context, user string, start, end time.Time) (int, error)
	GetDistinctAlbumCount(ctx context.Context, user string, start, end time.Time) (int, error)
	GetDistinctArtistCount(ctx context.Context, user string, start, end time.Time) (int, error)
	GetTopSongsByDuration(ctx context.Context, user string, start, end time.Time, offset, limit int, include IncludeSongInfo) ([]*ScrobbleTopSong, error)
}
