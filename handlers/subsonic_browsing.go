package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
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
	artist, err := h.DB.Artist().FindByID(r.Context(), id, repos.IncludeArtistInfoFull(user))
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("get artist: %w", err))
		}
		return
	}
	dbAlbums, err := h.DB.Artist().GetAlbums(r.Context(), artist.ID, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get artist: get albums: %w", err))
		return
	}

	var coverArt *string
	if hasCoverArt(artist.ID) {
		coverArt = &artist.ID
	}

	albums := mapList(dbAlbums, func(a *repos.CompleteAlbum) *responses.Album {
		return &responses.Album{
			ID:            a.ID,
			Created:       a.Created,
			CoverArt:      coverArt,
			Title:         a.Name,
			Name:          a.Name,
			SongCount:     int(a.TrackCount),
			Duration:      int(a.Duration.ToStd().Seconds()),
			Year:          a.Year,
			Starred:       a.Starred,
			UserRating:    a.UserRating,
			AverageRating: a.AverageRating,
			MusicBrainzID: a.MusicBrainzID,
			RecordLabels: mapList(a.RecordLabels, func(label string) *responses.RecordLabel {
				return &responses.RecordLabel{
					Name: label,
				}
			}),
			ReleaseTypes:  a.ReleaseTypes,
			IsCompilation: a.IsCompilation,
			Genres: mapList(a.Genres, func(g string) *responses.GenreRef {
				return &responses.GenreRef{
					Name: g,
				}
			}),
			Artists: mapList(a.Artists, func(a repos.ArtistRef) *responses.ArtistRef {
				return &responses.ArtistRef{
					ID:   a.ID,
					Name: a.Name,
				}
			}),
		}
	})

	err = h.completeAlbumInfo(albums)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("handle get artist: %w", err))
		return
	}

	res := responses.New()
	res.Artist = &responses.Artist{
		ID:            artist.ID,
		Name:          artist.Name,
		CoverArt:      coverArt,
		Starred:       artist.Starred,
		MusicBrainzID: artist.MusicBrainzID,
		UserRating:    artist.UserRating,
		AverageRating: artist.AverageRating,
		Albums:        albums,
		AlbumCount:    &artist.AlbumCount,
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
	dbAlbum, err := h.DB.Album().FindByID(r.Context(), id, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("get album: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}

	dbSongs, err := h.DB.Album().GetTracks(r.Context(), id, repos.IncludeSongInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album: get songs: %w", err))
		return
	}
	songs := mapList(dbSongs, func(s *repos.CompleteSong) *responses.Song {
		return &responses.Song{
			ID:            s.ID,
			IsDir:         false,
			Title:         s.Title,
			Album:         s.AlbumName,
			Track:         s.Track,
			Year:          s.Year,
			CoverArt:      s.CoverID,
			Size:          s.Size,
			ContentType:   s.ContentType,
			Suffix:        filepath.Ext(s.Path),
			Duration:      int(s.Duration.ToStd().Seconds()),
			BitRate:       s.BitRate,
			SamplingRate:  s.SamplingRate,
			ChannelCount:  s.ChannelCount,
			UserRating:    s.UserRating,
			DiscNumber:    s.Disc,
			Created:       s.Created,
			AlbumID:       s.AlbumID,
			BPM:           s.BPM,
			MusicBrainzID: s.MusicBrainzID,
			Starred:       s.Starred,
			AverageRating: s.AverageRating,
			Genres: mapList(s.Genres, func(g string) *responses.GenreRef {
				return &responses.GenreRef{
					Name: g,
				}
			}),
			Artists: mapList(s.Artists, func(a repos.ArtistRef) *responses.ArtistRef {
				return &responses.ArtistRef{
					ID:   a.ID,
					Name: a.Name,
				}
			}),
			AlbumArtists: mapList(s.AlbumArtists, func(a repos.ArtistRef) *responses.ArtistRef {
				return &responses.ArtistRef{
					ID:   a.ID,
					Name: a.Name,
				}
			}),
			ReplayGain: &responses.ReplayGain{
				TrackGain: s.ReplayGain,
				AlbumGain: s.AlbumReplayGain,
				TrackPeak: s.ReplayGainPeak,
				AlbumPeak: s.AlbumReplayGainPeak,
			},
		}
	})

	err = h.completeSongInfo(r.Context(), songs)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album: %w", err))
		return
	}

	album := &responses.Album{
		ID:            dbAlbum.ID,
		Created:       dbAlbum.Created,
		Title:         dbAlbum.Name,
		Name:          dbAlbum.Name,
		SongCount:     int(dbAlbum.TrackCount),
		Duration:      int(dbAlbum.Duration.ToStd().Seconds()),
		Year:          dbAlbum.Year,
		Starred:       dbAlbum.Starred,
		UserRating:    dbAlbum.UserRating,
		AverageRating: dbAlbum.AverageRating,
		MusicBrainzID: dbAlbum.MusicBrainzID,
		IsCompilation: dbAlbum.IsCompilation,
		RecordLabels: mapList(dbAlbum.RecordLabels, func(label string) *responses.RecordLabel {
			return &responses.RecordLabel{
				Name: label,
			}
		}),
		ReleaseTypes: dbAlbum.ReleaseTypes,
		Genres: mapList(dbAlbum.Genres, func(g string) *responses.GenreRef {
			return &responses.GenreRef{
				Name: g,
			}
		}),
		Artists: mapList(dbAlbum.Artists, func(a repos.ArtistRef) *responses.ArtistRef {
			return &responses.ArtistRef{
				ID:   a.ID,
				Name: a.Name,
			}
		}),
	}
	err = h.completeAlbumInfo([]*responses.Album{album})
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

	dbGenres, err := h.DB.Genre().FindAllWithCounts(r.Context())
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get genres: %w", err))
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

	artists, err := h.DB.Artist().FindAll(r.Context(), true, repos.IncludeArtistInfoFull(query.Get("u")))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get artists: %w", err))
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
		var coverArt *string
		if hasCoverArt(a.ID) {
			coverArt = &a.ID
		}
		artist := &responses.Artist{
			ID:            a.ID,
			Name:          a.Name,
			CoverArt:      coverArt,
			AlbumCount:    &a.AlbumCount,
			Starred:       a.Starred,
			MusicBrainzID: a.MusicBrainzID,
			UserRating:    a.UserRating,
			AverageRating: a.AverageRating,
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
