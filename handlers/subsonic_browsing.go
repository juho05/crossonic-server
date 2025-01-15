package handlers

import (
	"errors"
	"fmt"
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

	albums := mapData(dbAlbums, func(a *sqlc.FindAlbumsByArtistRow) *responses.Album {
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

		return &responses.Album{
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
			MusicBrainzID: a.MusicBrainzID,
			RecordLabels:  recordLabels,
			ReleaseTypes:  releaseTypes,
			IsCompilation: a.IsCompilation,
		}
	})

	err = h.completeAlbumInfo(r.Context(), albums)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("handle get artist: %w", err))
		return
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
	dbAlbum, err := h.Store.FindAlbum(r.Context(), sqlc.FindAlbumParams{
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

	dbSongs, err := h.Store.FindSongsByAlbum(r.Context(), sqlc.FindSongsByAlbumParams{
		UserName: user,
		ID:       dbAlbum.ID,
	})
	if err != nil {
		log.Errorf("get album: get songs: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	songs := mapData(dbSongs, func(s *sqlc.FindSongsByAlbumRow) *responses.Song {
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
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album: %w", err))
		return
	}

	var starred *time.Time
	if dbAlbum.Starred.Valid {
		starred = &dbAlbum.Starred.Time
	}

	var averageRating *float64
	if dbAlbum.AvgRating != 0 {
		avgRating := math.Round(dbAlbum.AvgRating*100) / 100
		averageRating = &avgRating
	}

	var releaseTypes []string
	if dbAlbum.ReleaseTypes != nil {
		releaseTypes = strings.Split(*dbAlbum.ReleaseTypes, "\003")
	}
	var recordLabels []*responses.RecordLabel
	if dbAlbum.RecordLabels != nil {
		recordLabels = mapData(strings.Split(*dbAlbum.RecordLabels, "\003"), func(l string) *responses.RecordLabel {
			return &responses.RecordLabel{
				Name: l,
			}
		})
	}

	album := &responses.Album{
		ID:            dbAlbum.ID,
		Created:       dbAlbum.Created.Time,
		Title:         dbAlbum.Name,
		Name:          dbAlbum.Name,
		SongCount:     int(dbAlbum.TrackCount),
		Duration:      int(dbAlbum.DurationMs / 1000),
		Year:          int32PtrToIntPtr(dbAlbum.Year),
		Starred:       starred,
		UserRating:    int32PtrToIntPtr(dbAlbum.UserRating),
		AverageRating: averageRating,
		MusicBrainzID: dbAlbum.MusicBrainzID,
		IsCompilation: dbAlbum.IsCompilation,
		RecordLabels:  recordLabels,
		ReleaseTypes:  releaseTypes,
	}
	err = h.completeAlbumInfo(r.Context(), []*responses.Album{album})
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album: %w", err))
		return
	}

	res := responses.New()
	res.Album = &responses.AlbumWithSongs{
		Album: album,
		Songs: songs,
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
