package repos

import "context"

// models

type MusicFolder struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
	Path string `db:"path"`
}

type ArtistMusicFolderAssociation struct {
	MusicFolderID int    `db:"music_folder_id"`
	ArtistID      string `db:"artist_id"`
}

type MusicFolderRepository interface {
	FindAll(ctx context.Context, user string) ([]MusicFolder, error)
	CreateOrUpdate(ctx context.Context, folders []MusicFolder) error
	DeleteMusicFoldersNotIn(ctx context.Context, keepIDs []int) error

	DeleteAllUserAssociations(ctx context.Context) error
	CreateUserAssociations(ctx context.Context, folderId int, users []string) error

	GetAllArtistAsssociations(ctx context.Context) ([]ArtistMusicFolderAssociation, error)
	DeleteAllArtistAssociations(ctx context.Context) error
	CreateArtistAssociations(ctx context.Context, associations []ArtistMusicFolderAssociation) error

	// GetUserMusicFolderIDs returns requestedIDs if the user has access to all requested music folders and ErrForbidden otherwise.
	// If requestedIDs is empty it returns all music folders that the user has access to.
	GetUserMusicFolderIDs(ctx context.Context, user string, requestedIDs []int) ([]int, error)
}
