package postgres

import (
	"context"

	"github.com/juho05/crossonic-server/repos"
	"github.com/nullism/bqb"
)

type genreRepository struct {
	db executer
	tx func(ctx context.Context, fn func(g genreRepository) error) error
}

func (g genreRepository) CreateIfNotExists(ctx context.Context, names []string) error {
	return g.tx(ctx, func(g genreRepository) error {
		return execBatch(names, func(names []string) error {
			valueList := bqb.Optional("")
			for _, n := range names {
				valueList.Comma("(?)", n)
			}
			q := bqb.New("INSERT INTO genres (name) VALUES ? ON CONFLICT DO NOTHING", valueList)
			return executeQuery(ctx, g.db, q)
		})
	})
}

func (g genreRepository) DeleteIfNoSongs(ctx context.Context) error {
	q := bqb.New("DELETE FROM genres USING genres AS gens LEFT JOIN song_genre ON gens.name = song_genre.genre_name WHERE genres.name = gens.name AND song_genre.genre_name IS NULL")
	return executeQuery(ctx, g.db, q)
}

func (g genreRepository) FindAllWithCounts(ctx context.Context) ([]*repos.GenreWithCounts, error) {
	q := bqb.New(`SELECT genres.name, COALESCE(al.count, 0) AS album_count, COALESCE(so.count, 0) AS song_count FROM genres
		LEFT JOIN (
			SELECT genre_name, COUNT(*) AS count FROM (
				SELECT song_genre.genre_name, songs.album_id FROM song_genre JOIN songs ON songs.id = song_genre.song_id WHERE songs.album_id IS NOT NULL GROUP BY song_genre.genre_name,songs.album_id
			) GROUP BY genre_name
		) al ON al.genre_name = genres.name
		LEFT JOIN (
			SELECT genre_name, COUNT(*) AS count FROM song_genre GROUP BY genre_name
		) so ON so.genre_name = genres.name
		ORDER BY lower(genres.name)`)
	return selectQuery[*repos.GenreWithCounts](ctx, g.db, q)
}
