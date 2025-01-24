package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/nullism/bqb"
)

type artistRepository struct {
	db executer
	tx func(ctx context.Context, fn func(a artistRepository) error) error
}

func (a artistRepository) Create(ctx context.Context, params repos.CreateArtistParams) (*repos.Artist, error) {
	q := bqb.New(`INSERT INTO artists (id, name, created, updated, music_brainz_id)
		VALUES (?, ?, NOW(), NOW(), ?) RETURNING artists.*`, crossonic.GenIDArtist(), params.Name, params.MusicBrainzID)
	return getQuery[*repos.Artist](ctx, a.db, q)
}

func (a artistRepository) CreateIfNotExistsByName(ctx context.Context, params []repos.CreateArtistParams) error {
	valueList := bqb.Optional("")
	for _, p := range params {
		valueList.Comma("(?, ?, NOW(), NOW(), ?)", crossonic.GenIDArtist(), p.Name, p.MusicBrainzID)
	}
	q := bqb.New("INSERT INTO artists (id, name, created, updated, music_brainz_id) VALUES ? ON CONFLICT (name) DO NOTHING", valueList)
	return executeQuery(ctx, a.db, q)
}

func (a artistRepository) Update(ctx context.Context, id string, params repos.UpdateArtistParams) error {
	updateList := genUpdateList(map[string]repos.OptionalGetter{
		"name":            params.Name,
		"music_brainz_id": params.MusicBrainzID,
	}, true)
	q := bqb.New("UPDATE artists SET ? WHERE artists.id = ?", updateList, id)
	return executeQueryExpectAffectedRows(ctx, a.db, q)
}

func (a artistRepository) DeleteLastUpdatedBefore(ctx context.Context, before time.Time) error {
	q := bqb.New("DELETE FROM artists WHERE updated < ?", before)
	return executeQuery(ctx, a.db, q)
}

func (a artistRepository) FindOrCreateIDsByNames(ctx context.Context, names []string) ([]string, error) {
	ids := make([]string, len(names))
	err := a.tx(ctx, func(a artistRepository) error {
		params := make([]repos.CreateArtistParams, len(names))
		for i, n := range names {
			params[i] = repos.CreateArtistParams{
				Name: n,
			}
		}
		err := a.CreateIfNotExistsByName(ctx, params)
		if err != nil {
			return fmt.Errorf("create artists by name if they don't already exist: %w", err)
		}
		artists, err := a.FindByNames(ctx, names, repos.IncludeArtistInfo{})
		if err != nil {
			return fmt.Errorf("find artists by names: %w", err)
		}
		artistNameToID := make(map[string]string, len(artists))
		for _, a := range artists {
			artistNameToID[a.Name] = a.ID
		}
		for i, n := range names {
			id, ok := artistNameToID[n]
			if ok {
				ids[i] = id
				continue
			}
		}
		return nil
	})
	if err != nil {
		return nil, wrapErr("", err)
	}
	return ids, nil
}

func (a artistRepository) FindByID(ctx context.Context, id string, include repos.IncludeArtistInfo) (*repos.CompleteArtist, error) {
	q := bqb.New("SELECT ? FROM artists ? WHERE artists.id = ?", genArtistSelectList(include), genArtistJoins(include), id)
	return getQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) FindByNames(ctx context.Context, names []string, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	q := bqb.New("SELECT ? FROM artists ? WHERE artists.name IN (?)", genArtistSelectList(include), genArtistJoins(include), names)
	return selectQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) FindAll(ctx context.Context, onlyAlbumArtists bool, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	q := bqb.New("SELECT ? FROM artists ?", genArtistSelectList(include), genArtistJoins(include))
	if onlyAlbumArtists {
		if !include.AlbumInfo {
			return nil, repos.NewError("onlyAlbumArtists only allowed if include.AlbumInfo is true", repos.ErrInvalidParams, nil)
		}
		q.Space("WHERE COALESCE(aa.count, 0) > 0")
	}
	q.Space("ORDER BY lower(artists.name)")
	return selectQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) FindBySearch(ctx context.Context, query string, onlyAlbumArtists bool, offset, limit int, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	query = strings.ToLower(query)
	q := bqb.New("SELECT ? FROM artists ?", genArtistSelectList(include), genArtistJoins(include))
	q.Space("WHERE position(? in lower(artists.name)) > 0", query)
	if onlyAlbumArtists {
		if !include.AlbumInfo {
			return nil, repos.NewError("onlyAlbumArtists only allowed if include.AlbumInfo is true", repos.ErrInvalidParams, nil)
		}
		q.And("COALESCE(aa.count, 0) > 0")
	}
	q.Space("ORDER BY position(? in lower(artists.name)), lower(artists.name)", query)
	q.Space("OFFSET ? LIMIT ?", offset, limit)
	return selectQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) GetAlbums(ctx context.Context, id string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	q := bqb.New("SELECT ? FROM albums ?", genAlbumSelectList(include), genAlbumJoins(include))
	q.Space(`WHERE EXISTS (
			SELECT album_artist.album_id, album_artist.artist_id FROM album_artist
			WHERE album_artist.album_id = albums.id AND album_artist.artist_id = ?
		)`, id)
	q.Space("ORDER BY albums.year DESC, albums.name")
	return execAlbumSelectMany(ctx, a.db, q, include)
}

func (a artistRepository) Star(ctx context.Context, user, artistID string) error {
	q := bqb.New("INSERT INTO artist_stars (artist_id, user_name, created) VALUES (?, ?, NOW()) ON CONFLICT(artist_id,user_name) DO NOTHING", artistID, user)
	return executeQuery(ctx, a.db, q)
}

func (a artistRepository) UnStar(ctx context.Context, user, artistID string) error {
	q := bqb.New("DELETE FROM artist_stars WHERE user_name = ? AND artist_id = ?", user, artistID)
	return executeQuery(ctx, a.db, q)
}

func (a artistRepository) SetRating(ctx context.Context, user, artistID string, rating int) error {
	q := bqb.New("INSERT INTO artist_ratings (artist_id,user_name,rating) VALUES (?, ?, ?) ON CONFLICT(artist_id,user_name) DO UPDATE SET rating = ?", artistID, user, rating, rating)
	return executeQuery(ctx, a.db, q)
}

func (a artistRepository) RemoveRating(ctx context.Context, user, artistID string) error {
	q := bqb.New("DELETE FROM artist_ratings WHERE user_name = ? AND artist_id = ?", user, artistID)
	return executeQuery(ctx, a.db, q)
}

// helpers

func genArtistSelectList(include repos.IncludeArtistInfo) *bqb.Query {
	q := bqb.New(`artists.id, artists.name, artists.created, artists.updated, artists.music_brainz_id`)

	if include.AlbumInfo {
		q.Comma("COALESCE(aa.count, 0) AS album_count")
	}

	if include.Annotations {
		q.Comma("avgr.rating AS avg_rating")
		if include.AnnotationUser != "" {
			q.Comma("artist_stars.created as starred, artist_ratings.rating AS user_rating")
		}
	}
	return q
}

func genArtistJoins(include repos.IncludeArtistInfo) *bqb.Query {
	q := bqb.Optional("")

	if include.AlbumInfo {
		q.Space(`LEFT JOIN (
			SELECT artist_id, COUNT(artist_id) AS count FROM album_artist GROUP BY artist_id
		) aa ON aa.artist_id = artists.id`)
	}

	if include.Annotations {
		if include.AnnotationUser != "" {
			q.Space("LEFT JOIN artist_stars ON artist_stars.artist_id = artists.id AND artist_stars.user_name = ?", include.AnnotationUser)
			q.Space("LEFT JOIN artist_ratings ON artist_ratings.artist_id = artists.id AND artist_ratings.user_name = ?", include.AnnotationUser)
		}
		q.Space(`LEFT JOIN (
				SELECT artist_id, AVG(artist_ratings.rating) AS rating FROM artist_ratings GROUP BY artist_id
			) avgr ON avgr.artist_id = artists.id`)
	}

	return q
}
