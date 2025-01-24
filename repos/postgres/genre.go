package postgres

import (
	"context"

	"github.com/juho05/crossonic-server/repos"
	"github.com/nullism/bqb"
)

type genreRepository struct {
	db executer
}

func (g genreRepository) CreateIfNotExists(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	valueList := bqb.Optional("")
	for _, n := range names {
		valueList.Comma("(?)", n)
	}
	q := bqb.New("INSERT INTO genres (name) VALUES ? ON CONFLICT DO NOTHING", valueList)
	return executeQuery(ctx, g.db, q)
}

func (g genreRepository) DeleteAll(ctx context.Context) error {
	q := bqb.New("DELETE FROM genres")
	return executeQuery(ctx, g.db, q)
}

func (g genreRepository) FindAllWithCounts(ctx context.Context) ([]*repos.GenreWithCounts, error) {
	q := bqb.New(`SELECT genres.name, COALESCE(al.count, 0) AS album_count, COALESCE(so.count, 0) AS song_count FROM genres
		LEFT JOIN (
			SELECT genre_name, COUNT(*) AS count FROM album_genre GROUP BY genre_name
		) al ON al.genre_name = genres.name
		LEFT JOIN (
			SELECT genre_name, COUNT(*) AS count FROM song_genre GROUP BY genre_name
		) so ON so.genre_name = genres.name
		ORDER BY lower(genres.name)`)
	return selectQuery[*repos.GenreWithCounts](ctx, g.db, q)
}
