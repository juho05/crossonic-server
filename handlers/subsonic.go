package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/crossonic-server/handlers/responses"
)

func (h *Handler) registerSubsonicRoutes(r chi.Router) {
	r.Use(h.subsonicMiddleware)
	registerRoute(r, "/ping", h.handlePing)
}

// https://opensubsonic.netlify.app/docs/endpoints/ping/
func (h *Handler) handlePing(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	res := responses.New()
	res.EncodeOrLog(w, query.Get("f"))
}
