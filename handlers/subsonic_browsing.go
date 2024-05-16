package handlers

import (
	"math"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

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

	artists, err := h.Store.FindArtists(r.Context(), query.Get("u"))
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
