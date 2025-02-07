package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/nullism/bqb"
)

type albumRepository struct {
	db executer
	tx func(ctx context.Context, fn func(a albumRepository) error) error
}

func (a albumRepository) Create(ctx context.Context, params repos.CreateAlbumParams) (string, error) {
	id := crossonic.GenIDAlbum()
	q := bqb.New(`INSERT INTO albums (id, name, created, updated, year, record_labels, music_brainz_id, release_mbid,
		release_types, is_compilation, replay_gain, replay_gain_peak) VALUES (?, ?, NOW(), NOW(), ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING albums.*`,
		id, params.Name, params.Year, params.RecordLabels, params.MusicBrainzID, params.ReleaseMBID, params.ReleaseTypes,
		params.IsCompilation, params.ReplayGain, params.ReplayGainPeak)
	return id, executeQuery(ctx, a.db, q)
}

func (a albumRepository) Update(ctx context.Context, id string, params repos.UpdateAlbumParams) error {
	updateList := genUpdateList(map[string]repos.OptionalGetter{
		"name":             params.Name,
		"year":             params.Year,
		"record_labels":    params.RecordLabels,
		"music_brainz_id":  params.MusicBrainzID,
		"release_mbid":     params.ReleaseMBID,
		"release_types":    params.ReleaseTypes,
		"is_compilation":   params.IsCompilation,
		"replay_gain":      params.ReplayGain,
		"replay_gain_peak": params.ReplayGainPeak,
	}, true)
	q := bqb.New("UPDATE albums SET ? WHERE id = ?", updateList, id)
	return executeQueryExpectAffectedRows(ctx, a.db, q)
}

func (a albumRepository) DeleteIfNoTracks(ctx context.Context) error {
	q := bqb.New("DELETE FROM albums USING albums AS albs LEFT JOIN songs ON albs.id = songs.album_id WHERE albums.id = albs.id AND songs.id IS NULL")
	return executeQuery(ctx, a.db, q)
}

func (a albumRepository) FindByID(ctx context.Context, id string, include repos.IncludeAlbumInfo) (*repos.CompleteAlbum, error) {
	q := bqb.New("SELECT ? FROM albums ? WHERE albums.id = ?", genAlbumSelectList(include), genAlbumJoins(include), id)
	return execAlbumSelectOne(ctx, a.db, q, include)
}

func (a albumRepository) FindAll(ctx context.Context, params repos.FindAlbumParams, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	q := bqb.New("SELECT ? FROM albums ?", genAlbumSelectList(include), genAlbumJoins(include))

	where := bqb.Optional("WHERE")
	if params.FromYear != nil {
		where.And("(albums.year IS NOT NULL AND albums.year >= ?)", *params.FromYear)
	}
	if params.ToYear != nil {
		where.And("(albums.year IS NOT NULL AND albums.year <= ?)", *params.ToYear)
	}
	if len(params.Genres) > 0 {
		genres := util.Map(params.Genres, func(g string) string {
			return strings.ToLower(g)
		})
		where.And(`(EXISTS(
				SELECT songs.id FROM songs
				JOIN song_genre ON songs.id = song_genre.song_id
				WHERE songs.album_id = albums.id AND lower(song_genre.genre_name) IN (?)
			))`, genres)
	}

	orderBy := bqb.Optional("ORDER BY")
	switch params.SortBy {
	case repos.FindAlbumSortByName:
		orderBy.Space("lower(albums.name)")
	case repos.FindAlbumSortByCreated:
		orderBy.Space("albums.created DESC, lower(albums.name)")
	case repos.FindAlbumSortByRating:
		orderBy.Space("COALESCE(album_ratings.rating, 0), lower(albums.name)")
		if !include.Annotations || include.User == "" {
			return nil, repos.NewError("find all albums ordered by rating requires include.Annotations and include.User to be set", repos.ErrInvalidParams, nil)
		}
	case repos.FindAlbumSortByStarred:
		orderBy.Space("album_stars.created DESC, lower(albums.name)")
		where.And("(album_stars.created IS NOT NULL)")
		if !include.Annotations || include.User == "" {
			return nil, repos.NewError("find all albums ordered by starred requires include.Annotations and include.User to be set", repos.ErrInvalidParams, nil)
		}
	case repos.FindAlbumSortRandom:
		orderBy.Space("random()")
	case repos.FindAlbumSortByYear:
		orderBy.Space("albums.year, lower(albums.name)")
		where.And("(albums.year IS NOT NULL)")
	case repos.FindAlbumSortByFrequent:
		if !include.PlayInfo || include.User == "" {
			return nil, repos.NewError("find all albums ordered by frequency requires include.PlayInfo and include.User to be set", repos.ErrInvalidParams, nil)
		}
		orderBy.Space("COALESCE(plays.count, 0) DESC, lower(albums.name)")
	case repos.FindAlbumSortByRecent:
		if !include.PlayInfo || include.User == "" {
			return nil, repos.NewError("find all albums ordered by last played requires include.PlayInfo and include.User to be set", repos.ErrInvalidParams, nil)
		}
		orderBy.Space("plays.last_played DESC NULLS LAST, lower(albums.name)")
	}

	q = bqb.New("? ? ?", q, where, orderBy)
	params.Paginate.Apply(q)
	return execAlbumSelectMany(ctx, a.db, q, include)
}

func (a albumRepository) FindBySearch(ctx context.Context, query string, paginate repos.Paginate, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	query = strings.ToLower(query)
	q := bqb.New("SELECT ? FROM albums ?", genAlbumSelectList(include), genAlbumJoins(include))
	q.Space("WHERE position(? in lower(albums.name)) > 0", query)
	q.Space("ORDER BY position(? in lower(albums.name)), lower(albums.name)", query)
	paginate.Apply(q)
	return execAlbumSelectMany(ctx, a.db, q, include)
}

func (a albumRepository) FindStarred(ctx context.Context, paginate repos.Paginate, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	if !include.Annotations || include.User == "" {
		return nil, repos.NewError("include.Annotations and include.User required", repos.ErrInvalidParams, nil)
	}
	q := bqb.New("SELECT ? FROM albums ?", genAlbumSelectList(include), genAlbumJoins(include))
	q.Space("WHERE album_stars.created IS NOT NULL")
	q.Space("ORDER BY album_stars.created DESC")
	paginate.Apply(q)
	return execAlbumSelectMany(ctx, a.db, q, include)
}

func (s albumRepository) GetTracks(ctx context.Context, albumID string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	q := bqb.New("SELECT ? FROM songs ? WHERE songs.album_id = ? ORDER BY songs.disc_number, songs.track", genSongSelectList(include), genSongJoins(include), albumID)
	return execSongSelectMany(ctx, s.db, q, include)
}

func (a albumRepository) Star(ctx context.Context, user, albumID string) error {
	q := bqb.New("INSERT INTO album_stars (album_id, user_name, created) VALUES (?, ?, NOW()) ON CONFLICT(album_id,user_name) DO NOTHING", albumID, user)
	return executeQuery(ctx, a.db, q)
}

func (a albumRepository) UnStar(ctx context.Context, user, albumID string) error {
	q := bqb.New("DELETE FROM album_stars WHERE user_name = ? AND album_id = ?", user, albumID)
	return executeQuery(ctx, a.db, q)
}

func (a albumRepository) SetRating(ctx context.Context, user, albumID string, rating int) error {
	q := bqb.New("INSERT INTO album_ratings (album_id,user_name,rating) VALUES (?, ?, ?) ON CONFLICT(album_id,user_name) DO UPDATE SET rating = ?", albumID, user, rating, rating)
	return executeQuery(ctx, a.db, q)
}

func (a albumRepository) RemoveRating(ctx context.Context, user, albumID string) error {
	q := bqb.New("DELETE FROM album_ratings WHERE user_name = ? AND album_id = ?", user, albumID)
	return executeQuery(ctx, a.db, q)
}

func (a albumRepository) GetInfo(ctx context.Context, albumID string, after time.Time) (*repos.AlbumInfo, error) {
	q := bqb.New("SELECT albums.id, albums.info_updated, albums.description, albums.lastfm_url, albums.lastfm_mbid, albums.music_brainz_id FROM albums WHERE albums.id = ? AND (albums.info_updated IS NULL OR albums.info_updated > ?)", albumID, after)
	return getQuery[*repos.AlbumInfo](ctx, a.db, q)
}

func (a albumRepository) SetInfo(ctx context.Context, albumID string, params repos.SetAlbumInfo) error {
	q := bqb.New("UPDATE albums SET info_updated=NOW(), description=?, lastfm_url=?, lastfm_mbid=? WHERE id = ?", params.Description, params.LastFMURL, params.LastFMMBID, albumID)
	return executeQueryExpectAffectedRows(ctx, a.db, q)
}

func (a albumRepository) GetAllArtistConnections(ctx context.Context) ([]repos.AlbumArtistConnection, error) {
	q := bqb.New("SELECT album_artist.album_id, album_artist.artist_id FROM album_artist")
	return selectQuery[repos.AlbumArtistConnection](ctx, a.db, q)
}

func (a albumRepository) RemoveAllArtistConnections(ctx context.Context) error {
	q := bqb.New("DELETE FROM album_artist")
	return executeQuery(ctx, a.db, q)
}

func (a albumRepository) CreateArtistConnections(ctx context.Context, connections []repos.AlbumArtistConnection) error {
	if len(connections) == 0 {
		return nil
	}
	valueList := bqb.Optional("")
	for _, c := range connections {
		valueList.Comma("(?,?)", c.AlbumID, c.ArtistID)
	}
	q := bqb.New("INSERT INTO album_artist (album_id,artist_id) VALUES ? ON CONFLICT (album_id,artist_id) DO NOTHING", valueList)
	return executeQuery(ctx, a.db, q)
}

// helpers

func genAlbumSelectList(include repos.IncludeAlbumInfo) *bqb.Query {
	q := bqb.New(`albums.id, albums.name, albums.created, albums.updated, albums.year, albums.record_labels, albums.music_brainz_id, albums.release_mbid,
		albums.release_types, albums.is_compilation, albums.replay_gain, albums.replay_gain_peak`)

	if include.TrackInfo {
		q.Comma(`COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) AS duration_ms`)
	}

	if include.Annotations {
		q.Comma("avgr.rating AS avg_rating")
		if include.User != "" {
			q.Comma("album_stars.created as starred, album_ratings.rating AS user_rating")
		}
	}

	if include.PlayInfo && include.User != "" {
		q.Comma("COALESCE(plays.count, 0) as play_count, plays.last_played")
	}

	return q
}

func genAlbumJoins(include repos.IncludeAlbumInfo) *bqb.Query {
	q := bqb.Optional("")

	if include.TrackInfo {
		q.Space(`LEFT JOIN (
				SELECT album_id, COUNT(*) AS count, SUM(duration_ms) AS duration_ms FROM songs GROUP BY album_id
			) tracks ON tracks.album_id = albums.id`)
	}

	if include.Annotations {
		if include.User != "" {
			q.Space("LEFT JOIN album_stars ON album_stars.album_id = albums.id AND album_stars.user_name = ?", include.User)
			q.Space("LEFT JOIN album_ratings ON album_ratings.album_id = albums.id AND album_ratings.user_name = ?", include.User)
		}
		q.Space(`LEFT JOIN (
				SELECT album_id, AVG(album_ratings.rating) AS rating FROM album_ratings GROUP BY album_id
			) avgr ON avgr.album_id = albums.id`)
	}

	if include.PlayInfo && include.User != "" {
		q.Space(`LEFT JOIN (
			SELECT album_id, COUNT(*) as count, MAX(time) as last_played FROM scrobbles WHERE user_name = ? AND album_id IS NOT NULL GROUP BY (user_name, album_id)
		) plays ON plays.album_id = albums.id`, include.User)
	}

	return q
}

func execAlbumSelectOne(ctx context.Context, db executer, query *bqb.Query, include repos.IncludeAlbumInfo) (*repos.CompleteAlbum, error) {
	albums, err := execAlbumSelectMany(ctx, db, query, include)
	if err != nil {
		return nil, err
	}
	if len(albums) == 0 {
		return nil, repos.ErrNotFound
	}
	if len(albums) > 1 {
		return nil, repos.ErrTooMany
	}
	return albums[0], nil
}

func execAlbumSelectMany(ctx context.Context, db executer, query *bqb.Query, include repos.IncludeAlbumInfo) ([]*repos.CompleteAlbum, error) {
	albums, err := selectQuery[*repos.CompleteAlbum](ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}
	err = loadAlbumLists(ctx, db, albums, include)
	if err != nil {
		return nil, fmt.Errorf("load album lists: %w", err)
	}
	return albums, nil
}

func loadAlbumLists(ctx context.Context, db executer, albums []*repos.CompleteAlbum, include repos.IncludeAlbumInfo) error {
	if len(albums) == 0 {
		return nil
	}

	albumIDs := util.Map(albums, func(s *repos.CompleteAlbum) string {
		return s.ID
	})

	var genres map[string][]string
	var err error
	if include.Genres {
		genres, err = getAlbumGenres(ctx, db, albumIDs)
		if err != nil {
			return fmt.Errorf("get genres: %w", err)
		}
	}

	var artists map[string][]repos.ArtistRef
	if include.Artists {
		artists, err = getAlbumArtistRefs(ctx, db, albumIDs)
		if err != nil {
			return fmt.Errorf("get artist refs: %w", err)
		}
	}

	for _, a := range albums {
		a.AlbumLists = &repos.AlbumLists{
			Genres:  genres[a.ID],
			Artists: artists[a.ID],
		}
	}

	return nil
}

func getAlbumGenres(ctx context.Context, db executer, albumIDs []string) (map[string][]string, error) {
	q := bqb.New(`SELECT songs.album_id, song_genre.genre_name FROM songs
		JOIN song_genre ON song_genre.song_id = songs.id
		WHERE songs.album_id IN (?)
		GROUP BY songs.album_id, song_genre.genre_name`, albumIDs)

	type genre struct {
		AlbumID string `db:"album_id"`
		Name    string `db:"genre_name"`
	}

	genres, err := selectQuery[genre](ctx, db, q)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}

	genreMap := make(map[string][]string, len(albumIDs))
	for _, g := range genres {
		genreMap[g.AlbumID] = append(genreMap[g.AlbumID], g.Name)
	}
	return genreMap, nil
}

func getAlbumArtistRefs(ctx context.Context, db executer, albumIDs []string) (map[string][]repos.ArtistRef, error) {
	q := bqb.New(`SELECT album_artist.album_id, artists.id, artists.name, artists.music_brainz_id FROM album_artist
		JOIN artists ON album_artist.artist_id = artists.id
		WHERE album_artist.album_id IN (?)`, albumIDs)

	type artistRef struct {
		repos.ArtistRef
		AlbumID string `db:"album_id"`
	}

	artists, err := selectQuery[artistRef](ctx, db, q)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}

	artistMap := make(map[string][]repos.ArtistRef, len(albumIDs))
	for _, a := range artists {
		artistMap[a.AlbumID] = append(artistMap[a.AlbumID], a.ArtistRef)
	}
	return artistMap, nil
}
