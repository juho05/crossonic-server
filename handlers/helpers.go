package handlers

import (
	"context"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func (h *Handler) completeSongInfo(ctx context.Context, songs []*responses.Song) error {
	for _, s := range songs {
		if len(s.Genres) > 0 {
			s.Genre = &s.Genres[0].Name
		}

		if len(s.Artists) > 0 {
			s.ArtistID = &s.Artists[0].ID
			s.Artist = &s.Artists[0].Name
		}

		if len(s.AlbumArtists) > 0 && s.ArtistID == nil && s.Artist == nil {
			s.ArtistID = &s.AlbumArtists[0].ID
			s.Artist = &s.AlbumArtists[0].Name
		}

		s.Type = "music"
		s.MediaType = "song"
		if s.ReplayGain == nil {
			s.ReplayGain = &responses.ReplayGain{}
		}
		fallbackGain := h.DB.Song().GetFallbackGain(ctx)
		s.ReplayGain.FallbackGain = &fallbackGain
	}
	return nil
}

func (h *Handler) completeAlbumInfo(albums []*responses.Album) error {
	for _, a := range albums {
		if len(a.Genres) > 0 {
			a.Genre = &a.Genres[0].Name
		}

		if len(a.Artists) > 0 {
			a.ArtistID = &a.Artists[0].ID
			a.Artist = &a.Artists[0].Name
		}

		if hasCoverArt(a.ID) {
			a.CoverArt = &a.ID
		}

		a.IsDir = true
		a.Type = "music"
		a.MediaType = "album"
	}
	return nil
}

func respondInternalErr(w http.ResponseWriter, format string, err error) {
	log.Error(err)
	responses.EncodeError(w, format, http.StatusText(http.StatusInternalServerError), responses.SubsonicErrorGeneric)
}

func getQuery(r *http.Request) url.Values {
	query, ok := r.Context().Value(ContextKeyQuery).(url.Values)
	if !ok {
		panic("getQuery must be called after subsonicMiddleware")
	}
	return query
}

func registerRoute(r chi.Router, pattern string, handlerFunc func(w http.ResponseWriter, r *http.Request)) {
	r.Get(pattern, handlerFunc)
	r.Post(pattern, handlerFunc)
	r.Get(pattern+".view", handlerFunc)
	r.Post(pattern+".view", handlerFunc)
}

func mapList[T, U any](list []T, mapFn func(T) U) []U {
	if list == nil {
		return nil
	}
	newList := make([]U, len(list))
	for i := range list {
		newList[i] = mapFn(list[i])
	}
	return newList
}
