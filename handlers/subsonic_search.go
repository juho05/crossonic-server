package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
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

	artists, err := h.DB.Artist().FindBySearch(ctx, query.Get("query"), true, offset, limit, repos.IncludeArtistInfoFull(user))
	if err != nil {
		log.Errorf("search3: artists: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
	}
	return responses.NewArtists(artists), true
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

	dbAlbums, err := h.DB.Album().FindBySearchQuery(ctx, query.Get("query"), offset, limit, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		log.Errorf("search3: albums: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
	}
	albums := responses.NewAlbums(dbAlbums)
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

	dbSongs, err := h.DB.Song().FindBySearchQuery(ctx, repos.SongFindBySearchParams{
		Query:  query.Get("query"),
		Offset: offset,
		Limit:  limit,
	}, repos.IncludeSongInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("search3: songs: %w", err))
		return nil, false
	}
	songs := responses.NewSongs(dbSongs)
	return songs, true
}
