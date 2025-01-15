package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juho05/crossonic-server/db"
	sqlc "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func (h *Handler) handleSearch3(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	query.Set("query", strings.Trim(query.Get("query"), `"`))

	artists, ok := h.searchArtists(r.Context(), w, query)
	if !ok {
		return
	}

	albums, ok := h.searchAlbums(r.Context(), w, query)
	if !ok {
		return
	}

	songs, ok := h.searchSongs(r.Context(), w, query)
	if !ok {
		return
	}

	res := responses.New()
	res.SearchResult3 = &responses.SearchResult3{
		Songs:   songs,
		Albums:  albums,
		Artists: artists,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) searchArtists(ctx context.Context, w http.ResponseWriter, query url.Values) ([]*responses.Artist, bool) {
	user := query.Get("u")
	limitStr := query.Get("artistCount")
	limit := 20
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid artistCount value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	offsetStr := query.Get("artistOffset")
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid artistOffset value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	artists, err := h.Store.SearchAlbumArtists(ctx, sqlc.SearchAlbumArtistsParams{
		UserName:  user,
		Offset:    int32(offset),
		Limit:     int32(limit),
		SearchStr: query.Get("query"),
	})
	if err != nil {
		log.Errorf("search3: artists: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
	}
	return mapData(artists, func(a *sqlc.SearchAlbumArtistsRow) *responses.Artist {
		var coverArt *string
		if hasCoverArt(a.ID) {
			coverArt = &a.ID
		}
		albumCount := int(a.AlbumCount)
		var averageRating *float64
		if a.AvgRating != 0 {
			avgRating := math.Round(a.AvgRating*100) / 100
			averageRating = &avgRating
		}
		var starred *time.Time
		if a.Starred.Valid {
			starred = &a.Starred.Time
		}
		return &responses.Artist{
			ID:            a.ID,
			Name:          a.Name,
			CoverArt:      coverArt,
			AlbumCount:    &albumCount,
			Starred:       starred,
			MusicBrainzID: a.MusicBrainzID,
			UserRating:    int32PtrToIntPtr(a.UserRating),
			AverageRating: averageRating,
		}
	}), true
}

func (h *Handler) searchAlbums(ctx context.Context, w http.ResponseWriter, query url.Values) ([]*responses.Album, bool) {
	user := query.Get("u")
	limitStr := query.Get("albumCount")
	limit := 20
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid albumCount value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	offsetStr := query.Get("albumOffset")
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid albumOffset value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	dbAlbums, err := h.Store.SearchAlbums(ctx, sqlc.SearchAlbumsParams{
		UserName:  user,
		Offset:    int32(offset),
		Limit:     int32(limit),
		SearchStr: query.Get("query"),
	})
	if err != nil {
		log.Errorf("search3: albums: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
	}
	albums := mapData(dbAlbums, func(a *sqlc.SearchAlbumsRow) *responses.Album {
		var starred *time.Time
		if a.Starred.Valid {
			starred = &a.Starred.Time
		}
		var averageRating *float64
		if a.AvgRating != 0 {
			avgRating := math.Round(a.AvgRating*100) / 100
			averageRating = &avgRating
		}
		var releaseTypes []string
		if a.ReleaseTypes != nil {
			releaseTypes = strings.Split(*a.ReleaseTypes, "\003")
		}
		var recordLabels []*responses.RecordLabel
		if a.RecordLabels != nil {
			recordLabels = mapData(strings.Split(*a.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
				return &responses.RecordLabel{
					Name: l,
				}
			})
		}
		return &responses.Album{
			ID:            a.ID,
			Created:       a.Created.Time,
			Title:         a.Name,
			Name:          a.Name,
			SongCount:     int(a.TrackCount),
			Duration:      int(a.DurationMs / 1000),
			Year:          int32PtrToIntPtr(a.Year),
			Starred:       starred,
			UserRating:    int32PtrToIntPtr(a.UserRating),
			AverageRating: averageRating,
			MusicBrainzID: a.MusicBrainzID,
			IsCompilation: a.IsCompilation,
			ReleaseTypes:  releaseTypes,
			RecordLabels:  recordLabels,
		}
	})

	err = h.completeAlbumInfo(ctx, albums)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("search3: search albums: %w", err))
		return nil, false
	}

	return albums, true
}

func (h *Handler) searchSongs(ctx context.Context, w http.ResponseWriter, query url.Values) ([]*responses.Song, bool) {
	user := query.Get("u")
	limitStr := query.Get("songCount")
	limit := 20
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid songCount value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	offsetStr := query.Get("songOffset")
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid songOffset value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	dbSongs, err := h.Store.SearchSongs(ctx, sqlc.SearchSongsParams{
		UserName:  user,
		SearchStr: query.Get("query"),
		Offset:    int32(offset),
		Limit:     int32(limit),
	})
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("search3: songs: %w", err))
		return nil, false
	}
	songs := mapData(dbSongs, func(s *sqlc.SearchSongsRow) *responses.Song {
		var starred *time.Time
		if s.Starred.Valid {
			starred = &s.Starred.Time
		}
		var averageRating *float64
		if s.AvgRating != 0 {
			avgRating := math.Round(s.AvgRating*100) / 100
			averageRating = &avgRating
		}
		fallbackGain := float32(db.GetFallbackGain(ctx, h.Store))
		return &responses.Song{
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
	})
	err = h.completeSongInfo(ctx, songs)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("search3: songs: %w", err))
		return nil, false
	}
	return songs, true
}
