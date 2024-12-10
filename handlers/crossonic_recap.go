package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
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

	start := pgtype.Timestamptz{
		Time:  time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
	end := pgtype.Timestamptz{
		Time:  time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}

	totalDuration, err := h.Store.GetScrobbleDurationSumMS(r.Context(), db.GetScrobbleDurationSumMSParams{
		UserName: user,
		Start:    start,
		End:      end,
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get racap: get duration: %w", err))
		return
	}

	songCount, err := h.Store.GetScrobbleDistinctSongCount(r.Context(), db.GetScrobbleDistinctSongCountParams{
		UserName: user,
		Start:    start,
		End:      end,
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get racap: get song count: %w", err))
		return
	}

	albumCount, err := h.Store.GetScrobbleDistinctAlbumCount(r.Context(), db.GetScrobbleDistinctAlbumCountParams{
		UserName: user,
		Start:    start,
		End:      end,
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get racap: get album count: %w", err))
		return
	}

	artistCount, err := h.Store.GetScrobbleDistinctArtistCount(r.Context(), db.GetScrobbleDistinctArtistCountParams{
		UserName: user,
		Start:    start,
		End:      end,
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get racap: get artist count: %w", err))
		return
	}

	res := responses.New()
	res.Recap = &responses.Recap{
		TotalDurationMS: totalDuration.(int64),
		SongCount:       songCount.(int64),
		AlbumCount:      albumCount.(int64),
		ArtistCount:     artistCount.(int64),
	}
	res.EncodeOrLog(w, format)
}
