package mockdb

import (
	"context"
	"time"

	"github.com/juho05/crossonic-server/repos"
)

type ArtistRepository struct {
	CreateMock                     func(ctx context.Context, params repos.CreateArtistParams) (string, error)
	CreateIfNotExistsByNameMock    func(ctx context.Context, params []repos.CreateArtistParams) error
	UpdateMock                     func(ctx context.Context, id string, params repos.UpdateArtistParams) error
	DeleteIfNoAlbumsAndNoSongsMock func(ctx context.Context) error
	FindOrCreateIDsByNamesMock     func(ctx context.Context, names []string) ([]string, error)
	FindByIDMock                   func(ctx context.Context, id string, include repos.IncludeArtistInfo) (*repos.CompleteArtist, error)
	FindByNamesMock                func(ctx context.Context, names []string, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error)
	FindAllMock                    func(ctx context.Context, params repos.FindArtistsParams, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error)
	FindBySearchMock               func(ctx context.Context, query string, onlyAlbumArtists bool, paginate repos.Paginate, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error)
	FindStarredMock                func(ctx context.Context, paginate repos.Paginate, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error)
	GetAlbumsMock                  func(ctx context.Context, id string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error)
	GetAppearsOnAlbumsMock         func(ctx context.Context, id string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error)
	StarMock                       func(ctx context.Context, user, artistID string) error
	UnStarMock                     func(ctx context.Context, user, artistID string) error
	SetRatingMock                  func(ctx context.Context, user, artistID string, rating int) error
	RemoveRatingMock               func(ctx context.Context, user, artistID string) error
	GetInfoMock                    func(ctx context.Context, artistID string) (*repos.ArtistInfo, error)
	SetInfoMock                    func(ctx context.Context, artistID string, params repos.SetArtistInfo) error
	MigrateAnnotationsMock         func(ctx context.Context, oldId, newId string) error
	FindArtistIDsToMigrateMock     func(ctx context.Context, scanStartTime time.Time) ([]repos.FindArtistIDsToMigrateResult, error)
}

func (a ArtistRepository) GetAppearsOnAlbums(ctx context.Context, id string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	if a.GetAlbumsMock != nil {
		return a.GetAlbumsMock(ctx, id, include)
	}
	panic("not implemented")
}

func (a ArtistRepository) Create(ctx context.Context, params repos.CreateArtistParams) (string, error) {
	if a.CreateMock != nil {
		return a.CreateMock(ctx, params)
	}
	panic("not implemented")
}

func (a ArtistRepository) CreateIfNotExistsByName(ctx context.Context, params []repos.CreateArtistParams) error {
	if a.CreateIfNotExistsByNameMock != nil {
		return a.CreateIfNotExistsByNameMock(ctx, params)
	}
	panic("not implemented")
}

func (a ArtistRepository) Update(ctx context.Context, id string, params repos.UpdateArtistParams) error {
	if a.UpdateMock != nil {
		return a.UpdateMock(ctx, id, params)
	}
	panic("not implemented")
}

func (a ArtistRepository) DeleteIfNoAlbumsAndNoSongs(ctx context.Context) error {
	if a.DeleteIfNoAlbumsAndNoSongsMock != nil {
		return a.DeleteIfNoAlbumsAndNoSongsMock(ctx)
	}
	panic("not implemented")
}

func (a ArtistRepository) FindOrCreateIDsByNames(ctx context.Context, names []string) ([]string, error) {
	if a.FindOrCreateIDsByNamesMock != nil {
		return a.FindOrCreateIDsByNamesMock(ctx, names)
	}
	panic("not implemented")
}

func (a ArtistRepository) FindByID(ctx context.Context, id string, include repos.IncludeArtistInfo) (*repos.CompleteArtist, error) {
	if a.FindByIDMock != nil {
		return a.FindByIDMock(ctx, id, include)
	}
	panic("not implemented")
}

func (a ArtistRepository) FindByNames(ctx context.Context, names []string, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	if a.FindByNamesMock != nil {
		return a.FindByNamesMock(ctx, names, include)
	}
	panic("not implemented")
}

func (a ArtistRepository) FindAll(ctx context.Context, params repos.FindArtistsParams, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	if a.FindAllMock != nil {
		return a.FindAllMock(ctx, params, include)
	}
	panic("not implemented")
}

func (a ArtistRepository) FindBySearch(ctx context.Context, query string, onlyAlbumArtists bool, paginate repos.Paginate, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	if a.FindBySearchMock != nil {
		return a.FindBySearchMock(ctx, query, onlyAlbumArtists, paginate, include)
	}
	panic("not implemented")
}

func (a ArtistRepository) FindStarred(ctx context.Context, paginate repos.Paginate, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	if a.FindStarredMock != nil {
		return a.FindStarredMock(ctx, paginate, include)
	}
	panic("not implemented")
}

func (a ArtistRepository) GetAlbums(ctx context.Context, id string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	if a.GetAlbumsMock != nil {
		return a.GetAlbumsMock(ctx, id, include)
	}
	panic("not implemented")
}

func (a ArtistRepository) Star(ctx context.Context, user, artistID string) error {
	if a.StarMock != nil {
		return a.StarMock(ctx, user, artistID)
	}
	panic("not implemented")
}

func (a ArtistRepository) UnStar(ctx context.Context, user, artistID string) error {
	if a.UnStarMock != nil {
		return a.UnStarMock(ctx, user, artistID)
	}
	panic("not implemented")
}

func (a ArtistRepository) SetRating(ctx context.Context, user, artistID string, rating int) error {
	if a.SetRatingMock != nil {
		return a.SetRatingMock(ctx, user, artistID, rating)
	}
	panic("not implemented")
}

func (a ArtistRepository) RemoveRating(ctx context.Context, user, artistID string) error {
	if a.RemoveRatingMock != nil {
		return a.RemoveRatingMock(ctx, user, artistID)
	}
	panic("not implemented")
}

func (a ArtistRepository) GetInfo(ctx context.Context, artistID string) (*repos.ArtistInfo, error) {
	if a.GetInfoMock != nil {
		return a.GetInfoMock(ctx, artistID)
	}
	panic("not implemented")
}

func (a ArtistRepository) SetInfo(ctx context.Context, artistID string, params repos.SetArtistInfo) error {
	if a.SetInfoMock != nil {
		return a.SetInfoMock(ctx, artistID, params)
	}
	panic("not implemented")
}

func (a ArtistRepository) MigrateAnnotations(ctx context.Context, oldId, newId string) error {
	if a.MigrateAnnotationsMock != nil {
		return a.MigrateAnnotationsMock(ctx, oldId, newId)
	}
	panic("not implemented")
}

func (a ArtistRepository) FindArtistIDsToMigrate(ctx context.Context, scanStartTime time.Time) ([]repos.FindArtistIDsToMigrateResult, error) {
	if a.FindArtistIDsToMigrateMock != nil {
		return a.FindArtistIDsToMigrateMock(ctx, scanStartTime)
	}
	panic("not implemented")
}
