// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: playlists.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type AddPlaylistTracksParams struct {
	PlaylistID string
	SongID     string
	Track      int32
}

const checkPlaylistExists = `-- name: CheckPlaylistExists :one
SELECT EXISTS (SELECT id FROM playlists WHERE id = $1 AND owner = $2)
`

type CheckPlaylistExistsParams struct {
	ID    string
	Owner string
}

func (q *Queries) CheckPlaylistExists(ctx context.Context, arg CheckPlaylistExistsParams) (bool, error) {
	row := q.db.QueryRow(ctx, checkPlaylistExists, arg.ID, arg.Owner)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

const clearPlaylist = `-- name: ClearPlaylist :exec
DELETE FROM playlist_song WHERE playlist_id = $1
`

func (q *Queries) ClearPlaylist(ctx context.Context, playlistID string) error {
	_, err := q.db.Exec(ctx, clearPlaylist, playlistID)
	return err
}

const createPlaylist = `-- name: CreatePlaylist :exec
INSERT INTO playlists (id,name,created,updated,owner,public,comment)
VALUES ($1,$2,NOW(),NOW(),$3,$4,$5)
`

type CreatePlaylistParams struct {
	ID      string
	Name    string
	Owner   string
	Public  bool
	Comment *string
}

func (q *Queries) CreatePlaylist(ctx context.Context, arg CreatePlaylistParams) error {
	_, err := q.db.Exec(ctx, createPlaylist,
		arg.ID,
		arg.Name,
		arg.Owner,
		arg.Public,
		arg.Comment,
	)
	return err
}

const deletePlaylist = `-- name: DeletePlaylist :execresult
DELETE FROM playlists WHERE id = $1 AND owner = $2
`

type DeletePlaylistParams struct {
	ID    string
	Owner string
}

func (q *Queries) DeletePlaylist(ctx context.Context, arg DeletePlaylistParams) (pgconn.CommandTag, error) {
	return q.db.Exec(ctx, deletePlaylist, arg.ID, arg.Owner)
}

const findPlaylist = `-- name: FindPlaylist :one
SELECT playlists.id, playlists.name, playlists.created, playlists.updated, playlists.owner, playlists.public, playlists.comment, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) FROM playlists
LEFT JOIN (
  SELECT playlist_song.playlist_id, COUNT(*) AS count, SUM(songs.duration_ms) AS duration_ms FROM playlist_song
  JOIN songs ON songs.id = playlist_song.song_id
  GROUP BY playlist_song.playlist_id
) tracks ON tracks.playlist_id = playlists.id
WHERE playlists.id = $1 AND (playlists.owner = $2 OR playlists.public = true)
`

type FindPlaylistParams struct {
	ID   string
	User string
}

type FindPlaylistRow struct {
	ID         string
	Name       string
	Created    pgtype.Timestamptz
	Updated    pgtype.Timestamptz
	Owner      string
	Public     bool
	Comment    *string
	TrackCount int64
	DurationMs int64
}

func (q *Queries) FindPlaylist(ctx context.Context, arg FindPlaylistParams) (*FindPlaylistRow, error) {
	row := q.db.QueryRow(ctx, findPlaylist, arg.ID, arg.User)
	var i FindPlaylistRow
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Created,
		&i.Updated,
		&i.Owner,
		&i.Public,
		&i.Comment,
		&i.TrackCount,
		&i.DurationMs,
	)
	return &i, err
}

const findPlaylists = `-- name: FindPlaylists :many
SELECT playlists.id, playlists.name, playlists.created, playlists.updated, playlists.owner, playlists.public, playlists.comment, COALESCE(tracks.count, 0) AS track_count, COALESCE(tracks.duration_ms, 0) FROM playlists
LEFT JOIN (
  SELECT playlist_song.playlist_id, COUNT(*) AS count, SUM(songs.duration_ms) AS duration_ms FROM playlist_song
  JOIN songs ON songs.id = playlist_song.song_id
  GROUP BY playlist_song.playlist_id
) tracks ON tracks.playlist_id = playlists.id
WHERE playlists.owner = $1 OR playlists.public = true
ORDER BY playlists.updated DESC, playlists.name ASC, playlists.created DESC
`

type FindPlaylistsRow struct {
	ID         string
	Name       string
	Created    pgtype.Timestamptz
	Updated    pgtype.Timestamptz
	Owner      string
	Public     bool
	Comment    *string
	TrackCount int64
	DurationMs int64
}

func (q *Queries) FindPlaylists(ctx context.Context, user string) ([]*FindPlaylistsRow, error) {
	rows, err := q.db.Query(ctx, findPlaylists, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*FindPlaylistsRow
	for rows.Next() {
		var i FindPlaylistsRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Created,
			&i.Updated,
			&i.Owner,
			&i.Public,
			&i.Comment,
			&i.TrackCount,
			&i.DurationMs,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPlaylistOwner = `-- name: GetPlaylistOwner :one
SELECT owner FROM playlists WHERE id = $1
`

func (q *Queries) GetPlaylistOwner(ctx context.Context, id string) (string, error) {
	row := q.db.QueryRow(ctx, getPlaylistOwner, id)
	var owner string
	err := row.Scan(&owner)
	return owner, err
}

const getPlaylistTrackNumbers = `-- name: GetPlaylistTrackNumbers :many
SELECT track FROM playlist_song WHERE playlist_id = $1 ORDER BY track
`

func (q *Queries) GetPlaylistTrackNumbers(ctx context.Context, playlistID string) ([]int32, error) {
	rows, err := q.db.Query(ctx, getPlaylistTrackNumbers, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []int32
	for rows.Next() {
		var track int32
		if err := rows.Scan(&track); err != nil {
			return nil, err
		}
		items = append(items, track)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPlaylistTracks = `-- name: GetPlaylistTracks :many
SELECT songs.id, songs.path, songs.album_id, songs.title, songs.track, songs.year, songs.size, songs.content_type, songs.duration_ms, songs.bit_rate, songs.sampling_rate, songs.channel_count, songs.disc_number, songs.created, songs.updated, songs.bpm, songs.music_brainz_id, songs.replay_gain, songs.replay_gain_peak, songs.lyrics, songs.cover_id, albums.name as album_name, albums.replay_gain as album_replay_gain, albums.replay_gain_peak as album_replay_gain_peak, song_stars.created as starred, song_ratings.rating AS user_rating, COALESCE(avgr.rating, 0) AS avg_rating
FROM playlist_song
JOIN songs ON songs.id = playlist_song.song_id
LEFT JOIN albums ON albums.id = songs.album_id
LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = $2
LEFT JOIN (
  SELECT song_id, AVG(song_ratings.rating) AS rating FROM song_ratings GROUP BY song_id
) avgr ON avgr.song_id = songs.id
LEFT JOIN song_ratings ON song_ratings.song_id = songs.id AND song_ratings.user_name = $2
WHERE playlist_song.playlist_id = $1
ORDER BY playlist_song.track
`

type GetPlaylistTracksParams struct {
	PlaylistID string
	UserName   string
}

type GetPlaylistTracksRow struct {
	ID                  string
	Path                string
	AlbumID             *string
	Title               string
	Track               *int32
	Year                *int32
	Size                int64
	ContentType         string
	DurationMs          int32
	BitRate             int32
	SamplingRate        int32
	ChannelCount        int32
	DiscNumber          *int32
	Created             pgtype.Timestamptz
	Updated             pgtype.Timestamptz
	Bpm                 *int32
	MusicBrainzID       *string
	ReplayGain          *float32
	ReplayGainPeak      *float32
	Lyrics              *string
	CoverID             *string
	AlbumName           *string
	AlbumReplayGain     *float32
	AlbumReplayGainPeak *float32
	Starred             pgtype.Timestamptz
	UserRating          *int32
	AvgRating           float64
}

func (q *Queries) GetPlaylistTracks(ctx context.Context, arg GetPlaylistTracksParams) ([]*GetPlaylistTracksRow, error) {
	rows, err := q.db.Query(ctx, getPlaylistTracks, arg.PlaylistID, arg.UserName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*GetPlaylistTracksRow
	for rows.Next() {
		var i GetPlaylistTracksRow
		if err := rows.Scan(
			&i.ID,
			&i.Path,
			&i.AlbumID,
			&i.Title,
			&i.Track,
			&i.Year,
			&i.Size,
			&i.ContentType,
			&i.DurationMs,
			&i.BitRate,
			&i.SamplingRate,
			&i.ChannelCount,
			&i.DiscNumber,
			&i.Created,
			&i.Updated,
			&i.Bpm,
			&i.MusicBrainzID,
			&i.ReplayGain,
			&i.ReplayGainPeak,
			&i.Lyrics,
			&i.CoverID,
			&i.AlbumName,
			&i.AlbumReplayGain,
			&i.AlbumReplayGainPeak,
			&i.Starred,
			&i.UserRating,
			&i.AvgRating,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removePlaylistTracks = `-- name: RemovePlaylistTracks :exec
DELETE FROM playlist_song WHERE playlist_id = $1 AND track = any($2::int[])
`

type RemovePlaylistTracksParams struct {
	PlaylistID string
	Tracks     []int32
}

func (q *Queries) RemovePlaylistTracks(ctx context.Context, arg RemovePlaylistTracksParams) error {
	_, err := q.db.Exec(ctx, removePlaylistTracks, arg.PlaylistID, arg.Tracks)
	return err
}

const updatePlaylist = `-- name: UpdatePlaylist :exec
UPDATE playlists SET name = $3, updated = NOW(), public = $4, comment = $5
WHERE id = $1 AND owner = $2
`

type UpdatePlaylistParams struct {
	ID      string
	Owner   string
	Name    string
	Public  bool
	Comment *string
}

func (q *Queries) UpdatePlaylist(ctx context.Context, arg UpdatePlaylistParams) error {
	_, err := q.db.Exec(ctx, updatePlaylist,
		arg.ID,
		arg.Owner,
		arg.Name,
		arg.Public,
		arg.Comment,
	)
	return err
}

const updatePlaylistName = `-- name: UpdatePlaylistName :execresult
UPDATE playlists SET name = $3, updated = NOW()
WHERE id = $1 AND owner = $2
`

type UpdatePlaylistNameParams struct {
	ID    string
	Owner string
	Name  string
}

func (q *Queries) UpdatePlaylistName(ctx context.Context, arg UpdatePlaylistNameParams) (pgconn.CommandTag, error) {
	return q.db.Exec(ctx, updatePlaylistName, arg.ID, arg.Owner, arg.Name)
}

const updatePlaylistTrackNumbers = `-- name: UpdatePlaylistTrackNumbers :exec
UPDATE playlist_song SET track = track + $2 WHERE playlist_song.playlist_id = $1 AND track >= $3 AND track <= $4
`

type UpdatePlaylistTrackNumbersParams struct {
	PlaylistID string
	Add        int32
	MinTrack   int32
	MaxTrack   int32
}

func (q *Queries) UpdatePlaylistTrackNumbers(ctx context.Context, arg UpdatePlaylistTrackNumbersParams) error {
	_, err := q.db.Exec(ctx, updatePlaylistTrackNumbers,
		arg.PlaylistID,
		arg.Add,
		arg.MinTrack,
		arg.MaxTrack,
	)
	return err
}