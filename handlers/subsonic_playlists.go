package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	crossonic "github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

func (h *Handler) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	dbPlaylists, err := h.DB.Playlist().FindAll(r.Context(), user, repos.IncludePlaylistInfoFull())
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("find all playlists: %w", err))
		return
	}
	playlists := make([]*responses.Playlist, 0, len(dbPlaylists))
	for _, p := range dbPlaylists {
		var cover *string
		if hasCoverArt(p.ID) {
			cover = &p.ID
		}
		playlists = append(playlists, &responses.Playlist{
			ID:        p.ID,
			Name:      p.Name,
			Comment:   p.Comment,
			Owner:     p.Owner,
			Public:    p.Public,
			SongCount: int(p.TrackCount),
			Duration:  int(p.Duration.ToStd().Seconds()),
			Created:   p.Created,
			Changed:   p.Updated,
			CoverArt:  cover,
		})
	}
	response := responses.New()
	response.Playlists = &responses.Playlists{
		Playlists: playlists,
	}
	response.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetPlaylist(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	playlist, err := h.getPlaylistById(r.Context(), id, user)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("get playlist: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}
	response := responses.New()
	response.Playlist = playlist
	response.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	id := query.Get("playlistId")
	name := query.Get("name")
	if id == "" && name == "" {
		responses.EncodeError(w, query.Get("f"), "missing name parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	err := h.DB.Transaction(r.Context(), func(tx repos.Tx) error {
		if id == "" {
			p, err := tx.Playlist().Create(r.Context(), repos.CreatePlaylistParams{
				Name:   name,
				Owner:  user,
				Public: false,
			})
			if err != nil {
				return fmt.Errorf("create: %w", err)
			}
			id = p.ID
		} else if name != "" {
			err := tx.Playlist().Update(r.Context(), user, id, repos.UpdatePlaylistParams{
				Name: repos.NewOptionalFull(name),
			})
			if err != nil {
				return fmt.Errorf("update name: %w", err)
			}
		} else {
			err := tx.Playlist().Update(r.Context(), user, id, repos.UpdatePlaylistParams{})
			if err != nil {
				return fmt.Errorf("update: %w", err)
			}
		}

		err := tx.Playlist().ClearTracks(r.Context(), id)
		if err != nil {
			return fmt.Errorf("remove old tracks: %w", err)
		}

		if query.Has("songId") {
			err = tx.Playlist().AddTracks(r.Context(), id, query["songId"])
			if err != nil {
				return fmt.Errorf("add new tracks: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("create playlist: %w", err))
		return
	}

	playlist, err := h.getPlaylistById(r.Context(), id, user)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("create playlist: get playlist by id: %w", err))
		}
		return
	}
	response := responses.New()
	response.Playlist = playlist
	response.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	id := query.Get("playlistId")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	err := h.DB.Transaction(r.Context(), func(tx repos.Tx) error {
		playlist, err := tx.Playlist().FindByID(r.Context(), user, id, repos.IncludePlaylistInfo{
			TrackInfo: true,
		})
		if err != nil {
			if errors.Is(err, repos.ErrNotFound) {
				responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			} else {
				log.Errorf("update playlist: find playlist: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			}
			return fmt.Errorf("find playlist: %w", err)
		}

		name := query.Get("name")

		var comment *string
		var updateComment bool
		if query.Has("comment") {
			updateComment = true
			qComment := query.Get("comment")
			if qComment == "" {
				comment = nil
			} else {
				comment = &qComment
			}
		}

		err = tx.Playlist().Update(r.Context(), user, id, repos.UpdatePlaylistParams{
			Name:    repos.NewOptional(name, name != ""),
			Comment: repos.NewOptional(comment, updateComment),
		})
		if err != nil {
			return fmt.Errorf("update playlist: %w", err)
		}

		if query.Has("songIndexToRemove") {
			tracks := make([]int, 0, len(query["songIndexToRemove"]))
			for _, iStr := range query["songIndexToRemove"] {
				i, err := strconv.Atoi(iStr)
				if err != nil || i < 0 || i >= int(playlist.TrackCount) {
					return fmt.Errorf("invalid song index: %s", iStr)
				}
				tracks = append(tracks, i)
			}
			err = tx.Playlist().RemoveTracks(r.Context(), id, tracks)
			if err != nil {
				return fmt.Errorf("remove tracks: %w", err)
			}
		}

		if query.Has("songIdToAdd") {
			err = tx.Playlist().AddTracks(r.Context(), id, query["songIdToAdd"])
			if err != nil {
				return fmt.Errorf("add tracks: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("update playlist: %w", err))
		}
		return
	}

	response := responses.New()
	response.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	err := h.DB.Playlist().Delete(r.Context(), user, id)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("delete playlist: %w", err))
		}
		return
	}

	if hasCoverArt(id) && crossonic.IDRegex.MatchString(id) {
		err = os.Remove(filepath.Join(config.DataDir(), "covers", "playlists", id))
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
	}

	response := responses.New()
	response.EncodeOrLog(w, query.Get("f"))
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

	songs := mapList(tracks, func(s *repos.CompleteSong) *responses.Song {
		return &responses.Song{
			ID:            s.ID,
			IsDir:         false,
			Title:         s.Title,
			Album:         s.AlbumName,
			Track:         s.Track,
			Year:          s.Year,
			CoverArt:      s.CoverID,
			Size:          s.Size,
			ContentType:   s.ContentType,
			Suffix:        filepath.Ext(s.Path),
			Duration:      int(s.Duration.ToStd().Seconds()),
			BitRate:       s.BitRate,
			SamplingRate:  s.SamplingRate,
			ChannelCount:  s.ChannelCount,
			UserRating:    s.UserRating,
			DiscNumber:    s.Disc,
			Created:       s.Created,
			AlbumID:       s.AlbumID,
			BPM:           s.BPM,
			MusicBrainzID: s.MusicBrainzID,
			Starred:       s.Starred,
			AverageRating: s.AverageRating,
			ReplayGain: &responses.ReplayGain{
				TrackGain: s.ReplayGain,
				AlbumGain: s.AlbumReplayGain,
				TrackPeak: s.ReplayGainPeak,
				AlbumPeak: s.AlbumReplayGainPeak,
			},
			Genres: mapList(s.Genres, func(g string) *responses.GenreRef {
				return &responses.GenreRef{
					Name: g,
				}
			}),
			Artists: mapList(s.Artists, func(a repos.ArtistRef) *responses.ArtistRef {
				return &responses.ArtistRef{
					ID:   a.ID,
					Name: a.Name,
				}
			}),
			AlbumArtists: mapList(s.AlbumArtists, func(a repos.ArtistRef) *responses.ArtistRef {
				return &responses.ArtistRef{
					ID:   a.ID,
					Name: a.Name,
				}
			}),
		}
	})

	err = h.completeSongInfo(ctx, songs)
	if err != nil {
		return nil, fmt.Errorf("get playlist by id: %w", err)
	}

	var cover *string
	if hasCoverArt(id) {
		cover = &id
	}
	return &responses.Playlist{
		ID:        dbPlaylist.ID,
		Name:      dbPlaylist.Name,
		Comment:   dbPlaylist.Comment,
		Owner:     dbPlaylist.Owner,
		Public:    dbPlaylist.Public,
		SongCount: dbPlaylist.TrackCount,
		Duration:  int(dbPlaylist.Duration.ToStd().Seconds()),
		Created:   dbPlaylist.Created,
		Changed:   dbPlaylist.Updated,
		CoverArt:  cover,
		Entry:     &songs,
	}, nil
}
