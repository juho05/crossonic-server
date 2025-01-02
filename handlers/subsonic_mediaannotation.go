package handlers

import (
	"errors"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/db"
	sqlc "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/log"
)

// https://opensubsonic.netlify.app/docs/endpoints/scrobble/
func (h *Handler) handleScrobble(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	ids := query["id"]
	idTypes := make([]crossonic.IDType, len(ids))
	for i, id := range ids {
		idType, ok := crossonic.GetIDType(id)
		if !ok {
			responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
			return
		}
		if idType != crossonic.IDTypeSong {
			responses.EncodeError(w, query.Get("f"), "scrobbles are not supported for id type "+string(idType), responses.SubsonicErrorNotFound)
			return
		}
		idTypes[i] = idType
	}
	if len(ids) == 0 {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	timeStrs := query["time"]
	times := make([]time.Time, 0, len(ids))
	for i := range ids {
		if i < len(timeStrs) {
			timeInt, err := strconv.Atoi(timeStrs[i])
			if err != nil {
				responses.EncodeError(w, query.Get("f"), "invalid time value", responses.SubsonicErrorGeneric)
				return
			}
			times = append(times, time.UnixMilli(int64(timeInt)))
		} else {
			times = append(times, time.Now())
		}
	}
	durationMsStrs := query["duration_ms"]
	durationsMs := make([]*int, 0, len(durationMsStrs))
	for i := range ids {
		if i < len(durationMsStrs) {
			d, err := strconv.Atoi(durationMsStrs[i])
			if err != nil {
				responses.EncodeError(w, query.Get("f"), "invalid duration_ms value", responses.SubsonicErrorGeneric)
				return
			}
			if d > 0 {
				durationsMs = append(durationsMs, &d)
			} else {
				durationsMs = append(durationsMs, nil)
			}
		} else {
			durationsMs = append(durationsMs, nil)
		}
	}

	submissionStr := query.Get("submission")
	submission := true
	if submissionStr != "" {
		s, err := strconv.ParseBool(submissionStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid submission value", responses.SubsonicErrorGeneric)
			return
		}
		submission = s
	}

	if !submission && len(ids) != 1 {
		responses.EncodeError(w, query.Get("f"), "now playing scrobble requests must contain EXACTLY one id parameter", responses.SubsonicErrorGeneric)
		return
	}

	tx, err := h.Store.BeginTransaction(r.Context())
	if err != nil {
		log.Errorf("scrobble: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer tx.Rollback(r.Context())

	if submission {
		pgTimes := make([]pgtype.Timestamptz, len(times))
		for i, t := range times {
			pgTimes[i] = pgtype.Timestamptz{
				Time:  t,
				Valid: true,
			}
		}
		possibleConflicts, err := h.Store.FindPossibleScrobbleConflicts(r.Context(), sqlc.FindPossibleScrobbleConflictsParams{
			UserName: user,
			SongIds:  ids,
			Times:    pgTimes,
		})
		if err != nil {
			log.Errorf("scrobble: find possible conflicts: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		if len(possibleConflicts) > 0 {
			newIds := make([]string, 0, len(ids))
			newTimes := make([]time.Time, 0, len(times))
			newDurationsMs := make([]*int, 0, len(durationsMs))
			for i, songID := range ids {
				var isConflict bool
				for _, c := range possibleConflicts {
					if c.SongID == songID && times[i].Compare(c.Time.Time) == 0 {
						isConflict = true
						break
					}
				}
				if !isConflict {
					newIds = append(newIds, ids[i])
					newTimes = append(newTimes, times[i])
					newDurationsMs = append(newDurationsMs, durationsMs[i])
				}
			}
			ids = newIds
			times = newTimes
			durationsMs = newDurationsMs
		}
	}

	listens := make([]*listenbrainz.Listen, 0, len(ids))
	listenMap := make(map[string]*listenbrainz.Listen, len(ids))
	createScrobblesParams := make([]sqlc.CreateScrobblesParams, 0, len(ids))
	for i, id := range ids {
		if !submission {
			err = tx.DeleteNowPlaying(r.Context(), user)
			if err != nil {
				log.Errorf("scrobble: delete old now playing: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
				return
			}
		}
		song, err := h.Store.FindSong(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			} else {
				log.Errorf("scrobble: get song: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			}
			return
		}

		shouldSubmit := !submission || durationsMs[i] == nil || *durationsMs[i] > 4*60*1000 || float64(*durationsMs[i]) > float64(song.DurationMs)*0.5
		createScrobblesParams = append(createScrobblesParams, sqlc.CreateScrobblesParams{
			UserName: user,
			SongID:   song.ID,
			AlbumID:  song.AlbumID,
			Time: pgtype.Timestamptz{
				Time:  times[i],
				Valid: true,
			},
			SongDurationMs:          song.DurationMs,
			DurationMs:              intPtrToInt32Ptr(durationsMs[i]),
			SubmittedToListenbrainz: shouldSubmit,
			NowPlaying:              !submission,
		})
		if shouldSubmit {
			listen := &listenbrainz.Listen{
				ListenedAt:  times[i],
				SongName:    song.Title,
				AlbumName:   song.AlbumName,
				SongMBID:    song.MusicBrainzID,
				AlbumMBID:   song.AlbumMusicBrainzID,
				ReleaseMBID: song.AlbumReleaseMbid,
				TrackNumber: int32PtrToIntPtr(song.Track),
				DurationMS:  int32PtrToIntPtr(&song.DurationMs),
			}
			listens = append(listens, listen)
			listenMap[song.ID] = listen
		}
	}

	var listenbrainzSuccess bool
	if len(listens) > 0 {
		artists, err := tx.FindArtistRefsBySongs(r.Context(), ids)
		if err != nil {
			log.Errorf("scrobble: find artist refs: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		for _, a := range artists {
			listen, ok := listenMap[a.SongID]
			if !ok {
				continue
			}
			if listen.ArtistName == nil {
				listen.ArtistName = &a.Name
			}
			if a.MusicBrainzID != nil {
				listen.ArtistMBIDs = append(listen.ArtistMBIDs, *a.MusicBrainzID)
			}
		}

		if lbCon, err := h.ListenBrainz.GetListenbrainzConnection(r.Context(), user); err == nil {
			var err error
			if !submission {
				err = h.ListenBrainz.SubmitPlayingNow(r.Context(), lbCon, listens[0], query.Get("c"))
			} else {
				if len(listens) == 1 {
					err = h.ListenBrainz.SubmitSingle(r.Context(), lbCon, listens[0], query.Get("c"))
				} else {
					err = h.ListenBrainz.SubmitImport(r.Context(), lbCon, listens, query.Get("c"))
				}
			}
			listenbrainzSuccess = err == nil
			if !listenbrainzSuccess {
				log.Errorf("failed to scrobble to listenbrainz: %s", err)
			}
		} else if !errors.Is(err, listenbrainz.ErrUnauthenticated) {
			log.Errorf("failed to get listenbrainz connection for user %s: %s", user, err)
		}
	}

	for i := range createScrobblesParams {
		createScrobblesParams[i].SubmittedToListenbrainz = createScrobblesParams[i].SubmittedToListenbrainz && listenbrainzSuccess
	}

	_, err = tx.CreateScrobbles(r.Context(), createScrobblesParams)
	if err != nil {
		log.Errorf("scrobble: create scrobble(s): %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	err = tx.Commit(r.Context())
	if err != nil {
		log.Errorf("scrobble: commit: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	responses.New().EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/getnowplaying/
func (h *Handler) handleGetNowPlaying(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	songs, err := h.Store.GetNowPlayingSongs(r.Context(), query.Get("u"))
	if err != nil {
		log.Errorf("get now playing: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	songMap := make(map[string][]*responses.NowPlayingEntry, len(songs))
	songList := make([]*responses.NowPlayingEntry, 0, len(songs))
	songIDs := mapData(songs, func(s *sqlc.GetNowPlayingSongsRow) string {
		var starred *time.Time
		if s.Starred.Valid {
			starred = &s.Starred.Time
		}
		var averageRating *float64
		if s.AvgRating != 0 {
			avgRating := math.Round(s.AvgRating*100) / 100
			averageRating = &avgRating
		}
		fallbackGain := float32(db.GetFallbackGain(r.Context(), h.Store))
		song := responses.Song{
			ID:            s.ID,
			Title:         s.Title,
			Track:         int32PtrToIntPtr(s.Track),
			Year:          int32PtrToIntPtr(s.Year),
			CoverArt:      s.CoverID,
			Size:          s.Size,
			ContentType:   s.ContentType,
			Suffix:        filepath.Ext(s.Path),
			BitRate:       int(s.BitRate),
			SamplingRate:  int(s.SamplingRate),
			ChannelCount:  int(s.ChannelCount),
			Duration:      int(s.DurationMs) / 1000,
			UserRating:    int32PtrToIntPtr(s.UserRating),
			DiscNumber:    int32PtrToIntPtr(s.DiscNumber),
			Created:       s.Created.Time,
			AlbumID:       s.AlbumID,
			Album:         s.AlbumName,
			Type:          "music",
			MediaType:     "song",
			BPM:           int32PtrToIntPtr(s.Bpm),
			MusicBrainzID: s.MusicBrainzID,
			AverageRating: averageRating,
			Starred:       starred,
			ReplayGain: &responses.ReplayGain{
				TrackGain:    s.ReplayGain,
				AlbumGain:    s.AlbumReplayGain,
				TrackPeak:    s.ReplayGainPeak,
				AlbumPeak:    s.AlbumReplayGainPeak,
				FallbackGain: &fallbackGain,
			},
		}
		entry := &responses.NowPlayingEntry{
			Song:       song,
			Username:   s.UserName,
			MinutesAgo: int(time.Since(s.Time.Time).Minutes()),
		}
		if _, ok := songMap[song.ID]; !ok {
			songMap[song.ID] = make([]*responses.NowPlayingEntry, 0, 1)
		}
		songMap[song.ID] = append(songMap[song.ID], entry)
		songList = append(songList, entry)
		return s.ID
	})

	dbGenres, err := h.Store.FindGenresBySongs(r.Context(), songIDs)
	if err != nil {
		log.Errorf("get now playing: get genres: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, g := range dbGenres {
		songs := songMap[g.SongID]
		for _, song := range songs {
			if song.Genre == nil {
				song.Genre = &g.Name
			}
			song.Genres = append(song.Genres, &responses.GenreRef{
				Name: g.Name,
			})
		}
	}
	artists, err := h.Store.FindArtistRefsBySongs(r.Context(), songIDs)
	if err != nil {
		log.Errorf("get now playing: get artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, a := range artists {
		songs := songMap[a.SongID]
		for _, song := range songs {
			if song.ArtistID == nil && song.Artist == nil {
				song.ArtistID = &a.ID
				song.Artist = &a.Name
			}
			song.Artists = append(song.Artists, &responses.ArtistRef{
				ID:   a.ID,
				Name: a.Name,
			})
		}
	}
	albumArtists, err := h.Store.FindAlbumArtistRefsBySongs(r.Context(), songIDs)
	if err != nil {
		log.Errorf("get get now playing: get album artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, a := range albumArtists {
		songs := songMap[a.SongID]
		for _, song := range songs {
			if song.ArtistID == nil && song.Artist == nil {
				song.ArtistID = &a.ID
				song.Artist = &a.Name
			}
			song.AlbumArtists = append(song.AlbumArtists, &responses.ArtistRef{
				ID:   a.ID,
				Name: a.Name,
			})
		}
	}

	res := responses.New()
	res.NowPlaying = &responses.NowPlaying{
		Entries: songList,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/setrating/
func (h *Handler) handleSetRating(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	ratingStr := query.Get("rating")
	if ratingStr == "" {
		responses.EncodeError(w, query.Get("f"), "missing rating parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	idType, ok := crossonic.GetIDType(id)
	if !ok {
		responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
		return
	}

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 0 || rating > 5 {
		responses.EncodeError(w, query.Get("f"), "invalid rating parameter", responses.SubsonicErrorNotFound)
		return
	}

	switch idType {
	case crossonic.IDTypeSong:
		if rating == 0 {
			err = h.Store.RemoveSongRating(r.Context(), sqlc.RemoveSongRatingParams{
				UserName: user,
				SongID:   id,
			})
		} else {
			err = h.Store.SetSongRating(r.Context(), sqlc.SetSongRatingParams{
				SongID:   id,
				UserName: user,
				Rating:   int32(rating),
			})
		}
	case crossonic.IDTypeAlbum:
		if rating == 0 {
			err = h.Store.RemoveAlbumRating(r.Context(), sqlc.RemoveAlbumRatingParams{
				UserName: user,
				AlbumID:  id,
			})
		} else {
			err = h.Store.SetAlbumRating(r.Context(), sqlc.SetAlbumRatingParams{
				AlbumID:  id,
				UserName: user,
				Rating:   int32(rating),
			})
		}
	case crossonic.IDTypeArtist:
		if rating == 0 {
			err = h.Store.RemoveArtistRating(r.Context(), sqlc.RemoveArtistRatingParams{
				UserName: user,
				ArtistID: id,
			})
		} else {
			err = h.Store.SetArtistRating(r.Context(), sqlc.SetArtistRatingParams{
				ArtistID: id,
				UserName: user,
				Rating:   int32(rating),
			})
		}
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
				return
			}
		}
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	res := responses.New()
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/star/
func (h *Handler) handleStar(w http.ResponseWriter, r *http.Request) {
	h.handleStarUnstar(true)(w, r)
}

// https://opensubsonic.netlify.app/docs/endpoints/unstar/
func (h *Handler) handleUnstar(w http.ResponseWriter, r *http.Request) {
	h.handleStarUnstar(false)(w, r)
}

func (h *Handler) handleStarUnstar(star bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := getQuery(r)
		user := query.Get("u")

		var ids []string
		ids = append(ids, query["id"]...)
		ids = append(ids, query["albumId"]...)
		ids = append(ids, query["artistId"]...)

		tx, err := h.Store.BeginTransaction(r.Context())
		if err != nil {
			log.Errorf("(un)star: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		defer tx.Rollback(r.Context())

		songIDs := make([]string, 0, len(ids))
		for _, id := range ids {
			idType, ok := crossonic.GetIDType(id)
			if !ok {
				responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
				return
			}
			var err error
			switch idType {
			case crossonic.IDTypeSong:
				songIDs = append(songIDs, id)
				if star {
					err = tx.StarSong(r.Context(), sqlc.StarSongParams{
						SongID:   id,
						UserName: user,
					})
				} else {
					err = tx.UnstarSong(r.Context(), sqlc.UnstarSongParams{
						SongID:   id,
						UserName: user,
					})
				}
			case crossonic.IDTypeAlbum:
				if star {
					err = tx.StarAlbum(r.Context(), sqlc.StarAlbumParams{
						AlbumID:  id,
						UserName: user,
					})
				} else {
					err = tx.UnstarAlbum(r.Context(), sqlc.UnstarAlbumParams{
						AlbumID:  id,
						UserName: user,
					})
				}
			case crossonic.IDTypeArtist:
				if star {
					err = tx.StarArtist(r.Context(), sqlc.StarArtistParams{
						ArtistID: id,
						UserName: user,
					})
				} else {
					err = tx.UnstarArtist(r.Context(), sqlc.UnstarArtistParams{
						ArtistID: id,
						UserName: user,
					})
				}
			}
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) {
					if pgErr.Code == pgerrcode.ForeignKeyViolation {
						responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
						return
					}
				}
				log.Errorf("star: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
				return
			}
		}

		err = tx.RemoveLBFeedbackUpdated(r.Context(), sqlc.RemoveLBFeedbackUpdatedParams{
			UserName: user,
			SongIds:  songIDs,
		})
		if err != nil {
			log.Errorf("(un)star: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}

		if len(songIDs) > 0 {
			songs, err := tx.FindSongs(r.Context(), songIDs)
			if err != nil {
				log.Errorf("(un)star: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
				return
			}
			lbFeedback := make([]*listenbrainz.Feedback, 0, len(songs))
			feedbackMap := make(map[string]*listenbrainz.Feedback, len(songs))
			for _, s := range songs {
				score := listenbrainz.FeedbackScoreNone
				if star {
					score = listenbrainz.FeedbackScoreLove
				}
				feedback := &listenbrainz.Feedback{
					SongID:    s.ID,
					SongName:  s.Title,
					SongMBID:  s.MusicBrainzID,
					AlbumName: s.AlbumName,
					Score:     score,
				}
				lbFeedback = append(lbFeedback, feedback)
				feedbackMap[s.ID] = feedback
			}

			if len(feedbackMap) > 0 {
				artists, err := tx.FindArtistRefsBySongs(r.Context(), songIDs)
				if err != nil {
					log.Errorf("(un)star: %s", err)
					responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
					return
				}
				for _, a := range artists {
					feedback := feedbackMap[a.SongID]
					if feedback.ArtistName == nil {
						feedback.ArtistName = &a.Name
					}
				}
			}

			err = tx.Commit(r.Context())
			if err != nil {
				log.Errorf("(un)star: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
				return
			}

			lbCon, err := h.ListenBrainz.GetListenbrainzConnection(r.Context(), user)
			if err == nil {
				_, err = h.ListenBrainz.UpdateSongFeedback(r.Context(), lbCon, lbFeedback)
				if err != nil {
					log.Errorf("(un)star: %s", err)
				}
			} else {
				if !errors.Is(err, listenbrainz.ErrUnauthenticated) {
					log.Errorf("(un)star: %s", err)
				}
			}
		} else {
			err = tx.Commit(r.Context())
			if err != nil {
				log.Errorf("(un)star: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
				return
			}
		}

		res := responses.New()
		res.EncodeOrLog(w, query.Get("f"))
	}
}
