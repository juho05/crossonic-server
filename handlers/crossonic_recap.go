package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

func (h *Handler) handleGetRecap(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	format := query.Get("f")

	year := time.Now().Year()
	if query.Get("year") != "" {
		var err error
		year, err = strconv.Atoi(query.Get("year"))
		if err != nil {
			responses.EncodeError(w, format, "invalid year parameter value", responses.SubsonicErrorGeneric)
			return
		}
	}

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)

	totalDuration, err := h.DB.Scrobble().GetDurationSum(r.Context(), user, start, end)
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get recap: get duration: %w", err))
		return
	}

	songCount, err := h.DB.Scrobble().GetDistinctSongCount(r.Context(), user, start, end)
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get recap: get song count: %w", err))
		return
	}

	albumCount, err := h.DB.Scrobble().GetDistinctAlbumCount(r.Context(), user, start, end)
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get recap: get album count: %w", err))
		return
	}

	artistCount, err := h.DB.Scrobble().GetDistinctArtistCount(r.Context(), user, start, end)
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get recap: get artist count: %w", err))
		return
	}

	res := responses.New()
	res.Recap = &responses.Recap{
		TotalDurationMS: totalDuration.ToStd().Milliseconds(),
		SongCount:       songCount,
		AlbumCount:      albumCount,
		ArtistCount:     artistCount,
	}
	res.EncodeOrLog(w, format)
}

func (h *Handler) handleGetTopSongsRecap(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	format := query.Get("f")

	year := time.Now().Year()
	if query.Get("year") != "" {
		var err error
		year, err = strconv.Atoi(query.Get("year"))
		if err != nil {
			responses.EncodeError(w, format, "invalid year parameter value", responses.SubsonicErrorGeneric)
			return
		}
	}

	limit := 10
	if query.Get("limit") != "" {
		var err error
		limit, err = strconv.Atoi(query.Get("limit"))
		if err != nil {
			responses.EncodeError(w, format, "invalid limit parameter value", responses.SubsonicErrorGeneric)
			return
		}
	}

	offset := 0
	if query.Get("offset") != "" {
		var err error
		offset, err = strconv.Atoi(query.Get("offset"))
		if err != nil {
			responses.EncodeError(w, format, "invalid offset parameter value", responses.SubsonicErrorGeneric)
			return
		}
	}

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)

	dbSongs, err := h.DB.Scrobble().GetTopSongsByDuration(r.Context(), user, start, end, offset, limit, repos.IncludeSongInfoFull(user))
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get top songs recap: get songs: %w", err))
		return
	}

	songs := util.Map(dbSongs, func(s *repos.ScrobbleTopSong) *responses.TopSongsRecapSong {
		return &responses.TopSongsRecapSong{
			Song:            responses.NewSong(s.CompleteSong),
			TotalDurationMS: s.TotalDuration.Millis(),
		}
	})

	res := responses.New()
	res.TopSongsRecap = &responses.TopSongsRecap{
		Songs: songs,
	}
	res.EncodeOrLog(w, format)
}
