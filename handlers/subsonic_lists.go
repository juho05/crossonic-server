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

	limit, ok := paramLimitReq(w, r, "size", &maxListSize, 10)
	if !ok {
		return
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

	if fromYear != nil && toYear != nil && *fromYear > *toYear {
		*fromYear, *toYear = *toYear, *fromYear
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

	songs := responses.NewSongs(dbSongs, h.Config)
	res := responses.New()
	res.RandomSongs = &responses.RandomSongs{
		Songs: songs,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetAlbumList(version int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := getQuery(r)
		user := query.Get("u")

		listType := query.Get("type")

		limit, ok := paramLimitReq(w, r, "size", &maxListSize, 10)
		if !ok {
			return
		}

		offset, ok := paramOffset(w, r, "offset")
		if !ok {
			return
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

		if fromYear != nil && toYear != nil && *fromYear > *toYear {
			*fromYear, *toYear = *toYear, *fromYear
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
			"frequent":           repos.FindAlbumSortByFrequent,
			"recent":             repos.FindAlbumSortByRecent,
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
				Limit:  &limit,
			},
		}, repos.IncludeAlbumInfoFull(user))
		if err != nil {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("get album list 2: find all: %w", err))
			return
		}

		albums := responses.NewAlbums(a, h.Config)
		res := responses.New()
		if version == 2 {
			res.AlbumList2 = &responses.AlbumList2{
				Albums: albums,
			}
		} else {
			res.AlbumList = &responses.AlbumList{
				Albums: albums,
			}
		}
		res.EncodeOrLog(w, query.Get("f"))
	}
}

func (h *Handler) handleGetStarred(version int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user := user(r)
		f := format(r)

		songLimit, ok := paramLimitOpt(w, r, "songCount", nil)
		if !ok {
			return
		}
		songOffset, ok := paramOffset(w, r, "songOffset")
		if !ok {
			return
		}

		albumLimit, ok := paramLimitOpt(w, r, "albumCount", nil)
		if !ok {
			return
		}
		albumOffset, ok := paramOffset(w, r, "albumOffset")
		if !ok {
			return
		}

		artistLimit, ok := paramLimitOpt(w, r, "artistCount", nil)
		if !ok {
			return
		}
		artistOffset, ok := paramOffset(w, r, "artistOffset")
		if !ok {
			return
		}

		songs, err := h.DB.Song().FindStarred(r.Context(), repos.Paginate{
			Offset: songOffset,
			Limit:  songLimit,
		}, repos.IncludeSongInfoFull(user))
		if err != nil {
			respondInternalErr(w, f, fmt.Errorf("get starred2: find songs: %w", err))
			return
		}

		albums, err := h.DB.Album().FindStarred(r.Context(), repos.Paginate{
			Offset: albumOffset,
			Limit:  albumLimit,
		}, repos.IncludeAlbumInfoFull(user))
		if err != nil {
			respondInternalErr(w, f, fmt.Errorf("get starred2: find albums: %w", err))
			return
		}

		artists, err := h.DB.Artist().FindStarred(r.Context(), repos.Paginate{
			Offset: artistOffset,
			Limit:  artistLimit,
		}, repos.IncludeArtistInfoFull(user))
		if err != nil {
			respondInternalErr(w, f, fmt.Errorf("get starred2: find artists: %w", err))
			return
		}

		res := responses.New()
		if version == 2 {
			res.Starred2 = &responses.Starred2{
				Songs:   responses.NewSongs(songs, h.Config),
				Albums:  responses.NewAlbums(albums, h.Config),
				Artists: responses.NewArtists(artists, h.Config),
			}
		} else {
			res.Starred = &responses.Starred{
				Songs:   responses.NewSongs(songs, h.Config),
				Albums:  responses.NewAlbums(albums, h.Config),
				Artists: responses.NewArtists(artists, h.Config),
			}
		}
		res.EncodeOrLog(w, f)
	}
}

func (h *Handler) handleGetSongsByGenre(w http.ResponseWriter, r *http.Request) {
	genre, ok := paramStrReq(w, r, "genre")
	if !ok {
		return
	}
	limit, ok := paramLimitReq(w, r, "count", &maxListSize, 10)
	if !ok {
		return
	}
	offset, ok := paramOffset(w, r, "offset")
	if !ok {
		return
	}

	songs, err := h.DB.Song().FindByGenre(r.Context(), genre, repos.Paginate{
		Offset: offset,
		Limit:  &limit,
	}, repos.IncludeSongInfoFull(user(r)))
	if err != nil {
		respondInternalErr(w, format(r), fmt.Errorf("get songs by genre: find songs: %w", err))
		return
	}

	res := responses.New()
	res.SongsByGenre = &responses.SongsByGenre{
		Songs: responses.NewSongs(songs, h.Config),
	}
	res.EncodeOrLog(w, format(r))
}
