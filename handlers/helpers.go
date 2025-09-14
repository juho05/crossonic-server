package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

func respondGenericErr(w http.ResponseWriter, format string, message string) {
	responses.EncodeError(w, format, message, responses.SubsonicErrorGeneric)
}

func respondNotFoundErr(w http.ResponseWriter, format, message string) {
	if message == "" {
		message = "not found"
	}
	responses.EncodeError(w, format, message, responses.SubsonicErrorNotFound)
}

func respondErr(w http.ResponseWriter, format string, err error) {
	if errors.Is(err, repos.ErrNotFound) {
		log.Error(err)
		respondNotFoundErr(w, format, "")
		return
	}
	respondInternalErr(w, format, err)
}

func respondInternalErr(w http.ResponseWriter, format string, err error) {
	log.Error(err)
	responses.EncodeError(w, format, "internal server error", responses.SubsonicErrorGeneric)
}

func registerRoute(r chi.Router, pattern string, handlerFunc func(w http.ResponseWriter, r *http.Request)) {
	r.Get(pattern, handlerFunc)
	r.Post(pattern, handlerFunc)
	r.Get(pattern+".view", handlerFunc)
	r.Post(pattern+".view", handlerFunc)
}
