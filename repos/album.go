package repos

import (
	"context"
	"time"
)

// models

type Album struct {
	ID             string     `db:"id"`
	Name           string     `db:"name"`
	Created        time.Time  `db:"created"`
	Updated        time.Time  `db:"updated"`
	Year           *int       `db:"year"`
	RecordLabels   StringList `db:"record_labels"`
	MusicBrainzID  *string    `db:"music_brainz_id"`
	ReleaseMBID    *string    `db:"release_mbid"`
	ReleaseTypes   StringList `db:"release_types"`
	IsCompilation  *bool      `db:"is_compilation"`
	ReplayGain     *float64   `db:"replay_gain"`
	ReplayGainPeak *float64   `db:"replay_gain_peak"`
}

type AlbumTrackInfo struct {
	TrackCount int        `db:"track_count"`
	Duration   DurationMS `db:"duration_ms"`
}

type AlbumAnnotations struct {
	Starred       *time.Time `db:"starred"`
	UserRating    *int       `db:"user_rating"`
	AverageRating *float64   `db:"avg_rating"`
}

type AlbumPlayInfo struct {
	PlayCount  int        `db:"play_count"`
	LastPlayed *time.Time `db:"last_played"`
}

type AlbumLists struct {
	Genres  []string    `db:"-"`
	Artists []ArtistRef `db:"-"`
}

type CompleteAlbum struct {
	Album
	*AlbumTrackInfo
	*AlbumAnnotations
	*AlbumPlayInfo
	*AlbumLists
}

type AlbumInfo struct {
	AlbumID       string     `db:"id"`
	Updated       *time.Time `db:"info_updated"`
	Description   *string    `db:"description"`
	LastFMURL     *string    `db:"lastfm_url"`
	LastFMMBID    *string    `db:"lastfm_mbid"`
	MusicBrainzID *string    `db:"music_brainz_id"`
}

type AlbumArtistConnection struct {
	AlbumID  string `db:"album_id"`
	ArtistID string `db:"artist_id"`
}

// params

type IncludeAlbumInfo struct {
	TrackInfo bool

	User        string
	Annotations bool
	PlayInfo    bool

	Genres  bool
	Artists bool
}

func IncludeAlbumInfoBare() IncludeAlbumInfo {
	return IncludeAlbumInfo{}
}

func IncludeAlbumInfoFull(user string) IncludeAlbumInfo {
	return IncludeAlbumInfo{
		TrackInfo:   true,
		User:        user,
		Annotations: true,
		PlayInfo:    true,
		Artists:     true,
		Genres:      true,
	}
}

type CreateAlbumParams struct {
	Name           string
	Year           *int
	RecordLabels   StringList
	MusicBrainzID  *string
	ReleaseMBID    *string
	ReleaseTypes   StringList
	IsCompilation  *bool
	ReplayGain     *float64
	ReplayGainPeak *float64
}

type UpdateAlbumParams struct {
	Name           Optional[string]
	Year           Optional[*int]
	RecordLabels   Optional[StringList]
	MusicBrainzID  Optional[*string]
	ReleaseMBID    Optional[*string]
	ReleaseTypes   Optional[StringList]
	IsCompilation  Optional[*bool]
	ReplayGain     Optional[*float64]
	ReplayGainPeak Optional[*float64]
}

type FindAlbumSortBy int

const (
	FindAlbumSortByName FindAlbumSortBy = iota
	FindAlbumSortByCreated
	FindAlbumSortByRating
	FindAlbumSortByStarred
	FindAlbumSortRandom
	FindAlbumSortByYear
	FindAlbumSortByFrequent
	FindAlbumSortByRecent
)

type FindAlbumParams struct {
	SortBy   FindAlbumSortBy
	FromYear *int
	ToYear   *int
	Genres   []string
	Paginate Paginate
}

type SetAlbumInfo struct {
	Description *string
	LastFMURL   *string
	LastFMMBID  *string
}

// return types

type FindAlbumsByNameWithArtistMatchCountResult struct {
	AlbumID            string  `db:"id"`
	AlbumMusicBrainzID *string `db:"music_brainz_id"`
	ArtistMatches      int     `db:"artist_matches"`
}

type AlbumRepository interface {
	Create(ctx context.Context, params CreateAlbumParams) (string, error)
	Update(ctx context.Context, id string, params UpdateAlbumParams) error
	DeleteIfNoTracks(ctx context.Context) error

	FindByID(ctx context.Context, id string, include IncludeAlbumInfo) (*CompleteAlbum, error)
	FindAll(ctx context.Context, params FindAlbumParams, include IncludeAlbumInfo) ([]*CompleteAlbum, error)
	FindBySearch(ctx context.Context, query string, paginate Paginate, include IncludeAlbumInfo) ([]*CompleteAlbum, error)
	FindStarred(ctx context.Context, paginate Paginate, include IncludeAlbumInfo) ([]*CompleteAlbum, error)

	GetTracks(ctx context.Context, id string, include IncludeSongInfo) ([]*CompleteSong, error)

	Star(ctx context.Context, user, albumID string) error
	UnStar(ctx context.Context, user, albumID string) error

	SetRating(ctx context.Context, user, albumID string, rating int) error
	RemoveRating(ctx context.Context, user, albumID string) error

	GetInfo(ctx context.Context, albumID string) (*AlbumInfo, error)
	SetInfo(ctx context.Context, albumID string, params SetAlbumInfo) error

	GetAllArtistConnections(ctx context.Context) ([]AlbumArtistConnection, error)
	RemoveAllArtistConnections(ctx context.Context) error
	CreateArtistConnections(ctx context.Context, connections []AlbumArtistConnection) error
}
