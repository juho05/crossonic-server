package mockdb

import (
	"context"
	"time"

	"github.com/juho05/crossonic-server/repos"
)

type AlbumRepository struct {
	CreateMock                     func(ctx context.Context, params repos.CreateAlbumParams) (string, error)
	UpdateMock                     func(ctx context.Context, id string, params repos.UpdateAlbumParams) error
	FindAlbumsWithNoTracksMock     func(ctx context.Context) error
	DeleteIfNoTracksMock           func(ctx context.Context) error
	FindByIDMock                   func(ctx context.Context, id string, include repos.IncludeAlbumInfo) (*repos.CompleteAlbum, error)
	FindAllMock                    func(ctx context.Context, params repos.FindAlbumParams, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error)
	FindBySearchMock               func(ctx context.Context, query string, paginate repos.Paginate, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error)
	FindStarredMock                func(ctx context.Context, paginate repos.Paginate, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error)
	GetTracksMock                  func(ctx context.Context, id string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error)
	StartMock                      func(ctx context.Context, user, albumID string) error
	UnStartMock                    func(ctx context.Context, user, albumID string) error
	SetRatingMock                  func(ctx context.Context, user, albumID string, rating int) error
	RemoveRatingMock               func(ctx context.Context, user, albumID string) error
	GetInfoMock                    func(ctx context.Context, albumID string) (*repos.AlbumInfo, error)
	SetInfoMock                    func(ctx context.Context, albumID string, params repos.SetAlbumInfo) error
	GetAllArtistConnectionsMock    func(ctx context.Context) ([]repos.AlbumArtistConnection, error)
	RemoveAllArtistConnectionsMock func(ctx context.Context) error
	CreateArtistConnectionsMock    func(ctx context.Context, connections []repos.AlbumArtistConnection) error
	GetAlternateVersionsMock       func(ctx context.Context, albumId string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error)
	MigrateAnnotationsMock         func(ctx context.Context, oldId, newId string) error
	FindAlbumIDsToMigrateMock      func(ctx context.Context, scanStartTime time.Time) ([]repos.FindAlbumIDsToMigrateResult, error)
}

func (a AlbumRepository) Create(ctx context.Context, params repos.CreateAlbumParams) (string, error) {
	if a.CreateMock != nil {
		return a.CreateMock(ctx, params)
	}
	panic("not implemented")
}

func (a AlbumRepository) Update(ctx context.Context, id string, params repos.UpdateAlbumParams) error {
	if a.UpdateMock != nil {
		return a.UpdateMock(ctx, id, params)
	}
	panic("not implemented")
}

func (a AlbumRepository) FindAlbumsWithNoTracks(ctx context.Context) error {
	if a.FindAlbumsWithNoTracksMock != nil {
		return a.FindAlbumsWithNoTracksMock(ctx)
	}
	panic("not implemented")
}

func (a AlbumRepository) DeleteIfNoTracks(ctx context.Context) error {
	if a.DeleteIfNoTracksMock != nil {
		return a.DeleteIfNoTracksMock(ctx)
	}
	panic("not implemented")
}

func (a AlbumRepository) FindByID(ctx context.Context, id string, include repos.IncludeAlbumInfo) (*repos.CompleteAlbum, error) {
	if a.FindByIDMock != nil {
		return a.FindByIDMock(ctx, id, include)
	}
	panic("not implemented")
}

func (a AlbumRepository) FindAll(ctx context.Context, params repos.FindAlbumParams, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	if a.FindAllMock != nil {
		return a.FindAllMock(ctx, params, include)
	}
	panic("not implemented")
}

func (a AlbumRepository) FindBySearch(ctx context.Context, query string, paginate repos.Paginate, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	if a.FindBySearchMock != nil {
		return a.FindBySearchMock(ctx, query, paginate, include)
	}
	panic("not implemented")
}

func (a AlbumRepository) FindStarred(ctx context.Context, paginate repos.Paginate, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	if a.FindStarredMock != nil {
		return a.FindStarredMock(ctx, paginate, include)
	}
	panic("not implemented")
}

func (a AlbumRepository) GetTracks(ctx context.Context, id string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	if a.GetTracksMock != nil {
		return a.GetTracksMock(ctx, id, include)
	}
	panic("not implemented")
}

func (a AlbumRepository) Star(ctx context.Context, user, albumID string) error {
	if a.StartMock != nil {
		return a.StartMock(ctx, user, albumID)
	}
	panic("not implemented")
}

func (a AlbumRepository) UnStar(ctx context.Context, user, albumID string) error {
	if a.UnStartMock != nil {
		return a.UnStartMock(ctx, user, albumID)
	}
	panic("not implemented")
}

func (a AlbumRepository) SetRating(ctx context.Context, user, albumID string, rating int) error {
	if a.SetRatingMock != nil {
		return a.SetRatingMock(ctx, user, albumID, rating)
	}
	panic("not implemented")
}

func (a AlbumRepository) RemoveRating(ctx context.Context, user, albumID string) error {
	if a.RemoveRatingMock != nil {
		return a.RemoveRatingMock(ctx, user, albumID)
	}
	panic("not implemented")
}

func (a AlbumRepository) GetInfo(ctx context.Context, albumID string) (*repos.AlbumInfo, error) {
	if a.GetInfoMock != nil {
		return a.GetInfoMock(ctx, albumID)
	}
	panic("not implemented")
}

func (a AlbumRepository) SetInfo(ctx context.Context, albumID string, params repos.SetAlbumInfo) error {
	if a.SetInfoMock != nil {
		return a.SetInfoMock(ctx, albumID, params)
	}
	panic("not implemented")
}

func (a AlbumRepository) GetAllArtistConnections(ctx context.Context) ([]repos.AlbumArtistConnection, error) {
	if a.GetAllArtistConnectionsMock != nil {
		return a.GetAllArtistConnectionsMock(ctx)
	}
	panic("not implemented")
}

func (a AlbumRepository) RemoveAllArtistConnections(ctx context.Context) error {
	if a.RemoveAllArtistConnectionsMock != nil {
		return a.RemoveAllArtistConnectionsMock(ctx)
	}
	panic("not implemented")
}

func (a AlbumRepository) CreateArtistConnections(ctx context.Context, connections []repos.AlbumArtistConnection) error {
	if a.CreateArtistConnectionsMock != nil {
		return a.CreateArtistConnectionsMock(ctx, connections)
	}
	panic("not implemented")
}
func (a AlbumRepository) GetAlternateVersions(ctx context.Context, albumId string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	if a.GetAlternateVersionsMock != nil {
		return a.GetAlternateVersionsMock(ctx, albumId, include)
	}
	panic("not implemented")
}

func (a AlbumRepository) MigrateAnnotations(ctx context.Context, oldId, newId string) error {
	if a.MigrateAnnotationsMock != nil {
		return a.MigrateAnnotationsMock(ctx, oldId, newId)
	}
	panic("not implemented")
}

func (a AlbumRepository) FindAlbumIDsToMigrate(ctx context.Context, scanStartTime time.Time) ([]repos.FindAlbumIDsToMigrateResult, error) {
	if a.FindAlbumIDsToMigrateMock != nil {
		return a.FindAlbumIDsToMigrateMock(ctx, scanStartTime)
	}
	panic("not implemented")
}
