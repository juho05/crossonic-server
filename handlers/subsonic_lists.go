package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
)

func (h *Handler) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	limitStr := query.Get("size")
	limit := 10
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 || limit > 500 {
			responses.EncodeError(w, query.Get("f"), "invalid size value", responses.SubsonicErrorGeneric)
			return
		}
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

	genres := mapList(query["genre"], func(g string) string {
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
	err = h.completeSongInfo(r.Context(), songs)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get random songs: %w", err))
		return
	}

	res := responses.New()
	res.RandomSongs = &responses.RandomSongs{
		Songs: songs,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetAlbumList2(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	listType := query.Get("type")

	limitStr := query.Get("size")
	limit := 10
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 || limit > 500 {
			responses.EncodeError(w, query.Get("f"), "invalid size value", responses.SubsonicErrorGeneric)
			return
		}
	}

	offsetStr := query.Get("offset")
	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid offset value", responses.SubsonicErrorGeneric)
			return
		}
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

	genres := query["genre"]
	if listType == "byGenre" && len(genres) == 0 {
		responses.EncodeError(w, query.Get("f"), "missing genre parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	var albums []*responses.Album
	sortTypes := map[string]repos.FindAlbumSortBy{
		"random":             repos.FindAlbumSortRandom,
		"newest":             repos.FindAlbumSortByCreated,
		"highest":            repos.FindAlbumSortByRating,
		"alphabeticalByName": repos.FindAlbumSortByName,
		"starred":            repos.FindAlbumSortByStarred,
		"byYear":             repos.FindAlbumSortByYear,
		"byGenre":            repos.FindAlbumSortByName,
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
		Offset:   offset,
		Limit:    limit,
	}, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album list 2: find all: %w", err))
		return
	}

	albums = mapList(a, func(album *repos.CompleteAlbum) *responses.Album {
		return &responses.Album{
			ID:            album.ID,
			Created:       album.Created,
			Title:         album.Name,
			Name:          album.Name,
			SongCount:     album.TrackCount,
			Duration:      int(album.Duration.ToStd().Seconds()),
			Year:          album.Year,
			Starred:       album.Starred,
			UserRating:    album.UserRating,
			AverageRating: album.AverageRating,
			MusicBrainzID: album.MusicBrainzID,
			IsCompilation: album.IsCompilation,
			ReleaseTypes:  album.ReleaseTypes,
			RecordLabels: mapList(album.RecordLabels, func(label string) *responses.RecordLabel {
				return &responses.RecordLabel{
					Name: label,
				}
			}),
			Genres: mapList(album.Genres, func(g string) *responses.GenreRef {
				return &responses.GenreRef{
					Name: g,
				}
			}),
			Artists: mapList(album.Artists, func(a repos.ArtistRef) *responses.ArtistRef {
				return &responses.ArtistRef{
					ID:   a.ID,
					Name: a.Name,
				}
			}),
		}
	})

	err = h.completeAlbumInfo(albums)
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album list2: %w", err))
		return
	}

	res := responses.New()
	res.AlbumList2 = &responses.AlbumList2{
		Albums: albums,
	}
	res.EncodeOrLog(w, query.Get("f"))
}
