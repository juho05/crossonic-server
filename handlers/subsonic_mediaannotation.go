package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

// https://opensubsonic.netlify.app/docs/endpoints/scrobble/
func (h *Handler) handleScrobble(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	ids, ok := q.IDsTypeReq("id", []crossonic.IDType{crossonic.IDTypeSong})
	if !ok {
		return
	}

	timeInts, ok := q.Int64s("time")
	if !ok {
		return
	}
	times := make([]time.Time, 0, len(ids))
	for i := range ids {
		if i < len(timeInts) {
			times = append(times, time.UnixMilli(timeInts[i]))
		} else {
			times = append(times, time.Now())
		}
	}

	durationMsInts, ok := q.Ints("duration_ms")
	if !ok {
		return
	}
	durationsMs := make([]*int, 0, len(durationMsInts))
	for i := range ids {
		if i < len(durationMsInts) {
			d := durationMsInts[i]
			if d > 0 {
				durationsMs = append(durationsMs, &d)
			} else {
				durationsMs = append(durationsMs, nil)
			}
		} else {
			durationsMs = append(durationsMs, nil)
		}
	}

	submission, ok := q.BoolDef("submission", true)
	if !ok {
		return
	}

	if !submission && len(ids) != 1 {
		respondGenericErr(w, q.Format(), "now playing scrobble requests must contain exactly one id parameter")
		return
	}

	user, err := h.DB.User().FindByName(r.Context(), q.User())
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("scrobble: find user: %w", err))
		return
	}

	submitToListenBrainz := user.ListenBrainzUsername != nil && user.ListenBrainzScrobble

	err = h.DB.Transaction(r.Context(), func(tx repos.Tx) error {
		if submission {
			possibleConflicts, err := h.DB.Scrobble().FindPossibleConflicts(r.Context(), q.User(), ids, times)
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
		if !submission {
			err := tx.Scrobble().DeleteNowPlaying(r.Context(), q.User())
			if err != nil {
				return fmt.Errorf("delete old now playing: %w", err)
			}
		}
		for i, id := range ids {
			song, err := h.DB.Song().FindByID(r.Context(), id, repos.IncludeSongInfo{
				Album: true,
				Lists: true,
			})
			if err != nil {
				return fmt.Errorf("get song: %w", err)
			}

			shouldSubmit := submitToListenBrainz && (!submission || durationsMs[i] == nil || *durationsMs[i] > 4*60*1000 || float64(*durationsMs[i]) > float64(song.Duration.ToStd().Milliseconds())*0.5)

			var duration repos.NullDurationMS
			if durationsMs[i] != nil {
				d := repos.NewDurationMS(int64(*durationsMs[i]))
				duration = repos.NullDurationMS{
					Duration: d,
					Valid:    true,
				}
			}
			createScrobblesParams = append(createScrobblesParams, repos.CreateScrobbleParams{
				User:                    q.User(),
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
			if lbCon, err := h.ListenBrainz.GetListenbrainzConnection(r.Context(), q.User()); err == nil {
				var err error
				if !submission {
					err = h.ListenBrainz.SubmitPlayingNow(r.Context(), lbCon, listens[0], q.Client())
				} else {
					if len(listens) == 1 {
						err = h.ListenBrainz.SubmitSingle(r.Context(), lbCon, listens[0], q.Client())
					} else {
						err = h.ListenBrainz.SubmitImport(r.Context(), lbCon, listens, q.Client())
					}
				}
				listenbrainzSuccess = err == nil
				if !listenbrainzSuccess {
					log.Errorf("failed to scrobble to listenbrainz: %s", err)
				}
			} else if !errors.Is(err, listenbrainz.ErrUnauthenticated) {
				log.Errorf("failed to get listenbrainz connection for user %s: %s", q.User(), err)
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
		respondErr(w, q.Format(), fmt.Errorf("scrobble: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, q.Format())
}

// https://opensubsonic.netlify.app/docs/endpoints/getnowplaying/
func (h *Handler) handleGetNowPlaying(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	dbSongs, err := h.DB.Scrobble().GetNowPlayingSongs(r.Context(), repos.IncludeSongInfoFull(q.User()))
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get now playing: %w", err))
		return
	}

	entries := util.Map(dbSongs, func(s *repos.NowPlayingSong) *responses.NowPlayingEntry {
		return &responses.NowPlayingEntry{
			Song:       responses.NewSong(s.CompleteSong, h.Config),
			Username:   s.User,
			MinutesAgo: int(time.Since(s.Time).Minutes()),
		}
	})

	res := responses.New()
	res.NowPlaying = &responses.NowPlaying{
		Entries: entries,
	}
	res.EncodeOrLog(w, q.Format())
}

// https://opensubsonic.netlify.app/docs/endpoints/setrating/
func (h *Handler) handleSetRating(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	id, ok := q.IDTypeReq("id", []crossonic.IDType{crossonic.IDTypeSong, crossonic.IDTypeAlbum, crossonic.IDTypeArtist})
	if !ok {
		return
	}

	rating, ok := q.IntRangeReq("rating", 0, 5)
	if !ok {
		return
	}

	idType, ok := crossonic.GetIDType(id)
	if !ok {
		respondNotFoundErr(w, q.Format(), "unknown id type")
		return
	}

	var err error
	switch idType {
	case crossonic.IDTypeSong:
		if rating == 0 {
			err = h.DB.Song().RemoveRating(r.Context(), q.User(), id)
		} else {
			err = h.DB.Song().SetRating(r.Context(), q.User(), id, rating)
		}
	case crossonic.IDTypeAlbum:
		if rating == 0 {
			err = h.DB.Album().RemoveRating(r.Context(), q.User(), id)
		} else {
			err = h.DB.Album().SetRating(r.Context(), q.User(), id, rating)
		}
	case crossonic.IDTypeArtist:
		if rating == 0 {
			err = h.DB.Artist().RemoveRating(r.Context(), q.User(), id)
		} else {
			err = h.DB.Artist().SetRating(r.Context(), q.User(), id, rating)
		}
	}
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("set rating: %w", err))
		return
	}

	res := responses.New()
	res.EncodeOrLog(w, q.Format())
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
		q := getQuery(w, r)

		ids, ok := q.IDsTypeReq("id", []crossonic.IDType{crossonic.IDTypeSong, crossonic.IDTypeAlbum, crossonic.IDTypeArtist})
		if !ok {
			return
		}
		albumIDs, ok := q.IDsTypeReq("albumId", []crossonic.IDType{crossonic.IDTypeAlbum})
		if !ok {
			return
		}
		artistIDs, ok := q.IDsTypeReq("artistId", []crossonic.IDType{crossonic.IDTypeArtist})
		if !ok {
			return
		}

		allIDs := make([]string, 0, len(ids)+len(albumIDs)+len(artistIDs))
		allIDs = append(allIDs, ids...)
		allIDs = append(allIDs, albumIDs...)
		allIDs = append(allIDs, artistIDs...)

		songIDs := make([]string, 0, len(allIDs))
		err := h.DB.Transaction(r.Context(), func(tx repos.Tx) error {
			for _, id := range allIDs {
				idType, ok := crossonic.GetIDType(id)
				if !ok {
					return fmt.Errorf("%w: unknown id type", repos.ErrNotFound)
				}
				var err error
				switch idType {
				case crossonic.IDTypeSong:
					songIDs = append(songIDs, id)
					if star {
						err = tx.Song().Star(r.Context(), q.User(), id)
					} else {
						err = tx.Song().UnStar(r.Context(), q.User(), id)
					}
				case crossonic.IDTypeAlbum:
					if star {
						err = tx.Album().Star(r.Context(), q.User(), id)
					} else {
						err = tx.Album().UnStar(r.Context(), q.User(), id)
					}
				case crossonic.IDTypeArtist:
					if star {
						err = tx.Artist().Star(r.Context(), q.User(), id)
					} else {
						err = tx.Artist().UnStar(r.Context(), q.User(), id)
					}
				}
				if err != nil {
					// TODO handle not found
					return fmt.Errorf("star: %w", err)
				}
			}

			err := tx.Song().SetLBFeedbackUploaded(r.Context(), q.User(), util.Map(songIDs, func(id string) repos.SongSetLBFeedbackUploadedParams {
				return repos.SongSetLBFeedbackUploadedParams{
					SongID:     id,
					RemoteMBID: nil,
					Uploaded:   false,
				}
			}), false)
			if err != nil {
				return fmt.Errorf("set lb_feedback_status to uploaded = false: %w", err)
			}

			return nil
		})
		if err != nil {
			if errors.Is(err, repos.ErrNotFound) {
				respondNotFoundErr(w, q.Format(), "")
			} else {
				respondInternalErr(w, q.Format(), fmt.Errorf("(un)star: %w", err))
			}
			return
		}

		user, err := h.DB.User().FindByName(r.Context(), q.User())
		if err != nil {
			respondInternalErr(w, q.Format(), fmt.Errorf("(un)star: find user: %w", err))
			return
		}

		if user.ListenBrainzUsername != nil && user.ListenBrainzSyncFeedback && len(songIDs) > 0 {
			songs, err := h.DB.Song().FindByIDs(r.Context(), songIDs, repos.IncludeSongInfo{
				Album: true,
				Lists: true,
			})
			if err != nil {
				respondInternalErr(w, q.Format(), fmt.Errorf("handle (un)star: %w", err))
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

			lbCon, err := h.ListenBrainz.GetListenbrainzConnection(r.Context(), q.User())
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
		res.EncodeOrLog(w, q.Format())
	}
}
