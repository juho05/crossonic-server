package repos

import "context"

// models

type MusicFolder struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
	Path string `db:"path"`
}

type MusicFolderRepository interface {
	FindAll(ctx context.Context, user string) ([]MusicFolder, error)
	CreateOrUpdate(ctx context.Context, folders []MusicFolder) error
	DeleteMusicFoldersNotIn(ctx context.Context, keepIDs []int) error

	DeleteAllUserAssociations(ctx context.Context) error
	CreateUserAssociations(ctx context.Context, folderId int, users []string) error

	// GetUserMusicFolderIDs returns requestedIDs if the user has access to all requested music folders and ErrForbidden otherwise.
	// If requestedIDs is empty it returns all music folders that the user has access to.
	GetUserMusicFolderIDs(ctx context.Context, user string, requestedIDs []string) ([]int, error)
}
