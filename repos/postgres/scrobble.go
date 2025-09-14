package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/nullism/bqb"
)

type scrobbleRepository struct {
	db executer
	tx func(ctx context.Context, fn func(s scrobbleRepository) error) error
}

func (s scrobbleRepository) CreateMultiple(ctx context.Context, params []repos.CreateScrobbleParams) error {
	if len(params) == 0 {
		return nil
	}
	q := bqb.New("INSERT INTO scrobbles (user_name,song_id,album_id,time,song_duration_ms,duration_ms,submitted_to_listenbrainz,now_playing)")
	valueList := bqb.Optional("")
	for _, p := range params {
		valueList.Comma("(?,?,?,?,?,?,?,?)", p.User, p.SongID, p.AlbumID, p.Time, p.SongDuration, p.Duration, p.SubmittedToListenBrainz, p.NowPlaying)
	}
	return executeQuery(ctx, s.db, bqb.New("? VALUES ? ON CONFLICT (user_name,song_id,time,now_playing) DO UPDATE SET duration_ms = excluded.duration_ms, album_id = excluded.album_id, song_duration_ms = excluded.song_duration_ms, now_playing = excluded.now_playing", q, valueList))
}

func (s scrobbleRepository) FindPossibleConflicts(ctx context.Context, user string, songIDs []string, times []time.Time) ([]*repos.Scrobble, error) {
	if len(songIDs) != len(times) {
		return nil, repos.NewError(fmt.Sprintf("songIDs (%d) and times (%d) length mismatch", len(songIDs), len(times)), repos.ErrInvalidParams, nil)
	}
	if len(songIDs) == 0 {
		return []*repos.Scrobble{}, nil
	}
	q := bqb.New("SELECT scrobbles.* FROM scrobbles WHERE user_name = ? AND now_playing = false AND song_id IN (?) AND time = any(?)", user, songIDs, times)
	return selectQuery[*repos.Scrobble](ctx, s.db, q)
}

func (s scrobbleRepository) DeleteNowPlaying(ctx context.Context, user string) error {
	q := bqb.New("DELETE FROM scrobbles WHERE now_playing = true AND (user_name = ? OR EXTRACT(EPOCH FROM (NOW() - time))*1000 > song_duration_ms*3)", user)
	return executeQuery(ctx, s.db, q)
}

func (s scrobbleRepository) GetNowPlaying(ctx context.Context, user string) (*repos.Scrobble, error) {
	q := bqb.New("SELECT scrobbles.* FROM scrobbles WHERE user_name = ? AND now_playing = true AND EXTRACT(EPOCH FROM (NOW() - time))*1000 < song_duration_ms*3", user)
	return getQuery[*repos.Scrobble](ctx, s.db, q)
}

func (s scrobbleRepository) GetNowPlayingSongs(ctx context.Context, include repos.IncludeSongInfo) ([]*repos.NowPlayingSong, error) {
	q := bqb.New("SELECT scrobbles.user_name,scrobbles.time,? FROM songs JOIN scrobbles ON scrobbles.song_id = songs.id ?", genSongSelectList(include), genSongJoins(include))
	q.Space("WHERE scrobbles.now_playing = true AND EXTRACT(EPOCH FROM (NOW() - time))*1000 < scrobbles.song_duration_ms*3")
	q.Space("ORDER BY scrobbles.time DESC")

	songs, err := selectQuery[*repos.NowPlayingSong](ctx, s.db, q)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}
	err = loadSongLists(ctx, s.db, util.Map(songs, func(s *repos.NowPlayingSong) *repos.CompleteSong {
		return s.CompleteSong
	}), include)
	if err != nil {
		return nil, fmt.Errorf("load song lists: %w", err)
	}
	return songs, nil
}

func (s scrobbleRepository) FindUnsubmittedLBScrobbles(ctx context.Context) ([]*repos.Scrobble, error) {
	q := bqb.New(`SELECT scrobbles.* FROM scrobbles JOIN users ON scrobbles.user_name = users.name
		WHERE users.listenbrainz_username IS NOT NULL AND now_playing = false AND submitted_to_listenbrainz = false AND (duration_ms >= 4*60*1000 OR duration_ms >= song_duration_ms*0.5)`)
	return selectQuery[*repos.Scrobble](ctx, s.db, q)
}

func (s scrobbleRepository) SetLBSubmittedByUsers(ctx context.Context, users []string) error {
	if len(users) == 0 {
		return nil
	}
	q := bqb.New(`UPDATE scrobbles SET submitted_to_listenbrainz = true
		WHERE user_name IN (?) AND now_playing = false AND (duration_ms >= 4*60*1000 OR duration_ms >= song_duration_ms*0.5)`, users)
	return executeQuery(ctx, s.db, q)
}

func (s scrobbleRepository) GetDurationSum(ctx context.Context, user string, start, end time.Time) (repos.DurationMS, error) {
	q := bqb.New("SELECT COALESCE(SUM(duration_ms), 0) FROM scrobbles WHERE user_name = ? AND duration_ms IS NOT NULL AND now_playing = false AND time >= ? AND time < ?", user, start, end)
	return getQuery[repos.DurationMS](ctx, s.db, q)
}

func (s scrobbleRepository) GetDistinctSongCount(ctx context.Context, user string, start, end time.Time) (int, error) {
	q := bqb.New("SELECT COALESCE(COUNT(DISTINCT song_id), 0) FROM scrobbles WHERE user_name = ? AND now_playing = false AND time >= ? AND time < ?", user, start, end)
	return getQuery[int](ctx, s.db, q)
}

func (s scrobbleRepository) GetDistinctAlbumCount(ctx context.Context, user string, start, end time.Time) (int, error) {
	q := bqb.New("SELECT COALESCE(COUNT(DISTINCT album_id), 0) FROM scrobbles WHERE user_name = ? AND now_playing = false AND time >= ? AND time < ?", user, start, end)
	return getQuery[int](ctx, s.db, q)
}

func (s scrobbleRepository) GetDistinctArtistCount(ctx context.Context, user string, start, end time.Time) (int, error) {
	q := bqb.New(`SELECT COALESCE(COUNT(DISTINCT song_artist.artist_id), 0) FROM scrobbles
		INNER JOIN song_artist ON scrobbles.song_id = song_artist.song_id
		WHERE scrobbles.user_name = ? AND scrobbles.now_playing = false AND scrobbles.time >= ? AND scrobbles.time < ?`, user, start, end)
	return getQuery[int](ctx, s.db, q)
}

func (s scrobbleRepository) GetTopSongsByDuration(ctx context.Context, user string, start, end time.Time, paginate repos.Paginate, include repos.IncludeSongInfo) ([]*repos.ScrobbleTopSong, error) {
	q := bqb.New("SELECT SUM(scrobbles.duration_ms) as total_duration_ms,? FROM scrobbles INNER JOIN songs ON scrobbles.song_id = songs.id ?", genSongSelectList(include), genSongJoins(include))
	q.Space("WHERE scrobbles.user_name = ? AND scrobbles.now_playing = false AND scrobbles.time >= ? AND scrobbles.time < ?", user, start, end)
	q.Space("GROUP BY songs.id, albums.id, song_stars.created, song_ratings.rating, avgr.rating")
	q.Space("ORDER BY SUM(scrobbles.duration_ms) DESC")

	paginate.Apply(q)

	songs, err := selectQuery[*repos.ScrobbleTopSong](ctx, s.db, q)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}
	err = loadSongLists(ctx, s.db, util.Map(songs, func(s *repos.ScrobbleTopSong) *repos.CompleteSong {
		return s.CompleteSong
	}), include)
	if err != nil {
		return nil, fmt.Errorf("load song lists: %w", err)
	}
	return songs, nil
}
