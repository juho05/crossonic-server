package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/lastfm"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

const keepLastFMInfoFor = 30 * 24 * time.Hour

func (h *Handler) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}
	dbArtist, err := h.DB.Artist().FindByID(r.Context(), id, repos.IncludeArtistInfoFull(user))
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("get artist: %w", err))
		}
		return
	}
	dbAlbums, err := h.DB.Artist().GetAlbums(r.Context(), dbArtist.ID, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get artist: get albums: %w", err))
		return
	}

	albums := responses.NewAlbums(dbAlbums, h.Config)

	artist := responses.NewArtist(dbArtist, h.Config)
	artist.Albums = albums

	res := responses.New()
	res.Artist = artist
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetAlbum(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")
	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}
	dbAlbum, err := h.DB.Album().FindByID(r.Context(), id, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("get album: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}

	dbSongs, err := h.DB.Album().GetTracks(r.Context(), id, repos.IncludeSongInfoFull(user))
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get album: get songs: %w", err))
		return
	}

	songs := responses.NewSongs(dbSongs, h.Config)
	album := responses.NewAlbum(dbAlbum, h.Config)

	res := responses.New()
	res.Album = &responses.AlbumWithSongs{
		Album: album,
		Songs: songs,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetSong(w http.ResponseWriter, r *http.Request) {
	f := format(r)

	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}

	song, err := h.DB.Song().FindByID(r.Context(), id, repos.IncludeSongInfoFull(user(r)))
	if err != nil {
		respondErr(w, f, fmt.Errorf("get song: find song by id: %w", err))
		return
	}

	res := responses.New()
	res.Song = responses.NewSong(song, h.Config)
	res.EncodeOrLog(w, f)
}

func (h *Handler) handleGetGenres(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)

	dbGenres, err := h.DB.Genre().FindAllWithCounts(r.Context())
	if err != nil {
		respondInternalErr(w, query.Get("f"), fmt.Errorf("get genres: %w", err))
		return
	}
	genres := make([]*responses.Genre, 0, len(dbGenres))
	for _, g := range dbGenres {
		genres = append(genres, &responses.Genre{
			SongCount:  g.SongCount,
			AlbumCount: g.AlbumCount,
			Value:      g.Name,
		})
	}

	res := responses.New()
	res.Genres = &responses.Genres{
		Genres: genres,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

var ignoredArticles = []string{"The", "An", "A", "Der", "Die", "Das", "Ein", "Eine", "Les", "Le", "La", "L'"}

func (h *Handler) handleGetArtistsIndex(byID3 bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := getQuery(r)

		var ifModifiedSince time.Time
		if !byID3 {
			t, ok := paramTimeUnixMillis(w, r, "ifModifiedSince", time.Time{})
			if !ok {
				return
			}
			ifModifiedSince = t
		}

		artists, err := h.DB.Artist().FindAll(r.Context(), repos.FindArtistsParams{
			OnlyAlbumArtists: true,
			UpdatedAfter:     ifModifiedSince,
		}, repos.IncludeArtistInfoFull(query.Get("u")))
		if err != nil {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("get artists: %w", err))
			return
		}
		indexMap := make(map[rune]*responses.Index, 27)
		for _, a := range artists {
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

			artist := responses.NewArtist(a, h.Config)

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

		lastModified, err := h.DB.System().LastScan(r.Context())
		if err != nil {
			respondInternalErr(w, query.Get("f"), fmt.Errorf("get artists index: last scan: %w", err))
			return
		}

		res := responses.New()
		index := &responses.ArtistIndexes{
			IgnoredArticles: strings.Join(ignoredArticles, " "),
			LastModified:    lastModified.UnixMilli(),
			Index:           indexList,
		}
		if byID3 {
			res.Artists = index
		} else {
			res.Indexes = index
		}
		res.EncodeOrLog(w, query.Get("f"))
	}
}

func (h *Handler) handleGetAlbumInfo2(w http.ResponseWriter, r *http.Request) {
	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}

	if typ, _ := crossonic.GetIDType(id); typ == crossonic.IDTypeSong {
		song, err := h.DB.Song().FindByID(r.Context(), id, repos.IncludeSongInfoBare())
		if err != nil {
			respondErr(w, format(r), fmt.Errorf("get album info: find song: %w", err))
			return
		}
		if song.AlbumID == nil {
			responses.EncodeError(w, format(r), "song has no album", responses.SubsonicErrorNotFound)
			return
		}
		id = *song.AlbumID
	}

	info, err := h.DB.Album().GetInfo(r.Context(), id)
	if err != nil && !errors.Is(err, repos.ErrNotFound) {
		respondErr(w, format(r), fmt.Errorf("get album info: get info: %w", err))
		return
	}
	if info.Updated != nil && time.Since(*info.Updated) > keepLastFMInfoFor {
		info.Updated = nil
	}

	if info.Updated == nil && h.LastFM != nil {
		album, err := h.DB.Album().FindByID(r.Context(), id, repos.IncludeAlbumInfo{
			Artists: true,
		})
		if err != nil {
			respondErr(w, format(r), fmt.Errorf("get album info: find album: %w", err))
			return
		}
		if len(album.Artists) > 0 {
			lInfo, err := h.LastFM.GetAlbumInfo(r.Context(), album.Name, album.Artists[0].Name, album.MusicBrainzID)
			if err != nil && !errors.Is(err, lastfm.ErrNotFound) {
				respondErr(w, format(r), fmt.Errorf("get album info: fetch last.fm data: %w", err))
				return
			}
			info.Description = lInfo.Wiki.Content
			info.LastFMMBID = lInfo.MBID
			info.LastFMURL = lInfo.URL

			if err == nil {
				err = h.DB.Album().SetInfo(r.Context(), id, repos.SetAlbumInfo{
					Description: info.Description,
					LastFMURL:   info.LastFMURL,
					LastFMMBID:  info.LastFMMBID,
				})
				if err != nil {
					respondErr(w, format(r), fmt.Errorf("get album info: save new last.fm data in DB: %w", err))
					return
				}
			}
		}
	}

	mbid := info.MusicBrainzID
	if mbid == nil {
		mbid = info.LastFMMBID
	}

	var smallImageUrl *string
	var mediumImageUrl *string
	var largeImageUrl *string
	if responses.HasCoverArt(id, h.Config) {
		u := h.constructCoverURL(id, getQuery(r))
		sm := fmt.Sprintf("%s&size=64", u)
		md := fmt.Sprintf("%s&size=256", u)
		lg := fmt.Sprintf("%s&size=512", u)
		smallImageUrl = &sm
		mediumImageUrl = &md
		largeImageUrl = &lg
	}

	res := responses.New()
	res.AlbumInfo = &responses.AlbumInfo{
		Notes:          info.Description,
		MusicBrainzID:  mbid,
		ReleaseMBID:    info.ReleaseMBID,
		LastFMUrl:      info.LastFMURL,
		SmallImageURL:  smallImageUrl,
		MediumImageURL: mediumImageUrl,
		LargeImageURL:  largeImageUrl,
	}
	res.EncodeOrLog(w, format(r))
}

func (h *Handler) handleGetArtistInfo(version int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := paramIDReq(w, r, "id")
		if !ok {
			return
		}

		if typ, _ := crossonic.GetIDType(id); typ == crossonic.IDTypeSong {
			song, err := h.DB.Song().FindByID(r.Context(), id, repos.IncludeSongInfo{
				Lists: true,
			})
			if err != nil {
				respondErr(w, format(r), fmt.Errorf("get artist info: find song: %w", err))
				return
			}
			if len(song.AlbumArtists) > 0 {
				id = song.AlbumArtists[0].ID
			} else if len(song.Artists) > 0 {
				id = song.Artists[0].ID
			} else {
				responses.EncodeError(w, format(r), "song has no artists", responses.SubsonicErrorNotFound)
				return
			}
		} else if typ == crossonic.IDTypeAlbum {
			album, err := h.DB.Album().FindByID(r.Context(), id, repos.IncludeAlbumInfo{
				Artists: true,
			})
			if err != nil {
				respondErr(w, format(r), fmt.Errorf("get artist info: find album: %w", err))
				return
			}
			if len(album.Artists) > 0 {
				id = album.Artists[0].ID
			} else {
				responses.EncodeError(w, format(r), "album has no artists", responses.SubsonicErrorNotFound)
				return
			}
		}

		info, err := h.DB.Artist().GetInfo(r.Context(), id)
		if err != nil {
			respondErr(w, format(r), fmt.Errorf("get artist info: get info: %w", err))
			return
		}
		if info.Updated != nil && time.Since(*info.Updated) > keepLastFMInfoFor {
			info.Updated = nil
		}

		if info.Updated == nil && h.LastFM != nil {
			artist, err := h.DB.Artist().FindByID(r.Context(), id, repos.IncludeArtistInfoBare())
			if err != nil {
				respondErr(w, format(r), fmt.Errorf("get album info: find album: %w", err))
				return
			}
			lInfo, err := h.LastFM.GetArtistInfo(r.Context(), artist.Name, artist.MusicBrainzID)
			if err != nil && !errors.Is(err, lastfm.ErrNotFound) {
				respondErr(w, format(r), fmt.Errorf("get album info: fetch last.fm data: %w", err))
				return
			}
			info.Biography = lInfo.Bio.Content
			info.LastFMMBID = lInfo.MBID
			info.LastFMURL = lInfo.URL

			if err == nil {
				err = h.DB.Artist().SetInfo(r.Context(), id, repos.SetArtistInfo{
					Biography:  info.Biography,
					LastFMURL:  info.LastFMURL,
					LastFMMBID: info.LastFMMBID,
				})
				if err != nil {
					respondErr(w, format(r), fmt.Errorf("get album info: save new last.fm data in DB: %w", err))
					return
				}
			}
		}

		mbid := info.MusicBrainzID
		if mbid == nil {
			mbid = info.LastFMMBID
		}

		var smallImageUrl *string
		var mediumImageUrl *string
		var largeImageUrl *string
		if responses.HasCoverArt(id, h.Config) {
			u := h.constructCoverURL(id, getQuery(r))
			sm := fmt.Sprintf("%s&size=64", u)
			md := fmt.Sprintf("%s&size=256", u)
			lg := fmt.Sprintf("%s&size=512", u)
			smallImageUrl = &sm
			mediumImageUrl = &md
			largeImageUrl = &lg
		}

		res := responses.New()
		artistInfo := &responses.ArtistInfo{
			Biography:      info.Biography,
			MusicBrainzID:  mbid,
			LastFMUrl:      info.LastFMURL,
			SmallImageURL:  smallImageUrl,
			MediumImageURL: mediumImageUrl,
			LargeImageURL:  largeImageUrl,
		}
		if version == 2 {
			res.ArtistInfo2 = artistInfo
		} else {
			res.ArtistInfo = artistInfo
		}
		res.EncodeOrLog(w, format(r))
	}
}

func (h *Handler) handleGetMusicDirectory(w http.ResponseWriter, r *http.Request) {
	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}

	if crossonic.IsIDType(id, crossonic.IDTypeAlbum) {
		album, err := h.DB.Album().FindByID(r.Context(), id, repos.IncludeAlbumInfo{
			User:        user(r),
			Annotations: true,
			PlayInfo:    true,
			Artists:     true,
		})
		if err != nil {
			respondErr(w, format(r), fmt.Errorf("get music directory: get album tracks: %w", err))
			return
		}
		songs, err := h.DB.Album().GetTracks(r.Context(), id, repos.IncludeSongInfoFull(user(r)))
		if err != nil {
			respondErr(w, format(r), fmt.Errorf("get music directory: get album tracks: %w", err))
			return
		}

		var parent *string
		if len(album.Artists) > 0 {
			parent = &album.Artists[0].ID
		}

		res := responses.New()
		res.Directory = &responses.Directory{
			ID:            album.ID,
			Name:          album.Name,
			Parent:        parent,
			Starred:       album.Starred,
			UserRating:    album.UserRating,
			AverageRating: album.AverageRating,
			PlayCount:     &album.PlayCount,
			Child: util.Map(songs, func(s *repos.CompleteSong) any {
				return responses.NewSong(s, h.Config)
			}),
		}
		res.EncodeOrLog(w, format(r))
		return
	} else if crossonic.IsIDType(id, crossonic.IDTypeArtist) {
		artist, err := h.DB.Artist().FindByID(r.Context(), id, repos.IncludeArtistInfo{
			User:        user(r),
			Annotations: true,
		})
		if err != nil {
			respondErr(w, format(r), fmt.Errorf("get music directory: get artist albums: %w", err))
			return
		}
		albums, err := h.DB.Artist().GetAlbums(r.Context(), id, repos.IncludeAlbumInfoFull(user(r)))
		if err != nil {
			respondErr(w, format(r), fmt.Errorf("get music directory: get artist albums: %w", err))
			return
		}

		res := responses.New()
		res.Directory = &responses.Directory{
			ID:            artist.ID,
			Name:          artist.Name,
			Starred:       artist.Starred,
			UserRating:    artist.UserRating,
			AverageRating: artist.AverageRating,
			Child: util.Map(albums, func(a *repos.CompleteAlbum) any {
				return responses.NewAlbum(a, h.Config)
			}),
		}
		res.EncodeOrLog(w, format(r))
		return
	} else {
		responses.EncodeError(w, format(r), "invalid id type", responses.SubsonicErrorNotFound)
	}
}

func (h *Handler) constructCoverURL(id string, query url.Values) string {
	u := fmt.Sprintf("%s/rest/getCoverArt?id=%s&c=%s&u=%s&v=%s", h.Config.BaseURL, id, query.Get("c"), query.Get("u"), query.Get("v"))
	if query.Has("p") {
		u += "&p=" + query.Get("p")
	}
	if query.Has("t") {
		u += "&t=" + query.Get("t")
	}
	if query.Has("s") {
		u += "&s=" + query.Get("s")
	}
	if query.Has("apiKey") {
		u += "&apiKey=" + query.Get("apiKey")
	}
	return u
}
