package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

func (h *Handler) handleSearch3(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	search := strings.Trim(q.Str("query"), `"`)

	musicFolderIDs, ok := q.MusicFolderIDs(r.Context(), h.DB)
	if !ok {
		return
	}

	artists, ok := h.searchArtists(w, r, search, musicFolderIDs)
	if !ok {
		return
	}

	albums, ok := h.searchAlbums(w, r, search, musicFolderIDs)
	if !ok {
		return
	}

	songs, ok := h.searchSongs(w, r, search, musicFolderIDs)
	if !ok {
		return
	}

	res := responses.New()
	res.SearchResult3 = &responses.SearchResult3{
		Songs:   songs,
		Albums:  albums,
		Artists: artists,
	}
	res.EncodeOrLog(w, q.Format())
}

func (h *Handler) searchArtists(w http.ResponseWriter, r *http.Request, searchQuery string, musicFolderIDs []int) ([]*responses.Artist, bool) {
	q := getQuery(w, r)

	paginate, ok := q.Paginate("artistCount", "artistOffset", 20)
	if !ok {
		return nil, false
	}

	onlyAlbumArtists, ok := q.BoolDef("onlyAlbumArtists", true)
	if !ok {
		return nil, false
	}

	artists, err := h.DB.Artist().FindBySearch(r.Context(), searchQuery, onlyAlbumArtists, musicFolderIDs, paginate, repos.IncludeArtistInfoFull(q.User()))
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("search3: artists: %w", err))
		return nil, false
	}
	return responses.NewArtists(artists, h.Config), true
}

func (h *Handler) searchAlbums(w http.ResponseWriter, r *http.Request, searchQuery string, musicFolderIDs []int) ([]*responses.Album, bool) {
	q := getQuery(w, r)

	paginate, ok := q.Paginate("albumCount", "albumOffset", 20)
	if !ok {
		return nil, false
	}

	dbAlbums, err := h.DB.Album().FindBySearch(r.Context(), searchQuery, musicFolderIDs, paginate, repos.IncludeAlbumInfoFull(q.User()))
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("search3: albums: %w", err))
		return nil, false
	}
	albums := responses.NewAlbums(dbAlbums, h.Config)
	return albums, true
}

func (h *Handler) searchSongs(w http.ResponseWriter, r *http.Request, searchQuery string, musicFolderIDs []int) ([]*responses.Song, bool) {
	q := getQuery(w, r)

	paginate, ok := q.Paginate("songCount", "songOffset", 20)
	if !ok {
		return nil, false
	}

	var order *repos.SongOrder
	if searchQuery == "" {
		order = util.ToPtr(repos.SongOrderTitle)
	}

	dbSongs, err := h.DB.Song().FindAllFiltered(r.Context(), repos.SongFindAllFilter{
		Search:         searchQuery,
		Paginate:       paginate,
		Order:          order,
		MusicFolderIDs: musicFolderIDs,
	}, repos.IncludeSongInfoFull(q.User()))
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("search3: songs: %w", err))
		return nil, false
	}
	songs := responses.NewSongs(dbSongs, h.Config)
	return songs, true
}
