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
	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

// https://opensubsonic.netlify.app/docs/endpoints/scrobble/
func (h *Handler) handleScrobble(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	idType, ok := crossonic.GetIDType(id)
	if !ok {
		responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
		return
	}
	if idType != crossonic.IDTypeSong {
		responses.EncodeError(w, query.Get("f"), "scrobbles are not supported for id type "+string(idType), responses.SubsonicErrorNotFound)
		return
	}
	timeStr := query.Get("time")
	scrobbleTime := time.Now()
	if timeStr != "" {
		timeInt, err := strconv.Atoi(timeStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid time value", responses.SubsonicErrorGeneric)
			return
		}
		scrobbleTime = time.UnixMilli(int64(timeInt))
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

	durationMsStr := query.Get("duration_ms")
	var durationMs *int
	if durationMsStr != "" {
		d, err := strconv.Atoi(durationMsStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid duration_ms value", responses.SubsonicErrorGeneric)
			return
		}
		durationMs = &d
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

	tx, err := h.Store.BeginTransaction(r.Context())
	if err != nil {
		log.Errorf("scrobble: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer tx.Rollback(r.Context())
	if !submission {
		err = tx.DeleteNowPlaying(r.Context(), user)
		if err != nil {
			log.Errorf("scrobble: delete old now playing: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	}

	_, err = tx.CreateScrobble(r.Context(), db.CreateScrobbleParams{
		UserName: user,
		SongID:   song.ID,
		AlbumID:  song.AlbumID,
		Time: pgtype.Timestamptz{
			Time:  scrobbleTime,
			Valid: true,
		},
		SongDurationMs:          song.DurationMs,
		DurationMs:              intPtrToInt32Ptr(durationMs),
		SubmittedToListenbrainz: false,
		NowPlaying:              !submission,
	})
	if err != nil {
		log.Errorf("scrobble: create scrobble: %s", err)
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
	songIDs := mapData(songs, func(s *db.GetNowPlayingSongsRow) string {
		var starred *time.Time
		if s.Starred.Valid {
			starred = &s.Starred.Time
		}
		var averageRating *float64
		if s.AvgRating != 0 {
			avgRating := math.Round(s.AvgRating*100) / 100
			averageRating = &avgRating
		}
		fallbackGain := config.ReplayGainFallback()
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
			err = h.Store.RemoveSongRating(r.Context(), db.RemoveSongRatingParams{
				UserName: user,
				SongID:   id,
			})
		} else {
			err = h.Store.SetSongRating(r.Context(), db.SetSongRatingParams{
				SongID:   id,
				UserName: user,
				Rating:   int32(rating),
			})
		}
	case crossonic.IDTypeAlbum:
		if rating == 0 {
			err = h.Store.RemoveAlbumRating(r.Context(), db.RemoveAlbumRatingParams{
				UserName: user,
				AlbumID:  id,
			})
		} else {
			err = h.Store.SetAlbumRating(r.Context(), db.SetAlbumRatingParams{
				AlbumID:  id,
				UserName: user,
				Rating:   int32(rating),
			})
		}
	case crossonic.IDTypeArtist:
		if rating == 0 {
			err = h.Store.RemoveArtistRating(r.Context(), db.RemoveArtistRatingParams{
				UserName: user,
				ArtistID: id,
			})
		} else {
			err = h.Store.SetArtistRating(r.Context(), db.SetArtistRatingParams{
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
	query := getQuery(r)
	user := query.Get("u")

	var ids []string
	ids = append(ids, query["id"]...)
	ids = append(ids, query["albumId"]...)
	ids = append(ids, query["artistId"]...)

	tx, err := h.Store.BeginTransaction(r.Context())
	if err != nil {
		log.Errorf("star: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer tx.Rollback(r.Context())
	for _, id := range ids {
		idType, ok := crossonic.GetIDType(id)
		if !ok {
			responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
			return
		}
		var err error
		switch idType {
		case crossonic.IDTypeSong:
			err = tx.StarSong(r.Context(), db.StarSongParams{
				SongID:   id,
				UserName: user,
			})
		case crossonic.IDTypeAlbum:
			err = tx.StarAlbum(r.Context(), db.StarAlbumParams{
				AlbumID:  id,
				UserName: user,
			})
		case crossonic.IDTypeArtist:
			err = tx.StarArtist(r.Context(), db.StarArtistParams{
				ArtistID: id,
				UserName: user,
			})
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

	err = tx.Commit(r.Context())
	if err != nil {
		log.Errorf("star: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	res := responses.New()
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/unstar/
func (h *Handler) handleUnstar(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	var ids []string
	ids = append(ids, query["id"]...)
	ids = append(ids, query["albumId"]...)
	ids = append(ids, query["artistId"]...)

	tx, err := h.Store.BeginTransaction(r.Context())
	if err != nil {
		log.Errorf("unstar: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer tx.Rollback(r.Context())
	for _, id := range ids {
		idType, ok := crossonic.GetIDType(id)
		if !ok {
			responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
			return
		}
		var err error
		switch idType {
		case crossonic.IDTypeSong:
			err = tx.UnstarSong(r.Context(), db.UnstarSongParams{
				SongID:   id,
				UserName: user,
			})
		case crossonic.IDTypeAlbum:
			err = tx.UnstarAlbum(r.Context(), db.UnstarAlbumParams{
				AlbumID:  id,
				UserName: user,
			})
		case crossonic.IDTypeArtist:
			err = tx.UnstarArtist(r.Context(), db.UnstarArtistParams{
				ArtistID: id,
				UserName: user,
			})
		}
		if err != nil {
			log.Errorf("unstar: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	}

	err = tx.Commit(r.Context())
	if err != nil {
		log.Errorf("unstar: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	res := responses.New()
	res.EncodeOrLog(w, query.Get("f"))
}
