package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

func (h *Handler) handleGetRecap(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	year, ok := q.IntDef("year", time.Now().Year())
	if !ok {
		return
	}

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)

	totalDuration, err := h.DB.Scrobble().GetDurationSum(r.Context(), q.User(), start, end)
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get recap: get duration: %w", err))
		return
	}

	songCount, err := h.DB.Scrobble().GetDistinctSongCount(r.Context(), q.User(), start, end)
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get recap: get song count: %w", err))
		return
	}

	albumCount, err := h.DB.Scrobble().GetDistinctAlbumCount(r.Context(), q.User(), start, end)
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get recap: get album count: %w", err))
		return
	}

	artistCount, err := h.DB.Scrobble().GetDistinctArtistCount(r.Context(), q.User(), start, end)
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get recap: get artist count: %w", err))
		return
	}

	res := responses.New()
	res.Recap = &responses.Recap{
		TotalDurationMS: totalDuration.ToStd().Milliseconds(),
		SongCount:       songCount,
		AlbumCount:      albumCount,
		ArtistCount:     artistCount,
	}
	res.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleGetTopSongsRecap(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	year, ok := q.IntDef("year", time.Now().Year())
	if !ok {
		return
	}

	paginate, ok := q.Paginate("limit", "offset", 10)
	if !ok {
		return
	}

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)

	dbSongs, err := h.DB.Scrobble().GetTopSongsByDuration(r.Context(), q.User(), start, end, paginate, repos.IncludeSongInfoFull(q.User()))
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get top songs recap: get songs: %w", err))
		return
	}

	songs := util.Map(dbSongs, func(s *repos.ScrobbleTopSong) *responses.TopSongsRecapSong {
		return &responses.TopSongsRecapSong{
			Song:            responses.NewSong(s.CompleteSong, h.Config),
			TotalDurationMS: s.TotalDuration.Millis(),
		}
	})

	res := responses.New()
	res.TopSongsRecap = &responses.TopSongsRecap{
		Songs: songs,
	}
	res.EncodeOrLog(w, q.Format())
}
