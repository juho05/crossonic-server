package mockdb

import (
	"context"

	"github.com/juho05/crossonic-server/repos"
)

type MusicFolderRepository struct {
	FindAllMock                              func(ctx context.Context, user string) ([]repos.MusicFolder, error)
	CreateOrUpdateMock                       func(ctx context.Context, folders []repos.CreateMusicFolderParams) error
	DeleteMusicFoldersNotInMock              func(ctx context.Context, keepIDs []int) error
	DeleteAllUserAssociationsMock            func(ctx context.Context) error
	CreateUserAssociationsMock               func(ctx context.Context, folderId int, users []string) error
	GetAllArtistAsssociationsMock            func(ctx context.Context) ([]repos.ArtistMusicFolderAssociation, error)
	DeleteAllArtistAssociationsMock          func(ctx context.Context) error
	CreateArtistAssociationsMock             func(ctx context.Context, associations []repos.ArtistMusicFolderAssociation) error
	DeleteArtistAssociationsWithoutSongsMock func(ctx context.Context) error
	GetUserMusicFolderIDsMock                func(ctx context.Context, user string, requestedIDs []int) ([]int, error)
}

func (m MusicFolderRepository) FindAll(ctx context.Context, user string) ([]repos.MusicFolder, error) {
	if m.FindAllMock != nil {
		return m.FindAllMock(ctx, user)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) CreateOrUpdate(ctx context.Context, folders []repos.CreateMusicFolderParams) error {
	if m.CreateOrUpdateMock != nil {
		return m.CreateOrUpdateMock(ctx, folders)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) DeleteMusicFoldersNotIn(ctx context.Context, keepIDs []int) error {
	if m.DeleteMusicFoldersNotInMock != nil {
		return m.DeleteMusicFoldersNotInMock(ctx, keepIDs)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) DeleteAllUserAssociations(ctx context.Context) error {
	if m.DeleteAllUserAssociationsMock != nil {
		return m.DeleteAllUserAssociationsMock(ctx)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) CreateUserAssociations(ctx context.Context, folderId int, users []string) error {
	if m.CreateUserAssociationsMock != nil {
		return m.CreateUserAssociationsMock(ctx, folderId, users)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) GetAllArtistAsssociations(ctx context.Context) ([]repos.ArtistMusicFolderAssociation, error) {
	if m.GetAllArtistAsssociationsMock != nil {
		return m.GetAllArtistAsssociationsMock(ctx)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) DeleteAllArtistAssociations(ctx context.Context) error {
	if m.DeleteAllArtistAssociationsMock != nil {
		return m.DeleteAllArtistAssociationsMock(ctx)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) CreateArtistAssociations(ctx context.Context, associations []repos.ArtistMusicFolderAssociation) error {
	if m.CreateArtistAssociationsMock != nil {
		return m.CreateArtistAssociationsMock(ctx, associations)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) DeleteArtistAssociationsWithoutSongs(ctx context.Context) error {
	if m.DeleteArtistAssociationsWithoutSongsMock != nil {
		return m.DeleteArtistAssociationsWithoutSongsMock(ctx)
	}
	panic("not implemented")
}

func (m MusicFolderRepository) GetUserMusicFolderIDs(ctx context.Context, user string, requestedIDs []int) ([]int, error) {
	if m.GetUserMusicFolderIDsMock != nil {
		return m.GetUserMusicFolderIDsMock(ctx, user, requestedIDs)
	}
	panic("not implemented")
}
