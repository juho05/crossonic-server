package postgres

import (
	"context"
	"fmt"
	"slices"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/nullism/bqb"
)

type playlistRepository struct {
	db executer
	tx func(ctx context.Context, fn func(p playlistRepository) error) error
}

func (p playlistRepository) Create(ctx context.Context, params repos.CreatePlaylistParams) (*repos.Playlist, error) {
	q := bqb.New("INSERT INTO playlists (id,name,created,updated,owner,public,comment) VALUES (?,?,NOW(),NOW(),?,?,?) RETURNING playlists.*", crossonic.GenIDPlaylist(), params.Name, params.Owner, params.Public, params.Comment)
	return getQuery[*repos.Playlist](ctx, p.db, q)
}

func (p playlistRepository) Update(ctx context.Context, user, id string, params repos.UpdatePlaylistParams) error {
	updateList := genUpdateList(map[string]repos.OptionalGetter{
		"name":    params.Name,
		"public":  params.Public,
		"comment": params.Comment,
	}, true)
	q := bqb.New("UPDATE playlists SET ? WHERE id = ? AND owner = ?", updateList, id, user)
	return executeQueryExpectAffectedRows(ctx, p.db, q)
}

func (p playlistRepository) FindByID(ctx context.Context, user, id string, include repos.IncludePlaylistInfo) (*repos.CompletePlaylist, error) {
	q := bqb.New("SELECT ? FROM playlists ?", genPlaylistSelectList(include), genPlaylistJoins(include))
	q.Space("WHERE playlists.id = ? AND (playlists.owner = ? OR playlists.public = true)", id, user)
	return getQuery[*repos.CompletePlaylist](ctx, p.db, q)
}

func (p playlistRepository) FindAll(ctx context.Context, user string, include repos.IncludePlaylistInfo) ([]*repos.CompletePlaylist, error) {
	q := bqb.New("SELECT ? FROM playlists ?", genPlaylistSelectList(include), genPlaylistJoins(include))
	q.Space("WHERE playlists.owner = ? OR playlists.public = true", user)
	q.Space("ORDER BY playlists.updated DESC, playlists.name ASC, playlists.created DESC")
	return selectQuery[*repos.CompletePlaylist](ctx, p.db, q)
}

func (p playlistRepository) GetTracks(ctx context.Context, id string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	q := bqb.New("SELECT ? FROM playlist_song JOIN songs ON songs.id = playlist_song.song_id ?", genSongSelectList(include), genSongJoins(include))
	q.Space("WHERE playlist_song.playlist_id = ?", id)
	q.Space("ORDER BY playlist_song.track")
	return execSongSelectMany(ctx, p.db, q, include)
}

func (p playlistRepository) AddTracks(ctx context.Context, id string, songIDs []string) error {
	if len(songIDs) == 0 {
		return nil
	}
	return wrapErr("", p.tx(ctx, func(p playlistRepository) error {
		maxTrackNr, err := p.getMaxTrackNr(ctx, id)
		if err != nil {
			return fmt.Errorf("get max track number: %w", err)
		}
		q := bqb.New("INSERT INTO playlist_song (playlist_id,song_id,track) VALUES")
		valueList := bqb.Optional("")
		for _, sID := range songIDs {
			maxTrackNr++
			valueList.Comma("(?,?,?)", id, sID, maxTrackNr)
		}
		q = bqb.New("? ?", q, valueList)
		err = executeQuery(ctx, p.db, q)
		if err != nil {
			return fmt.Errorf("insert playlist_songs: %w", err)
		}
		return nil
	}))
}

func (p playlistRepository) RemoveTracks(ctx context.Context, id string, trackNumbers []int) error {
	if len(trackNumbers) == 0 {
		return nil
	}
	return wrapErr("", p.tx(ctx, func(p playlistRepository) error {
		q := bqb.New("DELETE FROM playlist_song WHERE playlist_id = ? AND track IN (?)", id, trackNumbers)
		count, err := executeQueryCountAffectedRows(ctx, p.db, q)
		if err != nil {
			return fmt.Errorf("delete playlist_songs: %w", err)
		}
		if count != len(trackNumbers) {
			return fmt.Errorf("delete playlist_songs: %w", repos.ErrNotFound)
		}

		slices.Sort(trackNumbers)

		caseExpr := bqb.New("CASE")
		if len(trackNumbers) == 1 {
			caseExpr = bqb.New("track - 1")
		} else {
			for i := range trackNumbers {
				if i == len(trackNumbers)-1 {
					caseExpr.Space("ELSE track - ?", i+1)
					continue
				}
				caseExpr.Space("WHEN track < ? THEN track - ?", trackNumbers[i+1], i+1)
			}
			caseExpr.Space("END")
		}

		err = executeQuery(ctx, p.db, bqb.New("SET CONSTRAINTS playlist_song_pkey DEFERRED"))
		if err != nil {
			return fmt.Errorf("set track key to deferred: %w", err)
		}

		q = bqb.New(`UPDATE playlist_song SET track = ? WHERE playlist_id = ? AND track > ?`, caseExpr, id, trackNumbers[0])
		err = executeQuery(ctx, p.db, q)
		if err != nil {
			return fmt.Errorf("update track numbers: %w", err)
		}
		return nil
	}))
}

func (p playlistRepository) ClearTracks(ctx context.Context, id string) error {
	q := bqb.New("DELETE FROM playlist_song WHERE playlist_id = ?", id)
	return executeQuery(ctx, p.db, q)
}

func (p playlistRepository) SetTracks(ctx context.Context, id string, songIDs []string) error {
	return wrapErr("", p.tx(ctx, func(p playlistRepository) error {
		err := p.ClearTracks(ctx, id)
		if err != nil {
			return fmt.Errorf("clear tracks: %w", err)
		}
		err = p.AddTracks(ctx, id, songIDs)
		if err != nil {
			return fmt.Errorf("add tracks: %w", err)
		}
		return nil
	}))
}

func (p playlistRepository) Delete(ctx context.Context, user, id string) error {
	q := bqb.New("DELETE FROM playlists WHERE id = ? AND owner = ?", id, user)
	return executeQueryExpectAffectedRows(ctx, p.db, q)
}

func (p playlistRepository) FixTrackNumbers(ctx context.Context) error {
	q := bqb.New(`UPDATE playlist_song SET track = t.new_track FROM
			(SELECT song_id,playlist_id,track,(row_number() OVER (PARTITION BY playlist_id ORDER BY track))-1 as new_track
		FROM playlist_song ORDER BY playlist_id) as t
		WHERE playlist_song.song_id = t.song_id AND playlist_song.playlist_id = t.playlist_id AND playlist_song.track != t.new_track`)
	return executeQuery(ctx, p.db, q)
}

func (p playlistRepository) getMaxTrackNr(ctx context.Context, id string) (int, error) {
	q := bqb.New("SELECT COALESCE(MAX(playlist_song.track), -1) FROM playlist_song WHERE playlist_song.playlist_id = ?", id)
	return getQuery[int](ctx, p.db, q)
}

func genPlaylistSelectList(include repos.IncludePlaylistInfo) *bqb.Query {
	q := bqb.New("playlists.id, playlists.name, playlists.created, playlists.updated, playlists.owner, playlists.public, playlists.comment")

	if include.TrackInfo {
		q.Comma("COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) as duration_ms")
	}

	return q
}

func genPlaylistJoins(include repos.IncludePlaylistInfo) *bqb.Query {
	q := bqb.Optional("")

	if include.TrackInfo {
		q.Space(`LEFT JOIN (
				SELECT playlist_song.playlist_id, COUNT(*) AS count, SUM(songs.duration_ms) AS duration_ms FROM playlist_song
				JOIN songs ON songs.id = playlist_song.song_id
				GROUP BY playlist_song.playlist_id
			) tracks ON tracks.playlist_id = playlists.id`)
	}

	return q
}
