package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
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

	err := h.DB.Transaction(r.Context(), func(tx repos.Tx) error {
		if submission {
			possibleConflicts, err := h.DB.Scrobble().FindPossibleConflicts(r.Context(), user, ids, times)
			if err != nil {
				return fmt.Errorf("find possible conflicts: %w", err)
			}
			if len(possibleConflicts) > 0 {
				newIds := make([]string, 0, len(ids))
				newTimes := make([]time.Time, 0, len(times))
				newDurationsMs := make([]*int, 0, len(durationsMs))
				for i, songID := range ids {
					var isConflict bool
					for _, c := range possibleConflicts {
						if c.SongID == songID && times[i].Compare(c.Time) == 0 {
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
		createScrobblesParams := make([]repos.CreateScrobbleParams, 0, len(ids))
		for i, id := range ids {
			if !submission {
				err := tx.Scrobble().DeleteNowPlaying(r.Context(), user)
				if err != nil {
					return fmt.Errorf("delete old now playing: %w", err)
				}
			}
			song, err := h.DB.Song().FindByID(r.Context(), id, repos.IncludeSongInfo{
				Album: true,
				Lists: true,
			})
			if err != nil {
				return fmt.Errorf("get song: %w", err)
			}

			shouldSubmit := !submission || durationsMs[i] == nil || *durationsMs[i] > 4*60*1000 || float64(*durationsMs[i]) > float64(song.Duration.ToStd().Milliseconds())*0.5

			var duration repos.NullDurationMS
			if durationsMs[i] != nil {
				d := repos.NewDurationMS(int64(*durationsMs[i]))
				duration = repos.NullDurationMS{
					Duration: d,
					Valid:    true,
				}
			}
			createScrobblesParams = append(createScrobblesParams, repos.CreateScrobbleParams{
				User:                    user,
				SongID:                  song.ID,
				AlbumID:                 song.AlbumID,
				Time:                    times[i],
				SongDuration:            song.Duration,
				Duration:                duration,
				SubmittedToListenBrainz: shouldSubmit,
				NowPlaying:              !submission,
			})
			if shouldSubmit {
				var artistName *string
				if len(song.Artists) > 0 {
					artistName = &song.Artists[0].Name
				}
				mbids := make([]string, 0, len(song.Artists))
				for _, a := range song.Artists {
					if a.MusicBrainzID != nil {
						mbids = append(mbids, *a.MusicBrainzID)
					}
				}
				listen := &listenbrainz.Listen{
					ListenedAt:  times[i],
					SongName:    song.Title,
					AlbumName:   song.AlbumName,
					SongMBID:    song.MusicBrainzID,
					AlbumMBID:   song.AlbumMusicBrainzID,
					ReleaseMBID: song.AlbumReleaseMBID,
					TrackNumber: song.Track,
					Duration:    &song.Duration,
					ArtistName:  artistName,
					ArtistMBIDs: mbids,
				}
				listens = append(listens, listen)
			}
		}

		var listenbrainzSuccess bool
		if len(listens) > 0 {
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
			createScrobblesParams[i].SubmittedToListenBrainz = createScrobblesParams[i].SubmittedToListenBrainz && listenbrainzSuccess
		}

		err := tx.Scrobble().CreateMultiple(r.Context(), createScrobblesParams)
		if err != nil {
			return fmt.Errorf("create scrobbles: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("scrobble: %w", err))
		}
		return
	}

	responses.New().EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/getnowplaying/
func (h *Handler) handleGetNowPlaying(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	dbSongs, err := h.DB.Scrobble().GetNowPlayingSongs(r.Context(), repos.IncludeSongInfoFull(query.Get("u")))
	if err != nil {
		log.Errorf("get now playing: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	entries := util.Map(dbSongs, func(s *repos.NowPlayingSong) *responses.NowPlayingEntry {
		return &responses.NowPlayingEntry{
			Song:       responses.NewSong(s.CompleteSong),
			Username:   s.User,
			MinutesAgo: int(time.Since(s.Time).Minutes()),
		}
	})

	res := responses.New()
	res.NowPlaying = &responses.NowPlaying{
		Entries: entries,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/setrating/
func (h *Handler) handleSetRating(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	id, ok := paramIDReq(w, r, "id")
	if !ok {
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
			err = h.DB.Song().RemoveRating(r.Context(), user, id)
		} else {
			err = h.DB.Song().SetRating(r.Context(), user, id, rating)
		}
	case crossonic.IDTypeAlbum:
		if rating == 0 {
			err = h.DB.Album().RemoveRating(r.Context(), user, id)
		} else {
			err = h.DB.Album().SetRating(r.Context(), user, id, rating)
		}
	case crossonic.IDTypeArtist:
		if rating == 0 {
			err = h.DB.Artist().RemoveRating(r.Context(), user, id)
		} else {
			err = h.DB.Artist().SetRating(r.Context(), user, id, rating)
		}
	}
	if err != nil {
		// TODO handle not found
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

		songIDs := make([]string, 0, len(ids))
		err := h.DB.Transaction(r.Context(), func(tx repos.Tx) error {
			for _, id := range ids {
				idType, ok := crossonic.GetIDType(id)
				if !ok {
					return fmt.Errorf("%w: unknown id type", repos.ErrNotFound)
				}
				var err error
				switch idType {
				case crossonic.IDTypeSong:
					songIDs = append(songIDs, id)
					if star {
						err = tx.Song().Star(r.Context(), user, id)
					} else {
						err = tx.Song().UnStar(r.Context(), user, id)
					}
				case crossonic.IDTypeAlbum:
					if star {
						err = tx.Album().Star(r.Context(), user, id)
					} else {
						err = tx.Album().UnStar(r.Context(), user, id)
					}
				case crossonic.IDTypeArtist:
					if star {
						err = tx.Artist().Star(r.Context(), user, id)
					} else {
						err = tx.Artist().UnStar(r.Context(), user, id)
					}
				}
				if err != nil {
					// TODO handle not found
					return fmt.Errorf("star: %w", err)
				}
			}

			err := tx.Song().RemoveLBFeedbackUpdated(r.Context(), user, songIDs)
			if err != nil {
				return fmt.Errorf("remove ListenBrainz feedback updated: %w", err)
			}

			return nil
		})
		if err != nil {
			if errors.Is(err, repos.ErrNotFound) {
				responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			} else {
				respondInternalErr(w, query.Get("f"), fmt.Errorf("(un)star: %w", err))
			}
			return
		}

		if len(songIDs) > 0 {
			songs, err := h.DB.Song().FindByIDs(r.Context(), songIDs, repos.IncludeSongInfo{
				Album: true,
				Lists: true,
			})
			if err != nil {
				respondInternalErr(w, query.Get("f"), fmt.Errorf("handle (un)star: %w", err))
				return
			}
			lbFeedback := make([]*listenbrainz.Feedback, 0, len(songs))
			feedbackMap := make(map[string]*listenbrainz.Feedback, len(songs))
			for _, s := range songs {
				score := listenbrainz.FeedbackScoreNone
				if star {
					score = listenbrainz.FeedbackScoreLove
				}
				var artistName *string
				if len(s.Artists) > 0 {
					artistName = &s.Artists[0].Name
				}
				feedback := &listenbrainz.Feedback{
					SongID:     s.ID,
					SongName:   s.Title,
					SongMBID:   s.MusicBrainzID,
					AlbumName:  s.AlbumName,
					Score:      score,
					ArtistName: artistName,
				}
				lbFeedback = append(lbFeedback, feedback)
				feedbackMap[s.ID] = feedback
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
		}

		res := responses.New()
		res.EncodeOrLog(w, query.Get("f"))
	}
}
