package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/nullism/bqb"
)

type artistRepository struct {
	db executer
	tx func(ctx context.Context, fn func(a artistRepository) error) error
}

func (a artistRepository) Create(ctx context.Context, params repos.CreateArtistParams) (string, error) {
	id := crossonic.GenIDArtist()
	q := bqb.New(`INSERT INTO artists (id, name, created, updated, music_brainz_id, search_text)
		VALUES (?, ?, NOW(), NOW(), ?, ?)`, id, params.Name, params.MusicBrainzID, " "+util.NormalizeText(params.Name)+" ")
	return id, executeQuery(ctx, a.db, q)
}

func (a artistRepository) CreateIfNotExistsByName(ctx context.Context, params []repos.CreateArtistParams) error {
	return a.tx(ctx, func(a artistRepository) error {
		return execBatch(params, func(params []repos.CreateArtistParams) error {
			valueList := bqb.Optional("")
			for _, p := range params {
				valueList.Comma("(?, ?, NOW(), NOW(), ?, ?)", crossonic.GenIDArtist(), p.Name, p.MusicBrainzID, " "+util.NormalizeText(p.Name)+" ")
			}
			q := bqb.New("INSERT INTO artists (id, name, created, updated, music_brainz_id, search_text) VALUES ? ON CONFLICT (name) DO NOTHING", valueList)
			return executeQuery(ctx, a.db, q)
		})
	})
}

func (a artistRepository) Update(ctx context.Context, id string, params repos.UpdateArtistParams) error {
	searchText := repos.NewOptionalEmpty[string]()
	if params.Name.HasValue() {
		searchText = repos.NewOptionalFull(" " + util.NormalizeText(params.Name.Get().(string)) + " ")
	}
	updateList := genUpdateList(map[string]repos.OptionalGetter{
		"name":            params.Name,
		"music_brainz_id": params.MusicBrainzID,
		"search_text":     searchText,
	}, true)
	q := bqb.New("UPDATE artists SET ? WHERE artists.id = ?", updateList, id)
	return executeQueryExpectAffectedRows(ctx, a.db, q)
}

func (a artistRepository) DeleteIfNoAlbumsAndNoSongs(ctx context.Context) error {
	q := bqb.New(`DELETE FROM artists USING artists as arts
			LEFT JOIN song_artist ON song_artist.artist_id = arts.id
			LEFT JOIN album_artist ON album_artist.artist_id = arts.id
		WHERE artists.id = arts.id AND song_artist.artist_id IS NULL AND album_artist.artist_id IS NULL`)
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
		return nil, err
	}
	return ids, nil
}

func (a artistRepository) FindByID(ctx context.Context, id string, include repos.IncludeArtistInfo) (*repos.CompleteArtist, error) {
	q := bqb.New("SELECT ? FROM artists ? WHERE artists.id = ?", genArtistSelectList(include), genArtistJoins(include), id)
	return getQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) FindByNames(ctx context.Context, names []string, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	if len(names) == 0 {
		return []*repos.CompleteArtist{}, nil
	}
	q := bqb.New("SELECT ? FROM artists ? WHERE artists.name IN (?)", genArtistSelectList(include), genArtistJoins(include), names)
	return selectQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) FindAll(ctx context.Context, params repos.FindArtistsParams, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	q := bqb.New("SELECT ? FROM artists ?", genArtistSelectList(include), genArtistJoins(include))
	where := bqb.Optional("WHERE")
	if params.OnlyAlbumArtists {
		if !include.AlbumInfo {
			return nil, repos.NewError("onlyAlbumArtists only allowed if include.AlbumInfo is true", repos.ErrInvalidParams, nil)
		}
		where.And("COALESCE(aa.count, 0) > 0")
	}
	if params.UpdatedAfter != nil {
		where.And("artists.updated >= ?", *params.UpdatedAfter)
	}
	q = bqb.New("? ? ORDER BY lower(artists.name)", q, where)
	return selectQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) FindBySearch(ctx context.Context, query string, onlyAlbumArtists bool, paginate repos.Paginate, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	query = strings.ToLower(query)
	q := bqb.New("SELECT ? FROM artists ?", genArtistSelectList(include), genArtistJoins(include))

	conditions, orderBy := genSearch(query, "artists.search_text", "artists.name")

	q.Space("WHERE (?)", conditions)
	if onlyAlbumArtists {
		if !include.AlbumInfo {
			return nil, repos.NewError("onlyAlbumArtists only allowed if include.AlbumInfo is true", repos.ErrInvalidParams, nil)
		}
		q.And("COALESCE(aa.count, 0) > 0")
	}
	q = bqb.New("? ORDER BY ?", q, orderBy)
	paginate.Apply(q)
	return selectQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) FindStarred(ctx context.Context, paginate repos.Paginate, include repos.IncludeArtistInfo) ([]*repos.CompleteArtist, error) {
	if !include.Annotations || include.User == "" {
		return nil, repos.NewError("include.Annotations and include.AnnotationUser required", repos.ErrInvalidParams, nil)
	}
	q := bqb.New("SELECT ? FROM artists ?", genArtistSelectList(include), genArtistJoins(include))
	q.Space("WHERE artist_stars.created IS NOT NULL")
	q.Space("ORDER BY artist_stars.created DESC")
	paginate.Apply(q)
	return selectQuery[*repos.CompleteArtist](ctx, a.db, q)
}

func (a artistRepository) GetAlbums(ctx context.Context, id string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	q := bqb.New("SELECT ? FROM albums ?", genAlbumSelectList(include), genAlbumJoins(include))
	q.Space("INNER JOIN album_artist ON album_artist.album_id = albums.id")
	q.Space(`WHERE album_artist.artist_id = ?`, id)
	q.Space("ORDER BY albums.year DESC, albums.name")
	return execAlbumSelectMany(ctx, a.db, q, include)
}

func (a artistRepository) GetAppearsOnAlbums(ctx context.Context, id string, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	q := bqb.New("SELECT DISTINCT ? FROM albums ?", genAlbumSelectList(include), genAlbumJoins(include))
	q.Space("LEFT JOIN album_artist ON album_artist.album_id = albums.id")
	q.Space("INNER JOIN songs ON songs.album_id = albums.id")
	q.Space("INNER JOIN song_artist ON song_artist.song_id = songs.id")
	q.Space(`WHERE album_artist.artist_id != ? AND song_artist.artist_id = ? AND NOT EXISTS (
		SELECT album_id FROM album_artist WHERE album_artist.artist_id = ? AND album_artist.album_id = albums.id
	)`, id, id, id)
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

func (a artistRepository) GetInfo(ctx context.Context, artistID string) (*repos.ArtistInfo, error) {
	q := bqb.New("SELECT artists.id, artists.info_updated, artists.biography, artists.lastfm_url, artists.lastfm_mbid, artists.music_brainz_id FROM artists WHERE artists.id = ?", artistID)
	return getQuery[*repos.ArtistInfo](ctx, a.db, q)
}

func (a artistRepository) SetInfo(ctx context.Context, artistID string, params repos.SetArtistInfo) error {
	q := bqb.New("UPDATE artists SET info_updated=NOW(), biography=?, lastfm_url=?, lastfm_mbid=? WHERE id = ?", params.Biography, params.LastFMURL, params.LastFMMBID, artistID)
	return executeQueryExpectAffectedRows(ctx, a.db, q)
}

// helpers

func genArtistSelectList(include repos.IncludeArtistInfo) *bqb.Query {
	q := bqb.New(`artists.id, artists.name, artists.created, artists.updated, artists.music_brainz_id`)

	if include.AlbumInfo {
		q.Comma("COALESCE(aa.count, 0) AS album_count")
	}

	if include.Annotations {
		q.Comma("avgr.rating AS avg_rating")
		if include.User != "" {
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
		if include.User != "" {
			q.Space("LEFT JOIN artist_stars ON artist_stars.artist_id = artists.id AND artist_stars.user_name = ?", include.User)
			q.Space("LEFT JOIN artist_ratings ON artist_ratings.artist_id = artists.id AND artist_ratings.user_name = ?", include.User)
		}
		q.Space(`LEFT JOIN (
				SELECT artist_id, AVG(artist_ratings.rating) AS rating FROM artist_ratings GROUP BY artist_id
			) avgr ON avgr.artist_id = artists.id`)
	}

	return q
}
