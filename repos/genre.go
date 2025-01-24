package repos

import "context"

// models

type Genre struct {
	Name string `db:"name"`
}

// return types

type GenreWithCounts struct {
	Genre
	AlbumCount int `db:"album_count"`
	SongCount  int `db:"song_count"`
}

type GenreRepository interface {
	CreateIfNotExists(ctx context.Context, names []string) error
	DeleteAll(ctx context.Context) error
	FindAllWithCounts(ctx context.Context) ([]*GenreWithCounts, error)
}
