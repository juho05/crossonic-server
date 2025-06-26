package mockdb

import (
	"context"
	"github.com/juho05/crossonic-server/repos"
)

type GenreRepository struct {
	CreateIfNotExistsMock func(ctx context.Context, names []string) error
	DeleteIfNoSongsMock   func(ctx context.Context) error
	FindAllWithCountsMock func(ctx context.Context) ([]*repos.GenreWithCounts, error)
}

func (g GenreRepository) CreateIfNotExists(ctx context.Context, names []string) error {
	if g.CreateIfNotExistsMock != nil {
		return g.CreateIfNotExistsMock(ctx, names)
	}
	return nil
}

func (g GenreRepository) DeleteIfNoSongs(ctx context.Context) error {
	if g.DeleteIfNoSongsMock != nil {
		return g.DeleteIfNoSongsMock(ctx)
	}
	return nil
}

func (g GenreRepository) FindAllWithCounts(ctx context.Context) ([]*repos.GenreWithCounts, error) {
	if g.FindAllWithCountsMock != nil {
		return g.FindAllWithCountsMock(ctx)
	}
	return nil, nil
}
