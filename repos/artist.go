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

// params

type IncludeArtistInfo struct {
	AlbumInfo      bool
	Annotations    bool
	AnnotationUser string
}

func IncludeArtistInfoBare() IncludeArtistInfo {
	return IncludeArtistInfo{}
}

func IncludeArtistInfoFull(user string) IncludeArtistInfo {
	return IncludeArtistInfo{
		AlbumInfo:      true,
		Annotations:    true,
		AnnotationUser: user,
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

type ArtistRepository interface {
	Create(ctx context.Context, params CreateArtistParams) (*Artist, error)
	CreateIfNotExistsByName(ctx context.Context, params []CreateArtistParams) error
	Update(ctx context.Context, id string, params UpdateArtistParams) error
	DeleteLastUpdatedBefore(ctx context.Context, before time.Time) error

	FindOrCreateIDsByNames(ctx context.Context, names []string) ([]string, error)

	FindByID(ctx context.Context, id string, include IncludeArtistInfo) (*CompleteArtist, error)
	FindByNames(ctx context.Context, names []string, include IncludeArtistInfo) ([]*CompleteArtist, error)
	FindAll(ctx context.Context, onlyAlbumArtists bool, include IncludeArtistInfo) ([]*CompleteArtist, error)
	FindBySearch(ctx context.Context, query string, onlyAlbumArtists bool, paginate Paginate, include IncludeArtistInfo) ([]*CompleteArtist, error)
	FindStarred(ctx context.Context, paginate Paginate, include IncludeArtistInfo) ([]*CompleteArtist, error)

	GetAlbums(ctx context.Context, id string, include IncludeAlbumInfo) ([]*CompleteAlbum, error)

	Star(ctx context.Context, user, artistID string) error
	UnStar(ctx context.Context, user, artistID string) error

	SetRating(ctx context.Context, user, artistID string, rating int) error
	RemoveRating(ctx context.Context, user, artistID string) error
}
