package handlers

import (
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
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

	songs, err := h.Store.FindRandomSongs(r.Context(), db.FindRandomSongsParams{
		UserName:    user,
		Limit:       int32(limit),
		FromYear:    fromYear,
		ToYear:      toYear,
		GenresLower: genres,
	})
	if err != nil {
		log.Errorf("get random songs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	songMap := make(map[string]*responses.Song, len(songs))
	songList := make([]*responses.Song, 0, len(songs))
	songIDs := mapData(songs, func(s *db.FindRandomSongsRow) string {
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
			Type:          "music",
			MediaType:     "song",
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
		songMap[song.ID] = song
		songList = append(songList, song)
		return s.ID
	})
	dbGenres, err := h.Store.FindGenresBySongs(r.Context(), songIDs)
	if err != nil {
		log.Errorf("get random songs: get genres: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, g := range dbGenres {
		song := songMap[g.SongID]
		if song.Genre == nil {
			song.Genre = &g.Name
		}
		song.Genres = append(song.Genres, &responses.GenreRef{
			Name: g.Name,
		})
	}
	artists, err := h.Store.FindArtistRefsBySongs(r.Context(), songIDs)
	if err != nil {
		log.Errorf("get random songs: get artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, a := range artists {
		song := songMap[a.SongID]
		if song.ArtistID == nil && song.Artist == nil {
			song.ArtistID = &a.ID
			song.Artist = &a.Name
		}
		song.Artists = append(song.Artists, &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		})
	}
	albumArtists, err := h.Store.FindAlbumArtistRefsBySongs(r.Context(), songIDs)
	if err != nil {
		log.Errorf("get random songs: get album artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, a := range albumArtists {
		song := songMap[a.SongID]
		if song.ArtistID == nil && song.Artist == nil {
			song.ArtistID = &a.ID
			song.Artist = &a.Name
		}
		song.AlbumArtists = append(song.AlbumArtists, &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		})
	}

	res := responses.New()
	res.RandomSongs = &responses.RandomSongs{
		Songs: songList,
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
			albums[album.ID] = &responses.Album{
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
			albumIds = append(albumIds, album.ID)
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
		if err != nil {
			log.Errorf("get album list 2: newest: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
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
			albums[album.ID] = &responses.Album{
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
			albumIds = append(albumIds, album.ID)
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
		if err != nil {
			log.Errorf("get album list 2: highest: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
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
			albums[album.ID] = &responses.Album{
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
			albumIds = append(albumIds, album.ID)
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
		if err != nil {
			log.Errorf("get album list 2: alphabetical by name: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
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
			albums[album.ID] = &responses.Album{
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
			albumIds = append(albumIds, album.ID)
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
		if err != nil {
			log.Errorf("get album list 2: starred: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
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
			albums[album.ID] = &responses.Album{
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
			albumIds = append(albumIds, album.ID)
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
		if err != nil {
			log.Errorf("get album list 2: by year: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
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
			albums[album.ID] = &responses.Album{
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
			albumIds = append(albumIds, album.ID)
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
		if err != nil {
			log.Errorf("get album list 2: by genre: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
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
			albums[album.ID] = &responses.Album{
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
			albumIds = append(albumIds, album.ID)
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
				album.CoverArt = &album.ID
			}
			album.IsDir = true
			album.Type = "music"
			album.MediaType = "album"
			return album
		}),
	}
	res.EncodeOrLog(w, query.Get("f"))
}
