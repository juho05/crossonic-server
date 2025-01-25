package handlers

import (
	"errors"
	"fmt"
	"net/http"
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
	dbArtist, err := h.DB.Artist().FindByID(r.Context(), id, repos.IncludeArtistInfoFull(user))
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("get artist: %w", err))
		}
		return
	}
	dbAlbums, err := h.DB.Artist().GetAlbums(r.Context(), dbArtist.ID, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get artist: get albums: %w", err))
		return
	}

	albums := responses.NewAlbums(dbAlbums)

	artist := responses.NewArtist(dbArtist)
	artist.Albums = albums

	res := responses.New()
	res.Artist = artist
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

	songs := responses.NewSongs(dbSongs)
	album := responses.NewAlbum(dbAlbum)

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
	for _, a := range artists {
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

		artist := responses.NewArtist(a)

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
