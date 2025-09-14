package handlers

import (
	"fmt"
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

func (h *Handler) handleGetSongs(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	search := q.Str("search")

	onlyStarred, ok := q.Bool("starred", false)
	if !ok {
		return
	}

	minBPM, ok := q.Int("minBpm")
	if !ok {
		return
	}
	maxBPM, ok := q.Int("maxBpm")
	if !ok {
		return
	}

	fromYear, ok := q.Int("fromYear")
	if !ok {
		return
	}
	toYear, ok := q.Int("toYear")
	if !ok {
		return
	}

	genres := q.Strs("genre")

	artistIDs, ok := q.IDs("artistId")
	if !ok {
		return
	}
	albumIDs, ok := q.IDs("albumId")
	if !ok {
		return
	}

	orderBy := util.NilIfEmpty(repos.SongOrder(q.Str("orderBy")))
	if orderBy != nil && !orderBy.Valid() {
		responses.EncodeError(q.responseWriter, q.Format(), "invalid orderBy parameter", responses.SubsonicErrorGeneric)
		return
	}

	orderDesc, ok := q.Bool("orderByDesc", false)
	if !ok {
		return
	}

	randomSeed := util.NilIfEmpty(q.Str("seed"))

	paginate, ok := q.Paginate("count", "offset", 10)
	if !ok {
		return
	}

	songs, err := h.DB.Song().FindAllFiltered(r.Context(), repos.SongFindAllFilter{
		Search:      search,
		OnlyStarred: onlyStarred,
		MinBPM:      minBPM,
		MaxBPM:      maxBPM,
		FromYear:    fromYear,
		ToYear:      toYear,
		Genres:      genres,
		ArtistIDs:   artistIDs,
		AlbumIDs:    albumIDs,
		Order:       orderBy,
		OrderDesc:   orderDesc,
		RandomSeed:  randomSeed,
		Paginate:    paginate,
	}, repos.IncludeSongInfoFull(q.User()))
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("find all songs filtered: %w", err))
		return
	}

	response := responses.New()
	response.Songs = &responses.Songs{
		Songs: responses.NewSongs(songs, h.Config),
	}
	response.EncodeOrLog(w, q.Format())
}
