package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/nullism/bqb"
)

type songRepository struct {
	db executer
	tx func(ctx context.Context, fn func(s songRepository) error) error
}

func (s songRepository) FindByID(ctx context.Context, id string, include repos.IncludeSongInfo) (*repos.CompleteSong, error) {
	q := bqb.New("SELECT ? FROM songs ? WHERE songs.id = ?", genSongSelectList(include), genSongJoins(include), id)
	return execSongSelectOne(ctx, s.db, q, include)
}

func (s songRepository) FindByIDs(ctx context.Context, ids []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	var result []*repos.CompleteSong
	err := s.tx(ctx, func(s songRepository) error {
		var err error
		result, err = selectBatch(ids, func(ids []string) ([]*repos.CompleteSong, error) {
			q := bqb.New("SELECT ? FROM songs ? WHERE songs.id IN (?)", genSongSelectList(include), genSongJoins(include), ids)
			return execSongSelectMany(ctx, s.db, q, include)
		})
		return err
	})
	return result, err
}

func (s songRepository) FindByMusicBrainzID(ctx context.Context, mbid string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	q := bqb.New("SELECT ? FROM songs ? WHERE songs.music_brainz_id = ?", genSongSelectList(include), genSongJoins(include), mbid)
	return execSongSelectMany(ctx, s.db, q, include)
}

func (s songRepository) FindByPath(ctx context.Context, path string, include repos.IncludeSongInfo) (*repos.CompleteSong, error) {
	q := bqb.New("SELECT ? FROM songs ? WHERE songs.path = ?", genSongSelectList(include), genSongJoins(include), path)
	return execSongSelectOne(ctx, s.db, q, include)
}

func (s songRepository) FindAllFiltered(ctx context.Context, filter repos.SongFindAllFilter, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	q := bqb.New("SELECT ? FROM songs ?", genSongSelectList(include), genSongJoins(include))

	where := bqb.Optional("WHERE")

	if len(filter.Genres) > 0 {
		genres := util.Map(filter.Genres, func(g string) string { return strings.ToLower(g) })
		where.And(`(songs.id IN (
				SELECT songs.id FROM songs
				JOIN song_genre ON songs.id = song_genre.song_id
				WHERE lower(song_genre.genre_name) IN (?)
			))`, genres)
	}
	if len(filter.ArtistIDs) > 0 {
		where.And(`(songs.id IN (
				SELECT songs.id FROM songs
				JOIN song_artist ON songs.id = song_artist.song_id
				WHERE song_artist.artist_id IN (?)
			))`, filter.ArtistIDs)
	}

	if filter.OnlyStarred {
		if !include.Annotations || include.User == "" {
			return nil, repos.NewError("find all songs filtered by starred requires include.Annotations and include.User to be set", repos.ErrInvalidParams, nil)
		}
		where.And("(song_stars.created IS NOT NULL)")
	}

	if filter.MinBPM != nil {
		where.And("(songs.bpm IS NOT NULL AND songs.bpm >= ?)", *filter.MinBPM)
	}
	if filter.MaxBPM != nil {
		where.And("(songs.bpm IS NOT NULL AND songs.bpm <= ?)", *filter.MaxBPM)
	}

	if filter.FromYear != nil {
		where.And("(songs.year IS NOT NULL AND songs.year >= ?)", *filter.FromYear)
	}
	if filter.ToYear != nil {
		where.And("(songs.year IS NOT NULL AND songs.year <= ?)", *filter.ToYear)
	}

	if len(filter.AlbumIDs) > 0 {
		where.And("(songs.album_id IN (?))", filter.AlbumIDs)
	}

	orderBy := bqb.Optional("ORDER BY")

	if filter.Order != nil {
		if (!include.PlayInfo || include.User == "") && (*filter.Order == repos.SongOrderLastPlayed || *filter.Order == repos.SongOrderPlayCount) {
			return nil, repos.NewError("find all songs ordered by play count/time requires include.PlayInfo and include.User to be set", repos.ErrInvalidParams, nil)
		}

		switch *filter.Order {
		case repos.SongOrderTitle:
			orderBy.Comma("songs.title")
		case repos.SongOrderRandom:
			if filter.RandomSeed != nil {
				orderBy.Comma("md5(songs.id||?)", *filter.RandomSeed)
			} else {
				orderBy.Comma("RANDOM()")
			}
		case repos.SongOrderReleaseDate:
			orderBy.Comma("songs.year")
		case repos.SongOrderAdded:
			orderBy.Comma("songs.created")
		case repos.SongOrderLastPlayed:
			orderBy.Comma("plays.last_played")
		case repos.SongOrderPlayCount:
			orderBy.Comma("COALESCE(plays.count, 0)")
		case repos.SongOrderStarred:
			if !include.Annotations || include.User == "" {
				return nil, repos.NewError("find all songs ordered by starred requires include.Annotations and include.User to be set", repos.ErrInvalidParams, nil)
			}
			orderBy.Comma("song_stars.created")
		case repos.SongOrderBPM:
			orderBy.Comma("songs.bpm")
		}

		if *filter.Order != repos.SongOrderRandom {
			if filter.OrderDesc {
				orderBy.Space("DESC")
			} else {
				orderBy.Space("ASC")
			}
			orderBy.Space("NULLS LAST")
		}
	}

	if filter.Search != "" {
		conditions, searchOrder := genSearch(filter.Search, "songs.search_text", "songs.title")
		where.And("(?)", conditions)
		orderBy.Comma("?", searchOrder)
	}

	q.Space("? ?", where, orderBy)

	filter.Paginate.Apply(q)

	return execSongSelectMany(ctx, s.db, q, include)
}

func (s songRepository) FindByTitle(ctx context.Context, title string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	q := bqb.New("SELECT ? FROM songs ? WHERE songs.title = ?", genSongSelectList(include), genSongJoins(include), title)
	return execSongSelectMany(ctx, s.db, q, include)
}

func (s songRepository) FindAllByPathOrMBID(ctx context.Context, paths []string, mbids []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	var songs []*repos.CompleteSong
	err := s.tx(ctx, func(s songRepository) error {
		var err error
		songs, err = selectBatch2(paths, mbids, func(paths []string, mbids []string) ([]*repos.CompleteSong, error) {
			condition := bqb.New("WHERE false")
			if len(paths) > 0 {
				condition.Or("songs.path IN (?)", paths)
			}
			if len(mbids) > 0 {
				condition.Or("(songs.music_brainz_id IS NOT NULL AND songs.music_brainz_id IN (?))", mbids)
			}
			q := bqb.New("SELECT ? FROM songs ? ?", genSongSelectList(include), genSongJoins(include), condition)
			return execSongSelectMany(ctx, s.db, q, include)
		})
		return err
	})
	return songs, err
}

func (s songRepository) FindNonExistentIDs(ctx context.Context, ids []string) ([]string, error) {
	var result []string
	err := s.tx(ctx, func(s songRepository) error {
		var err error
		result, err = selectBatch(ids, func(ids []string) ([]string, error) {
			valueList := bqb.Optional("")
			for _, id := range ids {
				valueList.Comma("(?)", id)
			}
			q := bqb.New("SELECT ids.id FROM (VALUES ?) AS ids(id) LEFT JOIN songs ON songs.id = ids.id WHERE songs.id IS NULL", valueList)
			return selectQuery[string](ctx, s.db, q)
		})
		return err
	})
	return result, err
}

func (s songRepository) FindPaths(ctx context.Context, updatedBefore time.Time, paginate repos.Paginate) ([]string, error) {
	q := bqb.New("SELECT songs.path FROM songs WHERE songs.updated <= ? ORDER BY songs.id", updatedBefore)
	paginate.Apply(q)
	return selectQuery[string](ctx, s.db, q)
}

func (s songRepository) DeleteByPaths(ctx context.Context, paths []string) error {
	return s.tx(ctx, func(s songRepository) error {
		return execBatch(paths, func(paths []string) error {
			q := bqb.New("DELETE FROM songs WHERE songs.path IN (?)", paths)
			return executeQuery(ctx, s.db, q)
		})
	})
}

func (s songRepository) GetStreamInfo(ctx context.Context, id string) (*repos.SongStreamInfo, error) {
	q := bqb.New("SELECT songs.path, songs.bit_rate, songs.content_type, songs.duration_ms, songs.channel_count FROM songs WHERE songs.id = ?", id)
	return getQuery[*repos.SongStreamInfo](ctx, s.db, q)
}

func (s songRepository) CreateAll(ctx context.Context, params []repos.CreateSongParams) error {
	return s.tx(ctx, func(s songRepository) error {
		return execBatch(params, func(params []repos.CreateSongParams) error {
			valueList := bqb.Optional("")
			for _, p := range params {
				var id string
				if p.ID != nil {
					id = *p.ID
				} else {
					id = crossonic.GenIDSong()
				}

				searchFields := []string{p.Title}
				if p.AlbumName != nil {
					searchFields = append(searchFields, *p.AlbumName)
				}
				searchFields = append(searchFields, p.ArtistNames...)
				searchText := util.NormalizeText(" " + strings.Join(searchFields, " ") + " ")

				valueList.Comma("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW(), ?, ?, ?, ?, ?, ?)", id, p.Path, p.AlbumID, p.Title, p.Track, p.Year, p.Size, p.ContentType, p.Duration,
					p.BitRate, p.SamplingRate, p.ChannelCount, p.Disc, p.BPM, p.MusicBrainzID, p.ReplayGain, p.ReplayGainPeak,
					p.Lyrics, searchText)
			}
			q := bqb.New(`INSERT INTO songs
		(id, path, album_id, title, track, year, size, content_type, duration_ms, bit_rate, sampling_rate, channel_count, disc_number, created, updated,
		bpm, music_brainz_id, replay_gain, replay_gain_peak, lyrics, search_text)
		VALUES ?`, valueList)
			return executeQuery(ctx, s.db, q)
		})
	})
}

func (s songRepository) TryUpdateAll(ctx context.Context, params []repos.UpdateSongAllParams) (int, error) {
	if len(params) == 0 {
		return 0, nil
	}

	var count int
	err := s.tx(ctx, func(s songRepository) error {
		return execBatch(params, func(params []repos.UpdateSongAllParams) error {
			valueList := bqb.Optional("")
			for _, p := range params {
				searchFields := []string{p.Title}
				if p.AlbumName != nil {
					searchFields = append(searchFields, *p.AlbumName)
				}
				searchFields = append(searchFields, p.ArtistNames...)
				searchText := util.NormalizeText(" " + strings.Join(searchFields, " ") + " ")
				valueList.Comma("(?::text,?::text,?::text,?::text,?::int,?::int,?::bigint,?::text,?::int,?::int,?::int,?::int,?::int,?::int,?,?::real,?::real,?::text,?::text)", p.ID, p.Path, p.AlbumID, p.Title, p.Track, p.Year, p.Size, p.ContentType, p.Duration, p.BitRate, p.SamplingRate,
					p.ChannelCount, p.Disc, p.BPM, p.MusicBrainzID, p.ReplayGain, p.ReplayGainPeak, p.Lyrics, searchText)
			}

			q := bqb.New(`UPDATE songs SET
					path=s.path,
					album_id=s.album_id,
					title=s.title,
					track=s.track,
					year=s.year,
					size=s.size,
					content_type=s.content_type,
					duration_ms=s.duration_ms,
					bit_rate=s.bit_rate,
					sampling_rate=s.sampling_rate,
					channel_count=s.channel_count,
					disc_number=s.disc_number,
					bpm=s.bpm,
					music_brainz_id=s.music_brainz_id,
					replay_gain=s.replay_gain,
					replay_gain_peak=s.replay_gain_peak,
					lyrics=s.lyrics,
					search_text=s.search_text,
					updated=NOW()
				FROM (VALUES ?) AS s(id,path,album_id,title,track,year,size,content_type,duration_ms,bit_rate,sampling_rate,channel_count,disc_number,
					bpm,music_brainz_id,replay_gain,replay_gain_peak,lyrics,search_text)
				WHERE songs.id = s.id`, valueList)
			c, err := executeQueryCountAffectedRows(ctx, s.db, q)
			if err != nil {
				return err
			}
			count += c
			return nil
		})
	})
	return count, err
}

func (s songRepository) DeleteLastUpdatedBefore(ctx context.Context, before time.Time) error {
	return executeQuery(ctx, s.db, bqb.New("DELETE FROM songs WHERE updated < ?", before))
}

func (s songRepository) DeleteArtistConnections(ctx context.Context, songIDs []string) error {
	if len(songIDs) == 0 {
		return nil
	}
	q := bqb.New("DELETE FROM song_artist WHERE song_artist.song_id IN (?)", songIDs)
	return executeQuery(ctx, s.db, q)
}

func (s songRepository) CreateArtistConnections(ctx context.Context, connections []repos.SongArtistConnection) error {
	return s.tx(ctx, func(s songRepository) error {
		return execBatch(connections, func(connections []repos.SongArtistConnection) error {
			valueList := bqb.Optional("")
			for _, c := range connections {
				valueList.Comma("(?,?,?)", c.SongID, c.ArtistID, c.Index)
			}
			q := bqb.New("INSERT INTO song_artist (song_id,artist_id,index) VALUES ? ON CONFLICT (song_id,artist_id) DO NOTHING", valueList)
			return executeQuery(ctx, s.db, q)
		})
	})
}

func (s songRepository) DeleteGenreConnections(ctx context.Context, songIDs []string) error {
	return s.tx(ctx, func(s songRepository) error {
		return execBatch(songIDs, func(songIDs []string) error {
			q := bqb.New("DELETE FROM song_genre WHERE song_id IN (?)", songIDs)
			return executeQuery(ctx, s.db, q)
		})
	})
}

func (s songRepository) CreateGenreConnections(ctx context.Context, connections []repos.SongGenreConnection) error {
	return s.tx(ctx, func(s songRepository) error {
		return execBatch(connections, func(connections []repos.SongGenreConnection) error {
			valueList := bqb.Optional("")
			for _, c := range connections {
				valueList.Comma("(?,?)", c.SongID, c.Genre)
			}
			q := bqb.New("INSERT INTO song_genre (song_id,genre_name) VALUES ? ON CONFLICT (song_id,genre_name) DO NOTHING", valueList)
			return executeQuery(ctx, s.db, q)
		})
	})
}

func (s songRepository) Star(ctx context.Context, user, songID string) error {
	q := bqb.New("INSERT INTO song_stars (song_id, user_name, created) VALUES (?, ?, NOW()) ON CONFLICT(song_id,user_name) DO NOTHING", songID, user)
	return executeQuery(ctx, s.db, q)
}

func (s songRepository) StarMultiple(ctx context.Context, user string, songIDs []string) (int, error) {
	var count int
	err := s.tx(ctx, func(s songRepository) error {
		return execBatch(songIDs, func(songIDs []string) error {
			valueList := bqb.Optional("")
			for _, s := range songIDs {
				valueList.Comma("(?, ?, NOW())", s, user)
			}
			q := bqb.New("INSERT INTO song_stars (song_id, user_name, created) VALUES ? ON CONFLICT(song_id,user_name) DO NOTHING", valueList)
			c, err := executeQueryCountAffectedRows(ctx, s.db, q)
			count += c
			return err
		})
	})
	return count, err
}

func (s songRepository) UnStar(ctx context.Context, user, songID string) error {
	q := bqb.New("DELETE FROM song_stars WHERE user_name = ? AND song_id = ?", user, songID)
	return executeQuery(ctx, s.db, q)
}

func (s songRepository) UnStarMultiple(ctx context.Context, user string, songIDs []string) (int, error) {
	var count int
	err := s.tx(ctx, func(s songRepository) error {
		return execBatch(songIDs, func(songIDs []string) error {
			q := bqb.New("DELETE FROM song_stars WHERE user_name = ? AND song_id IN (?)", user, songIDs)
			c, err := executeQueryCountAffectedRows(ctx, s.db, q)
			count += c
			return err
		})
	})
	return count, err
}

func (s songRepository) SetRating(ctx context.Context, user, songID string, rating int) error {
	q := bqb.New("INSERT INTO song_ratings (song_id,user_name,rating) VALUES (?, ?, ?) ON CONFLICT(song_id,user_name) DO UPDATE SET rating = ?", songID, user, rating, rating)
	return executeQuery(ctx, s.db, q)
}

func (s songRepository) RemoveRating(ctx context.Context, user, songID string) error {
	q := bqb.New("DELETE FROM song_ratings WHERE user_name = ? AND song_id = ?", user, songID)
	return executeQuery(ctx, s.db, q)
}

func (s songRepository) FindNotUploadedLBFeedback(ctx context.Context, user string, lbLovedMBIDs []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	include.Annotations = true
	if len(lbLovedMBIDs) == 0 {
		lbLovedMBIDs = append(lbLovedMBIDs, "\n") // prevent lbLovedMBIDs from being empty
	}
	q := bqb.New(`
		SELECT ? FROM songs ?
		LEFT JOIN lb_feedback_status ON lb_feedback_status.song_id = songs.id AND lb_feedback_status.user_name = ?
		WHERE
			(lb_feedback_status.uploaded IS NULL AND song_stars.created IS NOT NULL AND songs.music_brainz_id NOT IN (?))
			OR
			(lb_feedback_status.uploaded = false AND
				(
						song_stars.created IS NULL
						AND
						(songs.music_brainz_id IN (?) OR lb_feedback_status.remote_mbid IN (?))
					OR
						song_stars.created IS NOT NULL
						AND songs.music_brainz_id NOT IN (?)
						AND lb_feedback_status.remote_mbid NOT IN (?)
				)
			)
	`, genSongSelectList(include), genSongJoins(include), user, lbLovedMBIDs, lbLovedMBIDs, lbLovedMBIDs, lbLovedMBIDs, lbLovedMBIDs)
	return execSongSelectMany(ctx, s.db, q, include)
}

func (s songRepository) FindLocalOutdatedFeedbackByLB(ctx context.Context, user string, lbLovedMBIDs []string, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	include.Annotations = true
	if len(lbLovedMBIDs) == 0 {
		lbLovedMBIDs = append(lbLovedMBIDs, "\n") // prevent lbLovedMBIDs from being empty
	}
	q := bqb.New(`
		SELECT ? FROM songs ?
		LEFT JOIN lb_feedback_status ON lb_feedback_status.song_id = songs.id AND lb_feedback_status.user_name = ?
		WHERE
			(lb_feedback_status.uploaded IS NULL AND song_stars.created IS NULL AND songs.music_brainz_id IN (?))
			OR
			(lb_feedback_status.uploaded = true AND (song_stars.created IS NULL AND (songs.music_brainz_id IN (?) OR lb_feedback_status.remote_mbid IN (?)) OR song_stars.created IS NOT NULL AND songs.music_brainz_id NOT IN (?) AND lb_feedback_status.remote_mbid NOT IN (?)))
	`, genSongSelectList(include), genSongJoins(include), user, lbLovedMBIDs, lbLovedMBIDs, lbLovedMBIDs, lbLovedMBIDs, lbLovedMBIDs)
	return execSongSelectMany(ctx, s.db, q, include)
}

func (s songRepository) SetLBFeedbackUploaded(ctx context.Context, user string, params []repos.SongSetLBFeedbackUploadedParams, updateRemoteMBIDs bool) error {
	return s.tx(ctx, func(s songRepository) error {
		return execBatch(params, func(params []repos.SongSetLBFeedbackUploadedParams) error {
			valueList := bqb.Optional("")
			for _, p := range params {
				valueList.Comma("(?,?,?,?)", p.SongID, user, p.RemoteMBID, p.Uploaded)
			}
			q := bqb.New("INSERT INTO lb_feedback_status (song_id,user_name,remote_mbid,uploaded) VALUES ? ON CONFLICT (song_id,user_name) DO UPDATE SET uploaded = excluded.uploaded", valueList)
			if updateRemoteMBIDs {
				q.Comma("remote_mbid = excluded.remote_mbid")
			}
			return executeQuery(ctx, s.db, q)
		})
	})
}

func (s songRepository) SetLBFeedbackUploadedForAllMatchingStarredSongs(ctx context.Context, user string, lbLovedMBIDs []string) error {
	if len(lbLovedMBIDs) == 0 {
		return nil
	}
	q := bqb.New(`
		INSERT INTO lb_feedback_status (song_id,user_name,remote_mbid,uploaded)
			SELECT songs.id, song_stars.user_name, songs.music_brainz_id, true FROM songs
			LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = ?
			WHERE song_stars.created IS NOT NULL AND songs.music_brainz_id IN (?)
		ON CONFLICT (song_id,user_name) DO UPDATE SET uploaded = true`, user, lbLovedMBIDs)
	return executeQuery(ctx, s.db, q)
}

func (s songRepository) Count(ctx context.Context) (int, error) {
	return getQuery[int](ctx, s.db, bqb.New("SELECT COUNT(songs.id) FROM songs"))
}

func (s songRepository) GetMedianReplayGain(ctx context.Context) (float64, error) {
	return getQuery[float64](ctx, s.db, bqb.New("SELECT COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY songs.replay_gain), 0) FROM songs"))
}

// ================ helpers ================

func genSongSelectList(include repos.IncludeSongInfo) *bqb.Query {
	q := bqb.New(`songs.id, songs.path, songs.album_id, songs.title, songs.track, songs.year, songs.size, songs.content_type,
		songs.duration_ms, songs.bit_rate, songs.sampling_rate, songs.channel_count, songs.disc_number, songs.created, songs.updated,
		songs.bpm, songs.music_brainz_id, songs.replay_gain, songs.replay_gain_peak, songs.lyrics`)

	if include.Album {
		q.Comma(`albums.name as album_name, albums.replay_gain as album_replay_gain, albums.replay_gain_peak as album_replay_gain_peak,
		albums.music_brainz_id as album_music_brainz_id, albums.release_mbid as album_release_mbid`)
	}

	if include.Annotations {
		q.Comma("avgr.rating AS avg_rating")
		if include.User != "" {
			q.Comma("song_stars.created as starred, song_ratings.rating AS user_rating")
		}
	}

	if include.PlayInfo && include.User != "" {
		q.Comma("COALESCE(plays.count, 0) as play_count, plays.last_played")
	}
	return q
}

func genSongJoins(include repos.IncludeSongInfo) *bqb.Query {
	q := bqb.Optional("")

	if include.Album {
		q.Space("LEFT JOIN albums ON albums.id = songs.album_id")
	}

	if include.Annotations {
		if include.User != "" {
			q.Space("LEFT JOIN song_stars ON song_stars.song_id = songs.id AND song_stars.user_name = ?", include.User)
			q.Space("LEFT JOIN song_ratings ON song_ratings.song_id = songs.id AND song_ratings.user_name = ?", include.User)
		}
		q.Space(`LEFT JOIN (
				SELECT song_id, AVG(song_ratings.rating) AS rating FROM song_ratings GROUP BY song_id
			) avgr ON avgr.song_id = songs.id`)
	}

	if include.PlayInfo && include.User != "" {
		q.Space(`LEFT JOIN (
			SELECT song_id, COUNT(*) as count, MAX(time) as last_played FROM scrobbles WHERE user_name = ? AND now_playing = false AND (duration_ms IS NULL OR duration_ms >= 240000 OR duration_ms >= song_duration_ms/2) GROUP BY (user_name, song_id)
		) plays ON plays.song_id = songs.id`, include.User)
	}

	return q
}

func execSongSelectOne(ctx context.Context, db executer, query *bqb.Query, include repos.IncludeSongInfo) (*repos.CompleteSong, error) {
	songs, err := execSongSelectMany(ctx, db, query, include)
	if err != nil {
		return nil, err
	}
	if len(songs) == 0 {
		return nil, repos.ErrNotFound
	}
	if len(songs) > 1 {
		return nil, repos.ErrTooMany
	}
	return songs[0], nil
}

func execSongSelectMany(ctx context.Context, db executer, query *bqb.Query, include repos.IncludeSongInfo) ([]*repos.CompleteSong, error) {
	songs, err := selectQuery[*repos.CompleteSong](ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}
	err = loadSongLists(ctx, db, songs, include)
	if err != nil {
		return nil, fmt.Errorf("load song lists: %w", err)
	}
	return songs, nil
}

func loadSongLists(ctx context.Context, db executer, songs []*repos.CompleteSong, include repos.IncludeSongInfo) error {
	if len(songs) == 0 || !include.Lists {
		return nil
	}

	songIDs := util.Map(songs, func(s *repos.CompleteSong) string {
		return s.ID
	})

	genres, err := getSongGenres(ctx, db, songIDs)
	if err != nil {
		return fmt.Errorf("get genres: %w", err)
	}

	artists, err := getSongArtistRefs(ctx, db, songIDs)
	if err != nil {
		return fmt.Errorf("get artist refs: %w", err)
	}

	albumArtists, err := getSongAlbumArtistRefs(ctx, db, songIDs)
	if err != nil {
		return fmt.Errorf("get album artist refs: %w", err)
	}

	for _, s := range songs {
		s.SongLists = &repos.SongLists{
			Genres:       genres[s.ID],
			Artists:      artists[s.ID],
			AlbumArtists: albumArtists[s.ID],
		}
	}

	return nil
}

func getSongGenres(ctx context.Context, db executer, songIDs []string) (map[string][]string, error) {
	q := bqb.New(`SELECT song_genre.song_id, genres.name FROM song_genre
		JOIN genres ON song_genre.genre_name = genres.name
		WHERE song_genre.song_id IN (?)`, songIDs)

	type genre struct {
		SongID string `db:"song_id"`
		Name   string `db:"name"`
	}

	genres, err := selectQuery[genre](ctx, db, q)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}

	genreMap := make(map[string][]string, len(songIDs))
	for _, g := range genres {
		genreMap[g.SongID] = append(genreMap[g.SongID], g.Name)
	}
	return genreMap, nil
}

func getSongArtistRefs(ctx context.Context, db executer, songIDs []string) (map[string][]repos.ArtistRef, error) {
	q := bqb.New(`SELECT song_artist.song_id, artists.id, artists.name, artists.music_brainz_id FROM song_artist
		JOIN artists ON song_artist.artist_id = artists.id
		WHERE song_artist.song_id IN (?) ORDER BY song_artist.index`, songIDs)

	type artistRef struct {
		repos.ArtistRef
		SongID string `db:"song_id"`
	}

	artists, err := selectQuery[artistRef](ctx, db, q)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}

	artistMap := make(map[string][]repos.ArtistRef, len(songIDs))
	for _, a := range artists {
		artistMap[a.SongID] = append(artistMap[a.SongID], a.ArtistRef)
	}
	return artistMap, nil
}

func getSongAlbumArtistRefs(ctx context.Context, db executer, songIDs []string) (map[string][]repos.ArtistRef, error) {
	q := bqb.New(`SELECT songs.id as song_id, artists.id, artists.name FROM songs
		JOIN albums ON songs.album_id = albums.id
		JOIN album_artist ON album_artist.album_id = albums.id
		JOIN artists ON album_artist.artist_id = artists.id
		WHERE songs.id IN (?) ORDER BY album_artist.index`, songIDs)

	type artistRef struct {
		repos.ArtistRef
		SongID string `db:"song_id"`
	}

	artists, err := selectQuery[artistRef](ctx, db, q)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}

	artistMap := make(map[string][]repos.ArtistRef, len(songIDs))
	for _, a := range artists {
		artistMap[a.SongID] = append(artistMap[a.SongID], a.ArtistRef)
	}
	return artistMap, nil
}
