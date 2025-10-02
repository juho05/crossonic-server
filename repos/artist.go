package repos

import (
	"context"
	"time"
)

// models

type Artist struct {
	ID            string    `db:"id"`
	Name          string    `db:"name"`
	Created       time.Time `db:"created"`
	Updated       time.Time `db:"updated"`
	MusicBrainzID *string   `db:"music_brainz_id"`
}

type ArtistAnnotations struct {
	Starred       *time.Time `db:"starred"`
	UserRating    *int       `db:"user_rating"`
	AverageRating *float64   `db:"avg_rating"`
}

type ArtistAlbumInfo struct {
	AlbumCount int `db:"album_count"`
}

type CompleteArtist struct {
	Artist
	*ArtistAnnotations
	*ArtistAlbumInfo
}

type ArtistInfo struct {
	ArtistID      string     `db:"id"`
	Updated       *time.Time `db:"info_updated"`
	Biography     *string    `db:"biography"`
	LastFMURL     *string    `db:"lastfm_url"`
	LastFMMBID    *string    `db:"lastfm_mbid"`
	MusicBrainzID *string    `db:"music_brainz_id"`
}

// params

type IncludeArtistInfo struct {
	AlbumInfo   bool
	Annotations bool
	User        string
}

func IncludeArtistInfoBare() IncludeArtistInfo {
	return IncludeArtistInfo{}
}

func IncludeArtistInfoFull(user string) IncludeArtistInfo {
	return IncludeArtistInfo{
		AlbumInfo:   true,
		Annotations: true,
		User:        user,
	}
}

type CreateArtistParams struct {
	Name          string
	MusicBrainzID *string
}

type UpdateArtistParams struct {
	Name          Optional[string]
	MusicBrainzID Optional[*string]
}

type SetArtistInfo struct {
	Biography  *string
	LastFMURL  *string
	LastFMMBID *string
}

type FindArtistsParams struct {
	OnlyAlbumArtists bool
	UpdatedAfter     *time.Time
}

// results

type FindArtistIDsToMigrateResult struct {
	OldID string `db:"old_id"`
	NewID string `db:"new_id"`
}

type ArtistRepository interface {
	Create(ctx context.Context, params CreateArtistParams) (string, error)
	CreateIfNotExistsByName(ctx context.Context, params []CreateArtistParams) error
	Update(ctx context.Context, id string, params UpdateArtistParams) error
	DeleteIfNoAlbumsAndNoSongs(ctx context.Context) error

	FindOrCreateIDsByNames(ctx context.Context, names []string) ([]string, error)

	FindByID(ctx context.Context, id string, include IncludeArtistInfo) (*CompleteArtist, error)
	FindByNames(ctx context.Context, names []string, include IncludeArtistInfo) ([]*CompleteArtist, error)
	FindAll(ctx context.Context, params FindArtistsParams, include IncludeArtistInfo) ([]*CompleteArtist, error)
	FindBySearch(ctx context.Context, query string, onlyAlbumArtists bool, paginate Paginate, include IncludeArtistInfo) ([]*CompleteArtist, error)
	FindStarred(ctx context.Context, paginate Paginate, include IncludeArtistInfo) ([]*CompleteArtist, error)

	GetAlbums(ctx context.Context, id string, include IncludeAlbumInfo) ([]*CompleteAlbum, error)
	GetAppearsOnAlbums(ctx context.Context, id string, include IncludeAlbumInfo) ([]*CompleteAlbum, error)

	Star(ctx context.Context, user, artistID string) error
	UnStar(ctx context.Context, user, artistID string) error

	SetRating(ctx context.Context, user, artistID string, rating int) error
	RemoveRating(ctx context.Context, user, artistID string) error

	GetInfo(ctx context.Context, artistID string) (*ArtistInfo, error)
	SetInfo(ctx context.Context, artistID string, params SetArtistInfo) error

	MigrateAnnotations(ctx context.Context, oldId, newId string) error
	FindArtistIDsToMigrate(ctx context.Context, scanStartTime time.Time) ([]FindArtistIDsToMigrateResult, error)
}
