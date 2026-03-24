package postgres

import (
	"context"
	"fmt"

	"github.com/juho05/crossonic-server/repos"
	"github.com/nullism/bqb"
)

type musicFolderRepository struct {
	db executer
	tx func(ctx context.Context, fn func(g musicFolderRepository) error) error
}

func (m musicFolderRepository) FindAll(ctx context.Context, user string) ([]repos.MusicFolder, error) {
	q := bqb.New(`SELECT mf.id, mf.name, mf.path FROM music_folders mf 
		JOIN music_folder_users mfu ON mfu.music_folder_id = mf.id
		WHERE mfu.user_name = ?`, user)
	musicFolders, err := selectQuery[repos.MusicFolder](ctx, m.db, q)
	if err != nil {
		return nil, err
	}
	return musicFolders, nil
}

func (m musicFolderRepository) CreateOrUpdate(ctx context.Context, folders []repos.MusicFolder) error {
	if len(folders) == 0 {
		return nil
	}
	return m.tx(ctx, func(m musicFolderRepository) error {
		return execBatch(folders, func(folders []repos.MusicFolder) error {
			valueList := bqb.Optional("")
			for _, f := range folders {
				valueList.Comma("(?, ?, ?)", f.ID, f.Name, f.Path)
			}
			q := bqb.New("INSERT INTO music_folders (id, name, path) VALUES ? ON CONFLICT (id) DO UPDATE SET name = excluded.name, path = excluded.path", valueList)
			return executeQuery(ctx, m.db, q)
		})
	})
}

func (m musicFolderRepository) DeleteMusicFoldersNotIn(ctx context.Context, keepIDs []int) error {
	q := bqb.New("DELETE FROM music_folders WHERE id NOT IN (?)", keepIDs)
	return executeQuery(ctx, m.db, q)
}

func (m musicFolderRepository) DeleteAllUserAssociations(ctx context.Context) error {
	q := bqb.New("DELETE FROM music_folder_users")
	return executeQuery(ctx, m.db, q)
}

func (m musicFolderRepository) CreateUserAssociations(ctx context.Context, folderId int, users []string) error {
	if len(users) == 0 {
		return nil
	}
	return m.tx(ctx, func(m musicFolderRepository) error {
		return execBatch(users, func(users []string) error {
			valueList := bqb.Optional("")
			for _, u := range users {
				valueList.Comma("(?, ?)", folderId, u)
			}
			q := bqb.New("INSERT INTO music_folder_users (music_folder_id, user_name) VALUES ?", valueList)
			return executeQuery(ctx, m.db, q)
		})
	})
}

func (m musicFolderRepository) GetUserMusicFolderIDs(ctx context.Context, user string, requestedIDs []int) ([]int, error) {
	var result []int
	err := m.tx(ctx, func(m musicFolderRepository) error {
		var err error
		if len(requestedIDs) == 0 {
			q := bqb.New("SELECT mf.id FROM music_folders mf JOIN music_folder_users mfu ON mfu.music_folder_id = mf.id WHERE mfu.user_name = ?", user)
			result, err = selectQuery[int](ctx, m.db, q)
			return err
		}

		q := bqb.New("SELECT mf.id FROM music_folders mf JOIN music_folder_users mfu ON mfu.music_folder_id = mf.id WHERE mfu.user_name = ? AND mf.id IN (?)", user, requestedIDs)
		result, err = selectQuery[int](ctx, m.db, q)
		if len(result) != len(requestedIDs) {
			return fmt.Errorf("user does not have access to requested music folder: %w", repos.ErrForbidden)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
