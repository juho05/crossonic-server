package handlers

import "github.com/go-chi/chi/v5"

func (h *Handler) registerCrossonicRoutes(r chi.Router) {
	r.Use(h.subsonicMiddleware)
}
