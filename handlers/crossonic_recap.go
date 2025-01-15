package handlers

import (
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/juho05/crossonic-server/db"
	sqlc "github.com/juho05/crossonic-server/db/sqlc"
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

	totalDuration, err := h.Store.GetScrobbleDurationSumMS(r.Context(), sqlc.GetScrobbleDurationSumMSParams{
		UserName: user,
		Start:    start,
		End:      end,
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get recap: get duration: %w", err))
		return
	}

	songCount, err := h.Store.GetScrobbleDistinctSongCount(r.Context(), sqlc.GetScrobbleDistinctSongCountParams{
		UserName: user,
		Start:    start,
		End:      end,
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get recap: get song count: %w", err))
		return
	}

	albumCount, err := h.Store.GetScrobbleDistinctAlbumCount(r.Context(), sqlc.GetScrobbleDistinctAlbumCountParams{
		UserName: user,
		Start:    start,
		End:      end,
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get recap: get album count: %w", err))
		return
	}

	artistCount, err := h.Store.GetScrobbleDistinctArtistCount(r.Context(), sqlc.GetScrobbleDistinctArtistCountParams{
		UserName: user,
		Start:    start,
		End:      end,
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get recap: get artist count: %w", err))
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

	start := pgtype.Timestamptz{
		Time:  time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
	end := pgtype.Timestamptz{
		Time:  time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}

	dbSongs, err := h.Store.GetScrobbleTopSongsByDuration(r.Context(), sqlc.GetScrobbleTopSongsByDurationParams{
		UserName: user,
		Start:    start,
		End:      end,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get top songs recap: get songs: %w", err))
		return
	}

	songs := make([]*responses.Song, 0, len(dbSongs))
	topSongs := mapData(dbSongs, func(s *sqlc.GetScrobbleTopSongsByDurationRow) *responses.TopSongsRecapSong {
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
		song := &responses.Song{
			ID:            s.ID,
			IsDir:         false,
			Title:         s.Title,
			Album:         s.AlbumName,
			Track:         int32PtrToIntPtr(s.Track),
			Year:          int32PtrToIntPtr(s.Year),
			CoverArt:      s.CoverID,
			Size:          s.Size,
			ContentType:   s.ContentType,
			Suffix:        filepath.Ext(s.Path),
			Duration:      int(s.DurationMs) / 1000,
			BitRate:       int(s.BitRate),
			SamplingRate:  int(s.SamplingRate),
			ChannelCount:  int(s.ChannelCount),
			UserRating:    int32PtrToIntPtr(s.UserRating),
			DiscNumber:    int32PtrToIntPtr(s.DiscNumber),
			Created:       s.Created.Time,
			AlbumID:       s.AlbumID,
			BPM:           int32PtrToIntPtr(s.Bpm),
			MusicBrainzID: s.MusicBrainzID,
			Starred:       starred,
			AverageRating: averageRating,
			ReplayGain: &responses.ReplayGain{
				TrackGain:    s.ReplayGain,
				AlbumGain:    s.AlbumReplayGain,
				TrackPeak:    s.ReplayGainPeak,
				AlbumPeak:    s.AlbumReplayGainPeak,
				FallbackGain: &fallbackGain,
			},
		}
		songs = append(songs, song)
		return &responses.TopSongsRecapSong{
			Song:            song,
			TotalDurationMS: s.TotalDurationMs,
		}
	})

	err = h.completeSongInfo(r.Context(), songs)
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("get top songs recap: %w", err))
		return
	}

	res := responses.New()
	res.TopSongsRecap = &responses.TopSongsRecap{
		Songs: topSongs,
	}
	res.EncodeOrLog(w, format)
}
