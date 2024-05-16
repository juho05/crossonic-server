package handlers

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) registerSubsonicRoutes(r chi.Router) {
	r.Use(h.subsonicMiddleware)
	registerRoute(r, "/ping", h.handlePing)
	registerRoute(r, "/getLicense", h.handleGetLicense)
	registerRoute(r, "/getOpenSubsonicExtensions", h.handleGetOpenSubsonicExtensions)
	registerRoute(r, "/startScan", h.handleStartScan)
	registerRoute(r, "/getScanStatus", h.handleGetScanStatus)
}
