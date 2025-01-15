package handlers

import (
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juho05/crossonic-server/db"
	sqlc "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func (h *Handler) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	limitStr := query.Get("size")
	limit := 10
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 || limit > 500 {
			responses.EncodeError(w, query.Get("f"), "invalid size value", responses.SubsonicErrorGeneric)
			return
		}
	}

	fromYearStr := query.Get("fromYear")
	var fromYear *int32
	if fromYearStr != "" {
		y, err := strconv.Atoi(fromYearStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid fromYear value", responses.SubsonicErrorGeneric)
			return
		}
		y32 := int32(y)
		fromYear = &y32
	}

	toYearStr := query.Get("toYear")
	var toYear *int32
	if toYearStr != "" {
		y, err := strconv.Atoi(toYearStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid toYear value", responses.SubsonicErrorGeneric)
			return
		}
		y32 := int32(y)
		toYear = &y32
	}

	genres := mapData(query["genre"], func(g string) string {
		return strings.ToLower(g)
	})

	dbSongs, err := h.Store.FindRandomSongs(r.Context(), sqlc.FindRandomSongsParams{
		UserName:    user,
		Limit:       int32(limit),
		FromYear:    fromYear,
		ToYear:      toYear,
		GenresLower: genres,
	})
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get random songs: %w", err))
		return
	}
	songs := mapData(dbSongs, func(s *sqlc.FindRandomSongsRow) *responses.Song {
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
	err = h.completeSongInfo(r.Context(), songs)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get random songs: %w", err))
		return
	}

	res := responses.New()
	res.RandomSongs = &responses.RandomSongs{
		Songs: songs,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetAlbumList2(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	listType := query.Get("type")

	limitStr := query.Get("size")
	limit := 10
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 || limit > 500 {
			responses.EncodeError(w, query.Get("f"), "invalid size value", responses.SubsonicErrorGeneric)
			return
		}
	}

	offsetStr := query.Get("offset")
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid offset value", responses.SubsonicErrorGeneric)
			return
		}
	}

	fromYearStr := query.Get("fromYear")
	var fromYear *int32
	if fromYearStr != "" {
		y, err := strconv.Atoi(fromYearStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid fromYear value", responses.SubsonicErrorGeneric)
			return
		}
		y32 := int32(y)
		fromYear = &y32
	} else if listType == "byYear" {
		responses.EncodeError(w, query.Get("f"), "missing fromYear parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	toYearStr := query.Get("toYear")
	var toYear *int32
	if toYearStr != "" {
		y, err := strconv.Atoi(toYearStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid toYear value", responses.SubsonicErrorGeneric)
			return
		}
		y32 := int32(y)
		toYear = &y32
	} else if listType == "byYear" {
		responses.EncodeError(w, query.Get("f"), "missing toYear parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	genres := query["genre"]
	if listType == "byGenre" && len(genres) == 0 {
		responses.EncodeError(w, query.Get("f"), "missing genre parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	genres = mapData(genres, func(g string) string {
		return strings.ToLower(g)
	})

	var albums []*responses.Album
	switch listType {
	case "random":
		if offset != 0 {
			responses.EncodeError(w, query.Get("f"), "offset is not supported for list type random", responses.SubsonicErrorGeneric)
			return
		}
		a, err := h.Store.FindAlbumsRandom(r.Context(), sqlc.FindAlbumsRandomParams{
			UserName:    user,
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		if err != nil {
			log.Errorf("get album list 2: random: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		albums = mapData(a, func(album *sqlc.FindAlbumsRandomRow) *responses.Album {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			var releaseTypes []string
			if album.ReleaseTypes != nil {
				releaseTypes = strings.Split(*album.ReleaseTypes, "\003")
			}
			var recordLabels []*responses.RecordLabel
			if album.RecordLabels != nil {
				recordLabels = mapData(strings.Split(*album.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
					return &responses.RecordLabel{
						Name: l,
					}
				})
			}
			return &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				SongCount:     int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
				MusicBrainzID: album.MusicBrainzID,
				IsCompilation: album.IsCompilation,
				ReleaseTypes:  releaseTypes,
				RecordLabels:  recordLabels,
			}
		})
	case "newest":
		a, err := h.Store.FindAlbumsNewest(r.Context(), sqlc.FindAlbumsNewestParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		if err != nil {
			log.Errorf("get album list 2: newest: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		albums = mapData(a, func(album *sqlc.FindAlbumsNewestRow) *responses.Album {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			var releaseTypes []string
			if album.ReleaseTypes != nil {
				releaseTypes = strings.Split(*album.ReleaseTypes, "\003")
			}
			var recordLabels []*responses.RecordLabel
			if album.RecordLabels != nil {
				recordLabels = mapData(strings.Split(*album.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
					return &responses.RecordLabel{
						Name: l,
					}
				})
			}
			return &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				SongCount:     int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
				MusicBrainzID: album.MusicBrainzID,
				IsCompilation: album.IsCompilation,
				ReleaseTypes:  releaseTypes,
				RecordLabels:  recordLabels,
			}
		})
	case "highest":
		a, err := h.Store.FindAlbumsHighestRated(r.Context(), sqlc.FindAlbumsHighestRatedParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		if err != nil {
			log.Errorf("get album list 2: highest: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		albums = mapData(a, func(album *sqlc.FindAlbumsHighestRatedRow) *responses.Album {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			var releaseTypes []string
			if album.ReleaseTypes != nil {
				releaseTypes = strings.Split(*album.ReleaseTypes, "\003")
			}
			var recordLabels []*responses.RecordLabel
			if album.RecordLabels != nil {
				recordLabels = mapData(strings.Split(*album.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
					return &responses.RecordLabel{
						Name: l,
					}
				})
			}
			return &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				SongCount:     int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
				MusicBrainzID: album.MusicBrainzID,
				IsCompilation: album.IsCompilation,
				ReleaseTypes:  releaseTypes,
				RecordLabels:  recordLabels,
			}
		})
	case "alphabeticalByName":
		a, err := h.Store.FindAlbumsAlphabeticalByName(r.Context(), sqlc.FindAlbumsAlphabeticalByNameParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		if err != nil {
			log.Errorf("get album list 2: alphabetical by name: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		albums = mapData(a, func(album *sqlc.FindAlbumsAlphabeticalByNameRow) *responses.Album {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			var releaseTypes []string
			if album.ReleaseTypes != nil {
				releaseTypes = strings.Split(*album.ReleaseTypes, "\003")
			}
			var recordLabels []*responses.RecordLabel
			if album.RecordLabels != nil {
				recordLabels = mapData(strings.Split(*album.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
					return &responses.RecordLabel{
						Name: l,
					}
				})
			}
			return &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				SongCount:     int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
				MusicBrainzID: album.MusicBrainzID,
				IsCompilation: album.IsCompilation,
				ReleaseTypes:  releaseTypes,
				RecordLabels:  recordLabels,
			}
		})
	case "starred":
		a, err := h.Store.FindAlbumsStarred(r.Context(), sqlc.FindAlbumsStarredParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		if err != nil {
			log.Errorf("get album list 2: starred: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		albums = mapData(a, func(album *sqlc.FindAlbumsStarredRow) *responses.Album {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			var releaseTypes []string
			if album.ReleaseTypes != nil {
				releaseTypes = strings.Split(*album.ReleaseTypes, "\003")
			}
			var recordLabels []*responses.RecordLabel
			if album.RecordLabels != nil {
				recordLabels = mapData(strings.Split(*album.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
					return &responses.RecordLabel{
						Name: l,
					}
				})
			}
			return &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				SongCount:     int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
				MusicBrainzID: album.MusicBrainzID,
				IsCompilation: album.IsCompilation,
				ReleaseTypes:  releaseTypes,
				RecordLabels:  recordLabels,
			}
		})
	case "byYear":
		a, err := h.Store.FindAlbumsByYear(r.Context(), sqlc.FindAlbumsByYearParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		if err != nil {
			log.Errorf("get album list 2: by year: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		albums = mapData(a, func(album *sqlc.FindAlbumsByYearRow) *responses.Album {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			var releaseTypes []string
			if album.ReleaseTypes != nil {
				releaseTypes = strings.Split(*album.ReleaseTypes, "\003")
			}
			var recordLabels []*responses.RecordLabel
			if album.RecordLabels != nil {
				recordLabels = mapData(strings.Split(*album.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
					return &responses.RecordLabel{
						Name: l,
					}
				})
			}
			return &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				SongCount:     int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
				MusicBrainzID: album.MusicBrainzID,
				IsCompilation: album.IsCompilation,
				ReleaseTypes:  releaseTypes,
				RecordLabels:  recordLabels,
			}
		})
	case "byGenre":
		a, err := h.Store.FindAlbumsByGenre(r.Context(), sqlc.FindAlbumsByGenreParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		if err != nil {
			log.Errorf("get album list 2: by genre: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		albums = mapData(a, func(album *sqlc.FindAlbumsByGenreRow) *responses.Album {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			var releaseTypes []string
			if album.ReleaseTypes != nil {
				releaseTypes = strings.Split(*album.ReleaseTypes, "\003")
			}
			var recordLabels []*responses.RecordLabel
			if album.RecordLabels != nil {
				recordLabels = mapData(strings.Split(*album.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
					return &responses.RecordLabel{
						Name: l,
					}
				})
			}
			return &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				SongCount:     int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
				MusicBrainzID: album.MusicBrainzID,
				IsCompilation: album.IsCompilation,
				ReleaseTypes:  releaseTypes,
				RecordLabels:  recordLabels,
			}
		})
	default:
		responses.EncodeError(w, query.Get("f"), "unsupported list type: "+listType, responses.SubsonicErrorGeneric)
		return
	}

	err = h.completeAlbumInfo(r.Context(), albums)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album list2: %w", err))
		return
	}

	res := responses.New()
	res.AlbumList2 = &responses.AlbumList2{
		Albums: albums,
	}
	res.EncodeOrLog(w, query.Get("f"))
}
