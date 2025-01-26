package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

var maxSearchResultCount = 500

func (h *Handler) handleSearch3(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	query.Set("query", strings.Trim(query.Get("query"), `"`))

	artists, ok := h.searchArtists(w, r)
	if !ok {
		return
	}

	albums, ok := h.searchAlbums(w, r)
	if !ok {
		return
	}

	songs, ok := h.searchSongs(w, r)
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

func (h *Handler) searchArtists(w http.ResponseWriter, r *http.Request) ([]*responses.Artist, bool) {
	user := user(r)
	format := format(r)
	query := getQuery(r)

	limit, ok := paramLimitReq(w, r, "artistCount", &maxSearchResultCount, 20)
	if !ok {
		return nil, false
	}

	offset, ok := paramOffset(w, r, "artistOffset")
	if !ok {
		return nil, false
	}

	artists, err := h.DB.Artist().FindBySearch(r.Context(), query.Get("query"), true, repos.Paginate{Offset: offset, Limit: &limit}, repos.IncludeArtistInfoFull(user))
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("search3: artists: %w", err))
		return nil, false
	}
	return responses.NewArtists(artists), true
}

func (h *Handler) searchAlbums(w http.ResponseWriter, r *http.Request) ([]*responses.Album, bool) {
	user := user(r)
	format := format(r)
	query := getQuery(r)

	limit, ok := paramLimitReq(w, r, "albumCount", &maxSearchResultCount, 20)
	if !ok {
		return nil, false
	}

	offset, ok := paramOffset(w, r, "albumOffset")
	if !ok {
		return nil, false
	}

	dbAlbums, err := h.DB.Album().FindBySearch(r.Context(), query.Get("query"), repos.Paginate{Offset: offset, Limit: &limit}, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		log.Errorf("search3: albums: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		respondInternalErr(w, format, fmt.Errorf("search3: albums: %w", err))
		return nil, false
	}
	albums := responses.NewAlbums(dbAlbums)
	return albums, true
}

func (h *Handler) searchSongs(w http.ResponseWriter, r *http.Request) ([]*responses.Song, bool) {
	user := user(r)
	format := format(r)
	query := getQuery(r)

	limit, ok := paramLimitReq(w, r, "songCount", &maxSearchResultCount, 20)
	if !ok {
		return nil, false
	}

	offset, ok := paramOffset(w, r, "songOffset")
	if !ok {
		return nil, false
	}

	dbSongs, err := h.DB.Song().FindBySearch(r.Context(), repos.SongFindBySearchParams{
		Query: query.Get("query"),
		Paginate: repos.Paginate{
			Offset: offset,
			Limit:  &limit,
		},
	}, repos.IncludeSongInfoFull(user))
	if err != nil {
		respondInternalErr(w, format, fmt.Errorf("search3: songs: %w", err))
		return nil, false
	}
	songs := responses.NewSongs(dbSongs)
	return songs, true
}
