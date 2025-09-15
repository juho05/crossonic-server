package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

func (h *Handler) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	dbPlaylists, err := h.DB.Playlist().FindAll(r.Context(), q.User(), repos.IncludePlaylistInfoFull())
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("find all playlists: %w", err))
		return
	}
	playlists := responses.NewPlaylists(dbPlaylists, h.Config)
	response := responses.New()
	response.Playlists = &responses.Playlists{
		Playlists: playlists,
	}
	response.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleGetPlaylist(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	id, ok := q.IDReq("id")
	if !ok {
		return
	}

	playlist, err := h.getPlaylistById(r.Context(), id, q.User())
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("get playlist: %w", err))
		return
	}
	response := responses.New()
	response.Playlist = playlist
	response.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	id := q.Str("playlistId")
	name := q.Str("name")
	if id == "" && name == "" {
		responses.EncodeError(w, q.Format(), "missing name parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	songIds, ok := q.IDsType("songId", []crossonic.IDType{crossonic.IDTypeSong})
	if !ok {
		return
	}

	err := h.DB.Transaction(r.Context(), func(tx repos.Tx) error {
		if id == "" {
			p, err := tx.Playlist().Create(r.Context(), repos.CreatePlaylistParams{
				Name:   name,
				Owner:  q.User(),
				Public: false,
			})
			if err != nil {
				return fmt.Errorf("create: %w", err)
			}
			id = p.ID
		} else if name != "" {
			err := tx.Playlist().Update(r.Context(), q.User(), id, repos.UpdatePlaylistParams{
				Name: repos.NewOptionalFull(name),
			})
			if err != nil {
				return fmt.Errorf("update name: %w", err)
			}
		} else {
			err := tx.Playlist().Update(r.Context(), q.User(), id, repos.UpdatePlaylistParams{})
			if err != nil {
				return fmt.Errorf("update: %w", err)
			}
		}

		err := tx.Playlist().ClearTracks(r.Context(), id)
		if err != nil {
			return fmt.Errorf("remove old tracks: %w", err)
		}

		if len(songIds) > 0 {
			err = tx.Playlist().AddTracks(r.Context(), id, songIds)
			if err != nil {
				return fmt.Errorf("add new tracks: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("create playlist: %w", err))
		return
	}

	playlist, err := h.getPlaylistById(r.Context(), id, q.User())
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, q.Format(), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, q.Format(), fmt.Errorf("create playlist: get playlist by id: %w", err))
		}
		return
	}
	response := responses.New()
	response.Playlist = playlist
	response.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	id, ok := q.IDReq("playlistId")
	if !ok {
		return
	}

	name := q.Str("name")

	comment := util.NilIfEmpty(q.Str("comment"))
	updateComment := q.Has("comment")

	removeIndices, ok := q.Ints("songIndexToRemove")
	if !ok {
		return
	}

	songIdsToAdd, ok := q.IDsType("songIdToAdd", []crossonic.IDType{crossonic.IDTypeSong})
	if !ok {
		return
	}

	err := h.DB.Transaction(r.Context(), func(tx repos.Tx) error {
		playlist, err := tx.Playlist().FindByID(r.Context(), q.User(), id, repos.IncludePlaylistInfo{
			TrackInfo: true,
		})
		if err != nil {
			return fmt.Errorf("find playlist: %w", err)
		}

		err = tx.Playlist().Update(r.Context(), q.User(), id, repos.UpdatePlaylistParams{
			Name:    repos.NewOptional(name, name != ""),
			Comment: repos.NewOptional(comment, updateComment),
		})
		if err != nil {
			return fmt.Errorf("update playlist: %w", err)
		}

		if len(removeIndices) > 0 {
			for _, i := range removeIndices {
				if i < 0 || i >= playlist.TrackCount {
					return fmt.Errorf("invalid remove index: %d, track count: %d", i, playlist.TrackCount)
				}
			}
			err = tx.Playlist().RemoveTracks(r.Context(), id, removeIndices)
			if err != nil {
				return fmt.Errorf("remove tracks: %w", err)
			}
		}

		if len(songIdsToAdd) > 0 {
			err = tx.Playlist().AddTracks(r.Context(), id, songIdsToAdd)
			if err != nil {
				return fmt.Errorf("add tracks: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("update playlist: %w", err))
		return
	}

	response := responses.New()
	response.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	id, ok := q.IDReq("id")
	if !ok {
		return
	}

	err := h.DB.Playlist().Delete(r.Context(), q.User(), id)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, q.Format(), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, q.Format(), fmt.Errorf("delete playlist: %w", err))
		}
		return
	}

	err = os.Remove(filepath.Join(h.Config.DataDir, "covers", id))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Errorf("delete playlist: delete cover file: %s", err)
	}
	for _, k := range h.CoverCache.Keys() {
		parts := strings.Split(k, "-")
		pID := strings.Join(parts[:len(parts)-1], "-")
		if pID == id {
			err = h.CoverCache.DeleteObject(k)
			if err != nil {
				log.Errorf("delete playlist: delete cover cache: %s", err)
			}
		}
	}

	response := responses.New()
	response.EncodeOrLog(w, q.Format())
}

func (h *Handler) getPlaylistById(ctx context.Context, id, user string) (*responses.Playlist, error) {
	dbPlaylist, err := h.DB.Playlist().FindByID(ctx, user, id, repos.IncludePlaylistInfoFull())
	if err != nil {
		return nil, fmt.Errorf("get playlist by id: find playlist: %w", err)
	}
	tracks, err := h.DB.Playlist().GetTracks(ctx, id, repos.IncludeSongInfoFull(user))
	if err != nil {
		return nil, fmt.Errorf("get playlist by id: get tracks: %w", err)
	}

	songs := responses.NewSongs(tracks, h.Config)

	playlist := responses.NewPlaylist(dbPlaylist, h.Config)
	playlist.Entry = songs
	return playlist, nil
}
