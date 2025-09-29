package handlers

import (
	"fmt"
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

func (h *Handler) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	limit, ok := q.IntRange("size", 0, maxListSize)
	if !ok {
		return
	}
	if limit == nil {
		limit = util.ToPtr(10)
	}

	fromYear, ok := q.Int("fromYear")
	toYear, ok := q.Int("toYear")

	if fromYear != nil && toYear != nil && *fromYear > *toYear {
		*fromYear, *toYear = *toYear, *fromYear
	}

	genres := q.Strs("genre")

	dbSongs, err := h.DB.Song().FindAllFiltered(r.Context(), repos.SongFindAllFilter{
		Order:    util.ToPtr(repos.SongOrderRandom),
		FromYear: fromYear,
		ToYear:   toYear,
		Genres:   genres,
		Paginate: repos.Paginate{
			Limit: limit,
		},
	}, repos.IncludeSongInfoFull(q.User()))
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get random songs: %w", err))
		return
	}

	songs := responses.NewSongs(dbSongs, h.Config)
	res := responses.New()
	res.RandomSongs = &responses.RandomSongs{
		Songs: songs,
	}
	res.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleGetAlbumList(version int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		q := getQuery(w, r)

		listType, ok := q.StrReq("type")
		if !ok {
			return
		}

		paginate, ok := q.Paginate("size", "offset", 10)
		if !ok {
			return
		}

		fromYear, ok := q.Int("fromYear")
		if !ok {
			return
		}
		if listType == "byYear" && fromYear == nil {
			responses.EncodeError(w, q.Format(), "missing fromYear parameter", responses.SubsonicErrorRequiredParameterMissing)
			return
		}

		toYear, ok := q.Int("toYear")
		if !ok {
			return
		}
		if listType == "byYear" && toYear == nil {
			responses.EncodeError(w, q.Format(), "missing toYear parameter", responses.SubsonicErrorRequiredParameterMissing)
			return
		}

		if fromYear != nil && toYear != nil && *fromYear > *toYear {
			*fromYear, *toYear = *toYear, *fromYear
		}

		genres := q.Strs("genre")
		if listType == "byGenre" && len(genres) == 0 {
			responses.EncodeError(w, q.Format(), "missing genre parameter", responses.SubsonicErrorRequiredParameterMissing)
			return
		}

		sortTypes := map[string]repos.FindAlbumSortBy{
			"random":             repos.FindAlbumSortRandom,
			"newest":             repos.FindAlbumSortByCreated,
			"highest":            repos.FindAlbumSortByRating,
			"alphabeticalByName": repos.FindAlbumSortByName,
			"starred":            repos.FindAlbumSortByStarred,
			"byYear":             repos.FindAlbumSortByOriginalDate,
			"byGenre":            repos.FindAlbumSortByName,
			"frequent":           repos.FindAlbumSortByFrequent,
			"recent":             repos.FindAlbumSortByRecent,
		}

		sortBy, ok := sortTypes[listType]
		if !ok {
			responses.EncodeError(w, q.Format(), "unsupported list type: "+listType, responses.SubsonicErrorGeneric)
			return
		}

		a, err := h.DB.Album().FindAll(r.Context(), repos.FindAlbumParams{
			SortBy:   sortBy,
			FromYear: fromYear,
			ToYear:   toYear,
			Genres:   genres,
			Paginate: paginate,
		}, repos.IncludeAlbumInfoFull(q.User()))
		if err != nil {
			respondInternalErr(w, q.Format(), fmt.Errorf("get album list 2: find all: %w", err))
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
		res.EncodeOrLog(w, q.Format())
	}
}

func (h *Handler) handleGetStarred(version int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		q := getQuery(w, r)

		songPaginate, ok := q.PaginateUnlimited("songCount", "songOffset")
		if !ok {
			return
		}

		albumPaginate, ok := q.PaginateUnlimited("albumCount", "albumOffset")
		if !ok {
			return
		}

		artistPaginate, ok := q.PaginateUnlimited("artistCount", "artistOffset")
		if !ok {
			return
		}

		songs, err := h.DB.Song().FindAllFiltered(r.Context(), repos.SongFindAllFilter{
			Order:       util.ToPtr(repos.SongOrderStarred),
			OrderDesc:   true,
			OnlyStarred: true,
			Paginate:    songPaginate,
		}, repos.IncludeSongInfoFull(q.User()))
		if err != nil {
			respondInternalErr(w, q.Format(), fmt.Errorf("get starred2: find songs: %w", err))
			return
		}

		albums, err := h.DB.Album().FindStarred(r.Context(), albumPaginate, repos.IncludeAlbumInfoFull(q.User()))
		if err != nil {
			respondInternalErr(w, q.Format(), fmt.Errorf("get starred2: find albums: %w", err))
			return
		}

		artists, err := h.DB.Artist().FindStarred(r.Context(), artistPaginate, repos.IncludeArtistInfoFull(q.User()))
		if err != nil {
			respondInternalErr(w, q.Format(), fmt.Errorf("get starred2: find artists: %w", err))
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
		res.EncodeOrLog(w, q.Format())
	}
}

func (h *Handler) handleGetSongsByGenre(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	genres, ok := q.StrsReq("genre")
	if !ok {
		return
	}

	paginate, ok := q.Paginate("count", "offset", 10)

	songs, err := h.DB.Song().FindAllFiltered(r.Context(), repos.SongFindAllFilter{
		Order:    util.ToPtr(repos.SongOrderTitle),
		Genres:   genres,
		Paginate: paginate,
	}, repos.IncludeSongInfoFull(q.User()))
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get songs by genre: find songs: %w", err))
		return
	}

	res := responses.New()
	res.SongsByGenre = &responses.SongsByGenre{
		Songs: responses.NewSongs(songs, h.Config),
	}
	res.EncodeOrLog(w, q.Format())
}
