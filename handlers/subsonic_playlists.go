package handlers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/db"
	sqlc "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func (h *Handler) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	dbPlaylists, err := h.Store.FindPlaylists(r.Context(), user)
	if err != nil {
		log.Errorf("get playlists: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
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
			Duration:  int(p.DurationMs / 1000),
			Created:   p.Created.Time,
			Changed:   p.Updated.Time,
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
		if errors.Is(err, pgx.ErrNoRows) {
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

	tx, err := h.Store.BeginTransaction(r.Context())
	if err != nil {
		log.Errorf("create playlist: begin transaction: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer tx.Rollback(r.Context())

	if id == "" {
		id = crossonic.GenIDPlaylist()
		err := tx.CreatePlaylist(r.Context(), sqlc.CreatePlaylistParams{
			ID:     id,
			Name:   name,
			Owner:  user,
			Public: false,
		})
		if err != nil {
			log.Errorf("create playlist: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	} else if name != "" {
		res, err := tx.UpdatePlaylistName(r.Context(), sqlc.UpdatePlaylistNameParams{
			ID:    id,
			Owner: user,
			Name:  name,
		})
		if err != nil {
			log.Errorf("create playlist (update): %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		if res.RowsAffected() == 0 {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			return
		}
	} else {
		res, err := tx.UpdatePlaylistUpdated(r.Context(), sqlc.UpdatePlaylistUpdatedParams{
			ID:    id,
			Owner: user,
		})
		if err != nil {
			log.Errorf("create playlist (update): %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		if res.RowsAffected() == 0 {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			return
		}
	}

	err = tx.ClearPlaylist(r.Context(), id)
	if err != nil {
		log.Errorf("create playlist: remove old tracks: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	if query.Has("songId") {
		tracks := make([]sqlc.AddPlaylistTracksParams, 0, len(query["songId"]))
		for i, songId := range query["songId"] {
			trackNr := int32(i)
			tracks = append(tracks, sqlc.AddPlaylistTracksParams{
				PlaylistID: id,
				SongID:     songId,
				Track:      trackNr,
			})
		}
		_, err = tx.AddPlaylistTracks(r.Context(), tracks)
		if err != nil {
			log.Errorf("create playlist: add new tracks: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	}

	err = tx.Commit(r.Context())
	if err != nil {
		log.Errorf("create playlist: commit: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	playlist, err := h.getPlaylistById(r.Context(), id, user)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("create playlist: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
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

	tx, err := h.Store.BeginTransaction(r.Context())
	if err != nil {
		log.Errorf("update playlist: begin transaction: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer tx.Rollback(r.Context())

	playlist, err := tx.FindPlaylist(r.Context(), sqlc.FindPlaylistParams{
		ID:   id,
		User: user,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("update playlist: find playlist: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}
	if playlist.Owner != user {
		responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		return
	}

	name := query.Get("name")
	if name == "" {
		name = playlist.Name
	}

	comment := playlist.Comment
	if query.Has("comment") {
		qComment := query.Get("comment")
		if qComment == "" {
			comment = nil
		} else {
			comment = &qComment
		}
	}

	err = tx.UpdatePlaylist(r.Context(), sqlc.UpdatePlaylistParams{
		ID:      playlist.ID,
		Owner:   user,
		Name:    name,
		Public:  false,
		Comment: comment,
	})
	if err != nil {
		log.Errorf("update playlist: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	trackCount := int32(playlist.TrackCount)
	if query.Has("songIndexToRemove") {
		tracks := make([]int32, 0, len(query["songIndexToRemove"]))
		for _, iStr := range query["songIndexToRemove"] {
			i, err := strconv.Atoi(iStr)
			if err != nil || i < 0 || i >= int(trackCount) {
				responses.EncodeError(w, query.Get("f"), fmt.Sprintf("invalid song index: %s", iStr), responses.SubsonicErrorGeneric)
				return
			}
			tracks = append(tracks, int32(i))
		}
		slices.Sort(tracks)
		lastTrack := int32(-1)
		for _, t := range tracks {
			if lastTrack == t {
				responses.EncodeError(w, query.Get("f"), fmt.Sprintf("duplicate song index: %d", t), responses.SubsonicErrorGeneric)
				return
			}
			lastTrack = t
		}
		err = tx.RemovePlaylistTracks(r.Context(), sqlc.RemovePlaylistTracksParams{
			PlaylistID: id,
			Tracks:     tracks,
		})
		if err != nil {
			log.Errorf("update playlist: remove tracks: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		for i, track := range tracks {
			next := trackCount
			if i+1 < len(tracks) {
				next = tracks[i+1]
			}
			err = tx.UpdatePlaylistTrackNumbers(r.Context(), sqlc.UpdatePlaylistTrackNumbersParams{
				PlaylistID: id,
				MinTrack:   track + 1,
				MaxTrack:   next - 1,
				Add:        -1 * int32(i+1),
			})
			if err != nil {
				log.Errorf("update playlist: fix track numbers after remove: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
				return
			}
		}
		trackCount -= int32(len(tracks))
	}

	if query.Has("songIdToAdd") {
		newEntries := make([]sqlc.AddPlaylistTracksParams, 0, len(query["songIdToAdd"]))
		for _, songID := range query["songIdToAdd"] {
			newEntries = append(newEntries, sqlc.AddPlaylistTracksParams{
				PlaylistID: id,
				SongID:     songID,
				Track:      trackCount,
			})
			trackCount++
		}
		_, err = tx.AddPlaylistTracks(r.Context(), newEntries)
		if err != nil {
			log.Errorf("update playlist: add tracks: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	}

	err = tx.Commit(r.Context())
	if err != nil {
		log.Errorf("update playlist: commit: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
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

	res, err := h.Store.DeletePlaylist(r.Context(), sqlc.DeletePlaylistParams{
		ID:    id,
		Owner: user,
	})
	if err != nil {
		log.Errorf("delete playlist: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	if res.RowsAffected() == 0 {
		responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
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
	dbPlaylist, err := h.Store.FindPlaylist(ctx, sqlc.FindPlaylistParams{
		ID:   id,
		User: user,
	})
	if err != nil {
		return nil, fmt.Errorf("get playlist by id: find playlist: %w", err)
	}
	tracks, err := h.Store.GetPlaylistTracks(ctx, sqlc.GetPlaylistTracksParams{
		PlaylistID: id,
		UserName:   user,
	})
	if err != nil {
		return nil, fmt.Errorf("get playlist by id: get tracks: %w", err)
	}

	songMap := make(map[string]*responses.Song, len(tracks))
	songList := make([]*responses.Song, 0, len(tracks))
	songIDs := mapData(tracks, func(s *sqlc.GetPlaylistTracksRow) string {
		var starred *time.Time
		if s.Starred.Valid {
			starred = &s.Starred.Time
		}
		var averageRating *float64
		if s.AvgRating != 0 {
			avgRating := math.Round(s.AvgRating*100) / 100
			averageRating = &avgRating
		}
		fallbackGain := float32(db.GetFallbackGain(ctx, h.Store))
		song := &responses.Song{
			ID:            s.ID,
			IsDir:         false,
			Title:         s.Title,
			Album:         s.AlbumName,
			Track:         int32PtrToIntPtr(s.Track),
			Year:          int32PtrToIntPtr(s.Year),
			CoverArt:      s.CoverID,
			Size:          s.Size,
			ContentType:   s.ContentType,
			Suffix:        filepath.Ext(s.Path),
			Duration:      int(s.DurationMs) / 1000,
			BitRate:       int(s.BitRate),
			SamplingRate:  int(s.SamplingRate),
			ChannelCount:  int(s.ChannelCount),
			UserRating:    int32PtrToIntPtr(s.UserRating),
			DiscNumber:    int32PtrToIntPtr(s.DiscNumber),
			Created:       s.Created.Time,
			AlbumID:       s.AlbumID,
			Type:          "music",
			MediaType:     "song",
			BPM:           int32PtrToIntPtr(s.Bpm),
			MusicBrainzID: s.MusicBrainzID,
			Starred:       starred,
			AverageRating: averageRating,
			ReplayGain: &responses.ReplayGain{
				TrackGain:    s.ReplayGain,
				AlbumGain:    s.AlbumReplayGain,
				TrackPeak:    s.ReplayGainPeak,
				AlbumPeak:    s.AlbumReplayGainPeak,
				FallbackGain: &fallbackGain,
			},
		}
		songMap[song.ID] = song
		songList = append(songList, song)
		return s.ID
	})
	dbGenres, err := h.Store.FindGenresBySongs(ctx, songIDs)
	if err != nil {
		return nil, fmt.Errorf("get playlist by id: get genres: %w", err)
	}
	for _, g := range dbGenres {
		song := songMap[g.SongID]
		if song.Genre == nil {
			song.Genre = &g.Name
		}
		song.Genres = append(song.Genres, &responses.GenreRef{
			Name: g.Name,
		})
	}
	songArtists, err := h.Store.FindArtistRefsBySongs(ctx, songIDs)
	if err != nil {
		return nil, fmt.Errorf("get playlist by id: find artist refs: %w", err)
	}
	for _, a := range songArtists {
		song := songMap[a.SongID]
		if song.ArtistID == nil && song.Artist == nil {
			song.ArtistID = &a.ID
			song.Artist = &a.Name
		}
		song.Artists = append(song.Artists, &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		})
	}
	albumArtists, err := h.Store.FindAlbumArtistRefsBySongs(ctx, songIDs)
	if err != nil {
		return nil, fmt.Errorf("get playlist by id: get album artist refs: %w", err)
	}
	for _, a := range albumArtists {
		song := songMap[a.SongID]
		if song.ArtistID == nil && song.Artist == nil {
			song.ArtistID = &a.ID
			song.Artist = &a.Name
		}
		song.AlbumArtists = append(song.AlbumArtists, &responses.ArtistRef{
			ID:   a.ID,
			Name: a.Name,
		})
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
		SongCount: int(dbPlaylist.TrackCount),
		Duration:  int(dbPlaylist.DurationMs / 1000),
		Created:   dbPlaylist.Created.Time,
		Changed:   dbPlaylist.Updated.Time,
		CoverArt:  cover,
		Entry:     &songList,
	}, nil
}
