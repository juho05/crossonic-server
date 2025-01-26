package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
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
	var fromYear *int
	if fromYearStr != "" {
		y, err := strconv.Atoi(fromYearStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid fromYear value", responses.SubsonicErrorGeneric)
			return
		}
		fromYear = &y
	}

	toYearStr := query.Get("toYear")
	var toYear *int
	if toYearStr != "" {
		y, err := strconv.Atoi(toYearStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid toYear value", responses.SubsonicErrorGeneric)
			return
		}
		toYear = &y
	}

	genres := util.Map(query["genre"], func(g string) string {
		return strings.ToLower(g)
	})

	dbSongs, err := h.DB.Song().FindRandom(r.Context(), repos.SongFindRandomParams{
		Limit:    limit,
		FromYear: fromYear,
		ToYear:   toYear,
		Genres:   genres,
	}, repos.IncludeSongInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get random songs: %w", err))
		return
	}

	songs := responses.NewSongs(dbSongs)
	res := responses.New()
	res.RandomSongs = &responses.RandomSongs{
		Songs: songs,
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
		if err != nil || limit <= 0 || limit > 500 {
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
	var fromYear *int
	if fromYearStr != "" {
		y, err := strconv.Atoi(fromYearStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid fromYear value", responses.SubsonicErrorGeneric)
			return
		}
		fromYear = &y
	} else if listType == "byYear" {
		responses.EncodeError(w, query.Get("f"), "missing fromYear parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	toYearStr := query.Get("toYear")
	var toYear *int
	if toYearStr != "" {
		y, err := strconv.Atoi(toYearStr)
		if err != nil {
			responses.EncodeError(w, query.Get("f"), "invalid toYear value", responses.SubsonicErrorGeneric)
			return
		}
		toYear = &y
	} else if listType == "byYear" {
		responses.EncodeError(w, query.Get("f"), "missing toYear parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	genres := query["genre"]
	if listType == "byGenre" && len(genres) == 0 {
		responses.EncodeError(w, query.Get("f"), "missing genre parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	sortTypes := map[string]repos.FindAlbumSortBy{
		"random":             repos.FindAlbumSortRandom,
		"newest":             repos.FindAlbumSortByCreated,
		"highest":            repos.FindAlbumSortByRating,
		"alphabeticalByName": repos.FindAlbumSortByName,
		"starred":            repos.FindAlbumSortByStarred,
		"byYear":             repos.FindAlbumSortByYear,
		"byGenre":            repos.FindAlbumSortByName,
	}

	sortBy, ok := sortTypes[listType]
	if !ok {
		responses.EncodeError(w, query.Get("f"), "unsupported list type: "+listType, responses.SubsonicErrorGeneric)
		return
	}

	a, err := h.DB.Album().FindAll(r.Context(), repos.FindAlbumParams{
		SortBy:   sortBy,
		FromYear: fromYear,
		ToYear:   toYear,
		Genres:   genres,
		Paginate: repos.Paginate{
			Offset: offset,
			Limit:  limit,
		},
	}, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album list 2: find all: %w", err))
		return
	}

	albums := responses.NewAlbums(a)
	res := responses.New()
	res.AlbumList2 = &responses.AlbumList2{
		Albums: albums,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetStarred2(w http.ResponseWriter, r *http.Request) {
	user := user(r)
	f := format(r)

	songLimit, songLimitExists, ok := paramLimit(w, r, "songCount", nil, 0)
	if !ok {
		return
	}
	songOffset, ok := paramOffset(w, r, "songOffset")
	if !ok {
		return
	}

	albumLimit, albumLimitExists, ok := paramLimit(w, r, "albumCount", nil, 0)
	if !ok {
		return
	}
	albumOffset, ok := paramOffset(w, r, "albumOffset")
	if !ok {
		return
	}

	artistLimit, artistLimitExists, ok := paramLimit(w, r, "artistCount", nil, 0)
	if !ok {
		return
	}
	artistOffset, ok := paramOffset(w, r, "artistOffset")
	if !ok {
		return
	}

	songs := []*repos.CompleteSong{}
	var err error
	if !songLimitExists || songLimit > 0 {
		songs, err = h.DB.Song().FindStarred(r.Context(), repos.Paginate{
			Offset: songOffset,
			Limit:  songLimit,
		}, repos.IncludeSongInfoFull(user))
		if err != nil {
			respondInternalErr(w, f, fmt.Errorf("get starred2: find songs: %w", err))
			return
		}
	}

	albums := []*repos.CompleteAlbum{}
	if !albumLimitExists || albumLimit > 0 {
		albums, err = h.DB.Album().FindStarred(r.Context(), repos.Paginate{
			Offset: albumOffset,
			Limit:  albumLimit,
		}, repos.IncludeAlbumInfoFull(user))
		if err != nil {
			respondInternalErr(w, f, fmt.Errorf("get starred2: find albums: %w", err))
			return
		}
	}

	artists := []*repos.CompleteArtist{}
	if !artistLimitExists || artistLimit > 0 {
		artists, err = h.DB.Artist().FindStarred(r.Context(), repos.Paginate{
			Offset: artistOffset,
			Limit:  artistLimit,
		}, repos.IncludeArtistInfoFull(user))
		if err != nil {
			respondInternalErr(w, f, fmt.Errorf("get starred2: find artists: %w", err))
			return
		}
	}

	res := responses.New()
	res.Starred2 = &responses.Starred2{
		Songs:   responses.NewSongs(songs),
		Albums:  responses.NewAlbums(albums),
		Artists: responses.NewArtists(artists),
	}
	res.EncodeOrLog(w, f)
}
