package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func (h *Handler) completeSongInfo(ctx context.Context, songs []*responses.Song) error {
	songIDs := mapData(songs, func(s *responses.Song) string {
		return s.ID
	})
	var genres map[string][]*responses.GenreRef
	genres, err := h.loadSongGenresRefs(ctx, songIDs)
	if err != nil {
		return fmt.Errorf("complete song info: %w", err)
	}
	artists, err := h.loadSongArtistsRefs(ctx, songIDs)
	if err != nil {
		return fmt.Errorf("complete song info: %w", err)
	}
	albumArtists, err := h.loadSongAlbumArtistRefs(ctx, songIDs)
	if err != nil {
		return fmt.Errorf("complete song info: %w", err)
	}
	for _, s := range songs {
		s.Genres = genres[s.ID]
		if len(s.Genres) > 0 {
			s.Genre = &s.Genres[0].Name
		}

		s.Artists = artists[s.ID]
		if len(s.Artists) > 0 {
			s.ArtistID = &s.Artists[0].ID
			s.Artist = &s.Artists[0].Name
		}

		s.AlbumArtists = albumArtists[s.ID]
		if len(s.AlbumArtists) > 0 && s.ArtistID == nil && s.Artist == nil {
			s.ArtistID = &s.AlbumArtists[0].ID
			s.Artist = &s.AlbumArtists[0].Name
		}

		s.Type = "music"
		s.MediaType = "song"
	}
	return nil
}

func (h *Handler) loadSongGenresRefs(ctx context.Context, songIDs []string) (map[string][]*responses.GenreRef, error) {
	result := make(map[string][]*responses.GenreRef)

	genres, err := h.Store.FindGenresBySongs(ctx, songIDs)
	if err != nil {
		return nil, fmt.Errorf("load genre refs: %w", err)
	}
	for _, g := range genres {
		if _, ok := result[g.SongID]; !ok {
			result[g.SongID] = make([]*responses.GenreRef, 0, 1)
		}
		result[g.SongID] = append(result[g.SongID], &responses.GenreRef{
			Name: g.Name,
		})
	}
	return result, nil
}

func (h *Handler) loadSongArtistsRefs(ctx context.Context, songIDs []string) (map[string][]*responses.ArtistRef, error) {
	result := make(map[string][]*responses.ArtistRef)

	artists, err := h.Store.FindArtistRefsBySongs(ctx, songIDs)
	if err != nil {
		return nil, fmt.Errorf("load artist refs: %w", err)
	}
	for _, a := range artists {
		if _, ok := result[a.SongID]; !ok {
			result[a.SongID] = make([]*responses.ArtistRef, 0, 1)
		}
		result[a.SongID] = append(result[a.SongID], &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		})
	}
	return result, nil
}

func (h *Handler) loadSongAlbumArtistRefs(ctx context.Context, songIDs []string) (map[string][]*responses.ArtistRef, error) {
	result := make(map[string][]*responses.ArtistRef)

	albumArtists, err := h.Store.FindAlbumArtistRefsBySongs(ctx, songIDs)
	if err != nil {
		return nil, fmt.Errorf("load album artist refs: %w", err)
	}
	for _, a := range albumArtists {
		if _, ok := result[a.SongID]; !ok {
			result[a.SongID] = make([]*responses.ArtistRef, 0, 1)
		}
		result[a.SongID] = append(result[a.SongID], &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		})
	}
	return result, nil
}

func (h *Handler) completeAlbumInfo(ctx context.Context, albums []*responses.Album) error {
	albumIDs := mapData(albums, func(a *responses.Album) string {
		return a.ID
	})

	genres, err := h.loadAlbumGenreRefs(ctx, albumIDs)
	if err != nil {
		return fmt.Errorf("complete album info: %w", err)
	}

	artists, err := h.loadAlbumArtistRefs(ctx, albumIDs)
	if err != nil {
		return fmt.Errorf("complete album info: %w", err)
	}

	for _, a := range albums {
		a.Genres = genres[a.ID]
		if len(a.Genres) > 0 {
			a.Genre = &a.Genres[0].Name
		}

		a.Artists = artists[a.ID]
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

func (h *Handler) loadAlbumGenreRefs(ctx context.Context, albumIDs []string) (map[string][]*responses.GenreRef, error) {
	result := make(map[string][]*responses.GenreRef)

	genreRefs, err := h.Store.FindGenresByAlbums(ctx, albumIDs)
	if err != nil {
		return nil, fmt.Errorf("load album genre refs: %w", err)
	}
	for _, g := range genreRefs {
		if _, ok := result[g.AlbumID]; !ok {
			result[g.AlbumID] = make([]*responses.GenreRef, 0, 1)
		}
		result[g.AlbumID] = append(result[g.AlbumID], &responses.GenreRef{
			Name: g.Name,
		})
	}
	return result, nil
}

func (h *Handler) loadAlbumArtistRefs(ctx context.Context, albumIDs []string) (map[string][]*responses.ArtistRef, error) {
	result := make(map[string][]*responses.ArtistRef)

	artistRefs, err := h.Store.FindArtistRefsByAlbums(ctx, albumIDs)
	if err != nil {
		return nil, fmt.Errorf("load album artist refs: %w", err)
	}
	for _, a := range artistRefs {
		if _, ok := result[a.AlbumID]; !ok {
			result[a.AlbumID] = make([]*responses.ArtistRef, 0, 1)
		}
		result[a.AlbumID] = append(result[a.AlbumID], &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		})
	}
	return result, nil
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

func mapData[T, U any](list []T, mapFn func(T) U) []U {
	if list == nil {
		return nil
	}
	newList := make([]U, len(list))
	for i := range list {
		newList[i] = mapFn(list[i])
	}
	return newList
}
