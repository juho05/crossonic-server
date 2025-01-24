package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

func (h *Handler) handleSearch3(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	query.Set("query", strings.Trim(query.Get("query"), `"`))

	artists, ok := h.searchArtists(r.Context(), w, query)
	if !ok {
		return
	}

	albums, ok := h.searchAlbums(r.Context(), w, query)
	if !ok {
		return
	}

	songs, ok := h.searchSongs(r.Context(), w, query)
	if !ok {
		return
	}

	res := responses.New()
	res.SearchResult3 = &responses.SearchResult3{
		Songs:   songs,
		Albums:  albums,
		Artists: artists,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) searchArtists(ctx context.Context, w http.ResponseWriter, query url.Values) ([]*responses.Artist, bool) {
	user := query.Get("u")
	limitStr := query.Get("artistCount")
	limit := 20
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid artistCount value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	offsetStr := query.Get("artistOffset")
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid artistOffset value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	artists, err := h.DB.Artist().FindBySearch(ctx, query.Get("query"), true, offset, limit, repos.IncludeArtistInfoFull(user))
	if err != nil {
		log.Errorf("search3: artists: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
	}
	return mapList(artists, func(a *repos.CompleteArtist) *responses.Artist {
		var coverArt *string
		if hasCoverArt(a.ID) {
			coverArt = &a.ID
		}
		return &responses.Artist{
			ID:            a.ID,
			Name:          a.Name,
			CoverArt:      coverArt,
			AlbumCount:    &a.AlbumCount,
			Starred:       a.Starred,
			MusicBrainzID: a.MusicBrainzID,
			UserRating:    a.UserRating,
			AverageRating: a.AverageRating,
		}
	}), true
}

func (h *Handler) searchAlbums(ctx context.Context, w http.ResponseWriter, query url.Values) ([]*responses.Album, bool) {
	user := query.Get("u")
	limitStr := query.Get("albumCount")
	limit := 20
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid albumCount value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	offsetStr := query.Get("albumOffset")
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid albumOffset value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	dbAlbums, err := h.DB.Album().FindBySearchQuery(ctx, query.Get("query"), offset, limit, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		log.Errorf("search3: albums: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return nil, false
	}
	albums := mapList(dbAlbums, func(a *repos.CompleteAlbum) *responses.Album {
		return &responses.Album{
			ID:            a.ID,
			Created:       a.Created,
			Title:         a.Name,
			Name:          a.Name,
			SongCount:     int(a.TrackCount),
			Duration:      int(a.Duration.ToStd().Seconds()),
			Year:          a.Year,
			Starred:       a.Starred,
			UserRating:    a.UserRating,
			AverageRating: a.AverageRating,
			MusicBrainzID: a.MusicBrainzID,
			IsCompilation: a.IsCompilation,
			ReleaseTypes:  a.ReleaseTypes,
			RecordLabels: mapList(a.RecordLabels, func(label string) *responses.RecordLabel {
				return &responses.RecordLabel{
					Name: label,
				}
			}),
			Artists: mapList(a.Artists, func(a repos.ArtistRef) *responses.ArtistRef {
				return &responses.ArtistRef{
					ID:   a.ID,
					Name: a.Name,
				}
			}),
			Genres: mapList(a.Genres, func(g string) *responses.GenreRef {
				return &responses.GenreRef{
					Name: g,
				}
			}),
		}
	})

	err = h.completeAlbumInfo(albums)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("search3: search albums: %w", err))
		return nil, false
	}

	return albums, true
}

func (h *Handler) searchSongs(ctx context.Context, w http.ResponseWriter, query url.Values) ([]*responses.Song, bool) {
	user := query.Get("u")
	limitStr := query.Get("songCount")
	limit := 20
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid songCount value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	offsetStr := query.Get("songOffset")
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid songOffset value", responses.SubsonicErrorGeneric)
			return nil, false
		}
	}

	dbSongs, err := h.DB.Song().FindBySearchQuery(ctx, repos.SongFindBySearchParams{
		Query:  query.Get("query"),
		Offset: offset,
		Limit:  limit,
	}, repos.IncludeSongInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("search3: songs: %w", err))
		return nil, false
	}
	songs := mapList(dbSongs, func(s *repos.CompleteSong) *responses.Song {
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
			ReplayGain: &responses.ReplayGain{
				TrackGain: s.ReplayGain,
				AlbumGain: s.AlbumReplayGain,
				TrackPeak: s.ReplayGainPeak,
				AlbumPeak: s.AlbumReplayGainPeak,
			},
		}
	})
	err = h.completeSongInfo(ctx, songs)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("search3: songs: %w", err))
		return nil, false
	}
	return songs, true
}
