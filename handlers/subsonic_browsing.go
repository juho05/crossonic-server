package handlers

import (
	"errors"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/juho05/crossonic-server/db"
	sqlc "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func (h *Handler) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	artist, err := h.Store.FindArtist(r.Context(), sqlc.FindArtistParams{
		UserName: user,
		ID:       id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("get artist: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}
	dbAlbums, err := h.Store.FindAlbumsByArtist(r.Context(), sqlc.FindAlbumsByArtistParams{
		UserName: user,
		ArtistID: artist.ID,
	})
	if err != nil {
		log.Errorf("get artist: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	var coverArt *string
	if hasCoverArt(artist.ID) {
		coverArt = &artist.ID
	}

	var starred *time.Time
	if artist.Starred.Valid {
		starred = &artist.Starred.Time
	}

	var averageRating *float64
	if artist.AvgRating != 0 {
		averageRating = &artist.AvgRating
	}

	albumIds := make([]string, 0, len(dbAlbums))
	albumsMap := make(map[string]*responses.Album, len(dbAlbums))
	albums := mapData(dbAlbums, func(a *sqlc.FindAlbumsByArtistRow) *responses.Album {
		var coverArt *string
		if hasCoverArt(a.ID) {
			coverArt = &a.ID
		}

		var starred *time.Time
		if a.Starred.Valid {
			starred = &a.Starred.Time
		}

		var averageRating *float64
		if a.AvgRating > 0 {
			averageRating = &a.AvgRating
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

		album := &responses.Album{
			ID:            a.ID,
			Created:       a.Created.Time,
			CoverArt:      coverArt,
			Title:         a.Name,
			Name:          a.Name,
			SongCount:     int(a.TrackCount),
			Duration:      int(a.DurationMs / 1000),
			Year:          int32PtrToIntPtr(a.Year),
			Starred:       starred,
			UserRating:    int32PtrToIntPtr(a.UserRating),
			AverageRating: averageRating,
			IsDir:         true,
			Type:          "music",
			MediaType:     "album",
			MusicBrainzID: a.MusicBrainzID,
			RecordLabels:  recordLabels,
			ReleaseTypes:  releaseTypes,
			IsCompilation: a.IsCompilation,
		}
		albumIds = append(albumIds, album.ID)
		albumsMap[album.ID] = album
		return album
	})

	artistRefs, err := h.Store.FindArtistRefsByAlbums(r.Context(), albumIds)
	if err != nil {
		log.Errorf("get artist: get albums: get artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, a := range artistRefs {
		album := albumsMap[a.AlbumID]
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
		log.Errorf("get artist: get albums: get genres: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, g := range genreRefs {
		album := albumsMap[g.AlbumID]
		if album.Genre == nil {
			album.Genre = &g.Name
		}
		album.Genres = append(album.Genres, &responses.GenreRef{
			Name: g.Name,
		})
	}

	albumCount := len(albums)

	res := responses.New()
	res.Artist = &responses.Artist{
		ID:            artist.ID,
		Name:          artist.Name,
		CoverArt:      coverArt,
		Starred:       starred,
		MusicBrainzID: artist.MusicBrainzID,
		UserRating:    int32PtrToIntPtr(artist.UserRating),
		AverageRating: averageRating,
		Albums:        albums,
		AlbumCount:    &albumCount,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetAlbum(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	album, err := h.Store.FindAlbum(r.Context(), sqlc.FindAlbumParams{
		UserName: user,
		ID:       id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("get album: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}
	genres, err := h.Store.FindGenresByAlbums(r.Context(), []string{album.ID})
	if err != nil {
		log.Errorf("get album: get genres: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	genreRefs := mapData(genres, func(g *sqlc.FindGenresByAlbumsRow) *responses.GenreRef {
		return &responses.GenreRef{
			Name: g.Name,
		}
	})
	artists, err := h.Store.FindArtistRefsByAlbums(r.Context(), []string{album.ID})
	if err != nil {
		log.Errorf("get album: get artists: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	artistRefs := mapData(artists, func(a *sqlc.FindArtistRefsByAlbumsRow) *responses.ArtistRef {
		return &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		}
	})
	songs, err := h.Store.FindSongsByAlbum(r.Context(), sqlc.FindSongsByAlbumParams{
		UserName: user,
		ID:       album.ID,
	})
	if err != nil {
		log.Errorf("get album: get songs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	songMap := make(map[string]*responses.Song, len(songs))
	songList := make([]*responses.Song, 0, len(songs))
	songIDs := mapData(songs, func(s *sqlc.FindSongsByAlbumRow) string {
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
			AlbumArtists: artistRefs,
		}
		songMap[song.ID] = song
		songList = append(songList, song)
		return s.ID
	})
	dbGenres, err := h.Store.FindGenresBySongs(r.Context(), songIDs)
	if err != nil {
		log.Errorf("get album: get songs: get genres: %s", err)
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
	songArtists, err := h.Store.FindArtistRefsBySongs(r.Context(), songIDs)
	if err != nil {
		log.Errorf("get album: get songs: get artist refs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	for _, a := range songArtists {
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

	var artistID *string
	var artistName *string
	if len(artistRefs) > 0 {
		artistID = &artists[0].ID
		artistName = &artists[0].Name
	}

	var genre *string
	if len(genreRefs) > 0 {
		genre = &genreRefs[0].Name
	}

	var coverID *string
	if hasCoverArt(album.ID) {
		coverID = &album.ID
	}

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

	res := responses.New()
	res.Album = &responses.AlbumWithSongs{
		Album: responses.Album{
			ID:            album.ID,
			Created:       album.Created.Time,
			ArtistID:      artistID,
			Artist:        artistName,
			Artists:       artistRefs,
			CoverArt:      coverID,
			Title:         album.Name,
			Name:          album.Name,
			SongCount:     int(album.TrackCount),
			Duration:      int(album.DurationMs / 1000),
			Genre:         genre,
			Genres:        genreRefs,
			Year:          int32PtrToIntPtr(album.Year),
			Starred:       starred,
			UserRating:    int32PtrToIntPtr(album.UserRating),
			AverageRating: averageRating,
			IsDir:         true,
			Type:          "music",
			MediaType:     "album",
			MusicBrainzID: album.MusicBrainzID,
			IsCompilation: album.IsCompilation,
			RecordLabels:  recordLabels,
			ReleaseTypes:  releaseTypes,
		},
		Songs: songList,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetGenres(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)

	dbGenres, err := h.Store.FindGenresWithCount(r.Context())
	if err != nil {
		log.Errorf("get genres: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	genres := make(responses.Genres, 0, len(dbGenres))
	for _, g := range dbGenres {
		genres = append(genres, responses.Genre{
			SongCount:  int(g.SongCount),
			AlbumCount: int(g.AlbumCount),
			Value:      g.Name,
		})
	}

	res := responses.New()
	res.Genres = &genres
	res.EncodeOrLog(w, query.Get("f"))
}

var ignoredArticles = []string{"The", "An", "A", "Der", "Die", "Das", "Ein", "Eine", "Les", "Le", "La", "L'"}

func (h *Handler) handleGetArtists(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)

	artists, err := h.Store.FindAlbumArtists(r.Context(), query.Get("u"))
	if err != nil {
		log.Errorf("get artists: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	indexMap := make(map[rune]*responses.Index, 27)
	for i, a := range artists {
		if artists[i].AlbumCount == 0 {
			continue
		}
		name := a.Name
		for _, i := range ignoredArticles {
			before := len(name)
			name = strings.TrimPrefix(name, i+" ")
			if len(name) < before {
				break
			}
		}
		name = strings.TrimSpace(name)
		runes := []rune(name)
		key := '#'
		if len(runes) > 0 && unicode.IsLetter(runes[0]) {
			key = unicode.ToLower(runes[0])
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
		var coverArt *string
		if hasCoverArt(a.ID) {
			coverArt = &a.ID
		}
		artist := &responses.Artist{
			ID:            a.ID,
			Name:          a.Name,
			CoverArt:      coverArt,
			AlbumCount:    &albumCount,
			Starred:       starred,
			MusicBrainzID: a.MusicBrainzID,
			UserRating:    int32PtrToIntPtr(a.UserRating),
			AverageRating: averageRating,
		}
		if i, ok := indexMap[key]; ok {
			i.Artist = append(i.Artist, artist)
		} else {
			indexMap[key] = &responses.Index{
				Name:   string(key),
				Artist: []*responses.Artist{artist},
			}
		}
	}

	indexList := make([]*responses.Index, 0, len(indexMap))
	for _, r := range "#abcdefghijklmnopqrstuvwxyz" {
		if k, ok := indexMap[r]; ok {
			indexList = append(indexList, k)
		}
	}

	res := responses.New()
	res.Artists = &responses.Artists{
		IgnoredArticles: strings.Join(ignoredArticles, " "),
		Index:           indexList,
	}
	res.EncodeOrLog(w, query.Get("f"))
}
