package handlers

import (
	"github.com/go-chi/chi/v5"
)

var maxListSize = 500

func (h *Handler) registerSubsonicRoutes(r chi.Router) {
	r.Use(h.subsonicMiddleware)
	registerRoute(r, "/ping", h.handlePing)
	registerRoute(r, "/getLicense", h.handleGetLicense)
	registerRoute(r, "/getOpenSubsonicExtensions", h.handleGetOpenSubsonicExtensions)
	registerRoute(r, "/startScan", h.handleStartScan)
	registerRoute(r, "/getScanStatus", h.handleGetScanStatus)
	registerRoute(r, "/setRating", h.handleSetRating)
	registerRoute(r, "/star", h.handleStar)
	registerRoute(r, "/unstar", h.handleUnstar)
	registerRoute(r, "/getCoverArt", h.handleGetCoverArt)
	registerRoute(r, "/getGenres", h.handleGetGenres)
	registerRoute(r, "/getArtists", h.handleGetArtists)
	registerRoute(r, "/getAlbumList", h.handleGetAlbumList(1))
	registerRoute(r, "/getAlbumList2", h.handleGetAlbumList(2))
	registerRoute(r, "/getRandomSongs", h.handleGetRandomSongs)
	registerRoute(r, "/getAlbum", h.handleGetAlbum)
	registerRoute(r, "/getArtist", h.handleGetArtist)
	registerRoute(r, "/stream", h.handleStream)
	registerRoute(r, "/download", h.handleDownload)
	registerRoute(r, "/scrobble", h.handleScrobble)
	registerRoute(r, "/getNowPlaying", h.handleGetNowPlaying)
	registerRoute(r, "/search3", h.handleSearch3)
	registerRoute(r, "/getLyricsBySongId", h.handleGetLyricsBySongId)
	registerRoute(r, "/getPlaylists", h.handleGetPlaylists)
	registerRoute(r, "/getPlaylist", h.handleGetPlaylist)
	registerRoute(r, "/createPlaylist", h.handleCreatePlaylist)
	registerRoute(r, "/updatePlaylist", h.handleUpdatePlaylist)
	registerRoute(r, "/deletePlaylist", h.handleDeletePlaylist)
	registerRoute(r, "/getSong", h.handleGetSong)
	registerRoute(r, "/getStarred", h.handleGetStarred(1))
	registerRoute(r, "/getStarred2", h.handleGetStarred(2))
	registerRoute(r, "/getSongsByGenre", h.handleGetSongsByGenre)
	registerRoute(r, "/getAlbumInfo", h.handleGetAlbumInfo2)
	registerRoute(r, "/getAlbumInfo2", h.handleGetAlbumInfo2)
	registerRoute(r, "/getArtistInfo", h.handleGetArtistInfo(1))
	registerRoute(r, "/getArtistInfo2", h.handleGetArtistInfo(2))
}
