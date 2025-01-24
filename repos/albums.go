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

type AlbumLists struct {
	Genres  []string    `db:"-"`
	Artists []ArtistRef `db:"-"`
}

type CompleteAlbum struct {
	Album
	*AlbumTrackInfo
	*AlbumAnnotations
	*AlbumLists
}

// params

type IncludeAlbumInfo struct {
	TrackInfo      bool
	Annotations    bool
	AnnotationUser string

	Lists bool
}

func IncludeAlbumInfoBare() IncludeAlbumInfo {
	return IncludeAlbumInfo{}
}

func IncludeAlbumInfoFull(user string) IncludeAlbumInfo {
	return IncludeAlbumInfo{
		TrackInfo:      true,
		Annotations:    true,
		AnnotationUser: user,
		Lists:          true,
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
)

type FindAlbumParams struct {
	SortBy   FindAlbumSortBy
	FromYear *int
	ToYear   *int
	Genres   []string
	Offset   int
	Limit    int
}

// return types

type FindAlbumsByNameWithArtistMatchCountResult struct {
	AlbumID            string  `db:"id"`
	AlbumMusicBrainzID *string `db:"music_brainz_id"`
	ArtistMatches      int     `db:"artist_matches"`
}

type AlbumRepository interface {
	Create(ctx context.Context, params CreateAlbumParams) (*Album, error)
	Update(ctx context.Context, id string, params UpdateAlbumParams) error
	DeleteLastUpdatedBefore(ctx context.Context, before time.Time) error

	FindByID(ctx context.Context, id string, include IncludeAlbumInfo) (*CompleteAlbum, error)
	FindAll(ctx context.Context, params FindAlbumParams, include IncludeAlbumInfo) ([]*CompleteAlbum, error)
	FindBySearchQuery(ctx context.Context, query string, offset, limit int, include IncludeAlbumInfo) ([]*CompleteAlbum, error)

	GetTracks(ctx context.Context, id string, include IncludeSongInfo) ([]*CompleteSong, error)

	RemoveArtists(ctx context.Context, albumID string) error
	AddArtists(ctx context.Context, albumID string, artistIDs []string) error
	SetArtists(ctx context.Context, albumID string, artistIDs []string) error

	RemoveGenres(ctx context.Context, albumID string) error
	AddGenres(ctx context.Context, albumID string, genres []string) error
	SetGenres(ctx context.Context, albumID string, genres []string) error

	Star(ctx context.Context, user, albumID string) error
	UnStar(ctx context.Context, user, albumID string) error

	SetRating(ctx context.Context, user, albumID string, rating int) error
	RemoveRating(ctx context.Context, user, albumID string) error

	FindAlbumsByNameWithArtistMatchCount(ctx context.Context, albumName string, artistNames []string) ([]*FindAlbumsByNameWithArtistMatchCountResult, error)
}
