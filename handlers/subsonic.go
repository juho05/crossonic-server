package handlers

import (
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
)

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
	registerRoute(r, "/getAlbumList2", h.handleGetAlbumList2)
	registerRoute(r, "/getRandomSongs", h.handleGetRandomSongs)
	registerRoute(r, "/getAlbum", h.handleGetAlbum)
	registerRoute(r, "/getArtist", h.handleGetArtist)
	registerRoute(r, "/stream", h.handleStream)
	registerRoute(r, "/download", h.handleDownload)
	registerRoute(r, "/scrobble", h.handleScrobble)
	registerRoute(r, "/getNowPlaying", h.handleGetNowPlaying)
	registerRoute(r, "/search3", h.handleSearch3)
}

func int32PtrToIntPtr(ptr *int32) *int {
	if ptr == nil {
		return nil
	}
	v32 := *ptr
	v := int(v32)
	return &v
}

func intPtrToInt32Ptr(ptr *int) *int32 {
	if ptr == nil {
		return nil
	}
	v := *ptr
	v32 := int32(v)
	return &v32
}

func hasCoverArt(id string) bool {
	idType, ok := crossonic.GetIDType(id)
	if !ok {
		return false
	}
	var path string
	switch idType {
	case crossonic.IDTypeSong:
		path = filepath.Join(config.DataDir(), "covers", "songs")
	case crossonic.IDTypeAlbum:
		path = filepath.Join(config.DataDir(), "covers", "albums")
	case crossonic.IDTypeArtist:
		path = filepath.Join(config.DataDir(), "covers", "artists")
	default:
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
