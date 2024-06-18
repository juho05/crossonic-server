package handlers

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) registerCrossonicRoutes(r chi.Router) {
	r.Use(h.subsonicMiddleware)
	registerRoute(r, "/connectListenBrainz", h.handleConnectListenbrainz)
	registerRoute(r, "/getListenBrainzConfig", h.handleGetListenbrainzConfig)
	registerRoute(r, "/connect", h.handleConnect)
	registerRoute(r, "/setPlaylistCover", h.handleSetPlaylistCover)
}
