package repos

import (
	"context"
	"time"
)

// models

type Playlist struct {
	ID      string    `db:"id"`
	Name    string    `db:"name"`
	Created time.Time `db:"created"`
	Updated time.Time `db:"updated"`
	Owner   string    `db:"owner"`
	Public  bool      `db:"public"`
	Comment *string   `db:"comment"`
}

type PlaylistTrackInfo struct {
	TrackCount int        `db:"track_count"`
	Duration   DurationMS `db:"duration_ms"`
}

type CompletePlaylist struct {
	Playlist
	*PlaylistTrackInfo
}

// params

type IncludePlaylistInfo struct {
	TrackInfo bool
}

func IncludePlaylistInfoBare() IncludePlaylistInfo {
	return IncludePlaylistInfo{}
}

func IncludePlaylistInfoFull() IncludePlaylistInfo {
	return IncludePlaylistInfo{
		TrackInfo: true,
	}
}

type CreatePlaylistParams struct {
	Name    string
	Owner   string
	Public  bool
	Comment *string
}

type UpdatePlaylistParams struct {
	Name    Optional[string]
	Public  Optional[bool]
	Comment Optional[*string]
}

type PlaylistRepository interface {
	Create(ctx context.Context, params CreatePlaylistParams) (*Playlist, error)
	Update(ctx context.Context, user, id string, params UpdatePlaylistParams) error

	FindByID(ctx context.Context, user, id string, include IncludePlaylistInfo) (*CompletePlaylist, error)
	FindAll(ctx context.Context, user string, include IncludePlaylistInfo) ([]*CompletePlaylist, error)

	GetTracks(ctx context.Context, id string, include IncludeSongInfo) ([]*CompleteSong, error)
	AddTracks(ctx context.Context, id string, songIDs []string) error
	RemoveTracks(ctx context.Context, id string, trackNumbers []int) error
	ClearTracks(ctx context.Context, id string) error
	SetTracks(ctx context.Context, id string, songIDs []string) error

	FixTrackNumbers(ctx context.Context) error

	Delete(ctx context.Context, user, id string) error
}
