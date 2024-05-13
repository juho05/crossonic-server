package handlers

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func respondInternalErr(w http.ResponseWriter, format string, err error) {
	log.Error(err)
	responses.EncodeError(w, format, http.StatusText(http.StatusInternalServerError), responses.SubsonicErrorGeneric)
}

func getQuery(r *http.Request) url.Values {
	query, ok := r.Context().Value(ContextKeyQuery).(url.Values)
	if !ok {
		panic("getQuery must be called after subsonicMiddleware")
	}
	return query
}

func registerRoute(r chi.Router, pattern string, handlerFunc func(w http.ResponseWriter, r *http.Request)) {
	r.Get(pattern, handlerFunc)
	r.Post(pattern, handlerFunc)
	r.Get(pattern+".view", handlerFunc)
	r.Post(pattern+".view", handlerFunc)
}
