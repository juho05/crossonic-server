package handlers

import (
	"context"
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
	albums := make(map[string]*responses.Album)
	var albumIds []string
	for _, album := range dbAlbums {
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
		var coverArt *string
		if hasCoverArt(album.ID) {
			coverArt = &album.ID
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
			CoverArt:      coverArt,
			IsDir:         true,
			Type:          "music",
			MediaType:     "album",
		}
		albumIds = append(albumIds, album.ID)
	}

	artistRefs, err := h.Store.FindArtistRefsByAlbums(ctx, albumIds)
	if err != nil {
		log.Errorf("search3: albums: get artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
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

	genreRefs, err := h.Store.FindGenresByAlbums(ctx, albumIds)
	if err != nil {
		log.Errorf("search3: albums: get genres: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
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

	return mapData(albumIds, func(albumID string) *responses.Album {
		return albums[albumID]
	}), true
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

	songs, err := h.Store.SearchSongs(ctx, sqlc.SearchSongsParams{
		UserName:  user,
		SearchStr: query.Get("query"),
		Offset:    int32(offset),
		Limit:     int32(limit),
	})
	if err != nil {
		log.Errorf("search3: songs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
	}
	songMap := make(map[string]*responses.Song, len(songs))
	songList := make([]*responses.Song, 0, len(songs))
	songIDs := mapData(songs, func(s *sqlc.SearchSongsRow) string {
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
	dbGenres, err := h.Store.FindGenresBySongs(ctx, songIDs)
	if err != nil {
		log.Errorf("search3: songs: get genres: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
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
	artists, err := h.Store.FindArtistRefsBySongs(ctx, songIDs)
	if err != nil {
		log.Errorf("search3: songs: get artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
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
	albumArtists, err := h.Store.FindAlbumArtistRefsBySongs(ctx, songIDs)
	if err != nil {
		log.Errorf("search3: songs: get album artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
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
	return songList, true
}
