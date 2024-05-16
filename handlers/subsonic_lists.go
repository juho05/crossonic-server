package handlers

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

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

	albums := make(map[string]*responses.Album)
	var albumIds []string
	switch listType {
	case "random":
		if offset != 0 {
			responses.EncodeError(w, query.Get("f"), "offset is not supported for list type random", responses.SubsonicErrorGeneric)
			return
		}
		a, err := h.Store.FindAlbumsRandom(r.Context(), db.FindAlbumsRandomParams{
			UserName:    user,
			Offset:      int32(offset),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		for _, album := range a {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			albums[album.ID] = &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				TrackCount:    int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
			}
			albumIds = append(albumIds, album.ID)
		}
		if err != nil {
			log.Errorf("get album list 2: random: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	case "newest":
		a, err := h.Store.FindAlbumsNewest(r.Context(), db.FindAlbumsNewestParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		for _, album := range a {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			albums[album.ID] = &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				TrackCount:    int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
			}
			albumIds = append(albumIds, album.ID)
		}
		if err != nil {
			log.Errorf("get album list 2: newest: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	case "highest":
		a, err := h.Store.FindAlbumsHighestRated(r.Context(), db.FindAlbumsHighestRatedParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		for _, album := range a {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			albums[album.ID] = &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				TrackCount:    int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
			}
			albumIds = append(albumIds, album.ID)
		}
		if err != nil {
			log.Errorf("get album list 2: highest: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	case "alphabeticalByName":
		a, err := h.Store.FindAlbumsAlphabeticalByName(r.Context(), db.FindAlbumsAlphabeticalByNameParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		for _, album := range a {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			albums[album.ID] = &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				TrackCount:    int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
			}
			albumIds = append(albumIds, album.ID)
		}
		if err != nil {
			log.Errorf("get album list 2: alphabetical by name: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	case "starred":
		a, err := h.Store.FindAlbumsStarred(r.Context(), db.FindAlbumsStarredParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		for _, album := range a {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			albums[album.ID] = &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				TrackCount:    int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
			}
			albumIds = append(albumIds, album.ID)
		}
		if err != nil {
			log.Errorf("get album list 2: starred: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	case "byYear":
		a, err := h.Store.FindAlbumsByYear(r.Context(), db.FindAlbumsByYearParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		for _, album := range a {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			albums[album.ID] = &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				TrackCount:    int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
			}
			albumIds = append(albumIds, album.ID)
		}
		if err != nil {
			log.Errorf("get album list 2: by year: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	case "byGenre":
		a, err := h.Store.FindAlbumsByGenre(r.Context(), db.FindAlbumsByGenreParams{
			UserName:    user,
			Offset:      int32(offset),
			Limit:       int32(limit),
			FromYear:    fromYear,
			ToYear:      toYear,
			GenresLower: genres,
		})
		for _, album := range a {
			var starred *time.Time
			if album.Starred.Valid {
				starred = &album.Starred.Time
			}
			var averageRating *float64
			if album.AvgRating != 0 {
				avgRating := math.Round(album.AvgRating*100) / 100
				averageRating = &avgRating
			}
			albums[album.ID] = &responses.Album{
				ID:            album.ID,
				Created:       album.Created.Time,
				Title:         album.Name,
				Name:          album.Name,
				TrackCount:    int(album.TrackCount),
				Duration:      int(album.DurationMs / 1000),
				Year:          int32PtrToIntPtr(album.Year),
				Starred:       starred,
				UserRating:    int32PtrToIntPtr(album.UserRating),
				AverageRating: averageRating,
			}
			albumIds = append(albumIds, album.ID)
		}
		if err != nil {
			log.Errorf("get album list 2: by genre: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	default:
		responses.EncodeError(w, query.Get("f"), "unsupported list type: "+listType, responses.SubsonicErrorGeneric)
		return
	}

	artistRefs, err := h.Store.FindArtistRefsByAlbums(r.Context(), albumIds)
	if err != nil {
		log.Errorf("get album list 2: get artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, a := range artistRefs {
		album := albums[a.AlbumID]
		if album.Artist == nil && album.ArtistID == nil {
			album.Artist = &a.Name
			album.ArtistID = &a.ID
		}
		album.Artists = append(album.Artists, &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		})
	}

	genreRefs, err := h.Store.FindGenresByAlbums(r.Context(), albumIds)
	if err != nil {
		log.Errorf("get album list 2: get genres: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, g := range genreRefs {
		album := albums[g.AlbumID]
		if album.Genre == nil {
			album.Genre = &g.Name
		}
		album.Genres = append(album.Genres, &responses.GenreRef{
			Name: g.Name,
		})
	}

	res := responses.New()
	res.AlbumList2 = &responses.AlbumList2{
		Albums: mapData(albumIds, func(albumID string) *responses.Album {
			album := albums[albumID]
			if hasCoverArt(album.ID) {
				album.CoverID = &album.ID
			}
			return album
		}),
	}
	res.EncodeOrLog(w, query.Get("f"))
}
