package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

func respondErr(w http.ResponseWriter, format string, err error) {
	if errors.Is(err, repos.ErrNotFound) {
		log.Error(err)
		responses.EncodeError(w, format, "not found", responses.SubsonicErrorNotFound)
		return
	}
	respondInternalErr(w, format, err)
}

func respondInternalErr(w http.ResponseWriter, format string, err error) {
	log.Error(err)
	responses.EncodeError(w, format, "internal server error", responses.SubsonicErrorGeneric)
}

func getQuery(r *http.Request) url.Values {
	query, ok := r.Context().Value(ContextKeyQuery).(url.Values)
	if !ok {
		panic("getQuery must be called after subsonicMiddleware")
	}
	return query
}

func user(r *http.Request) string {
	return getQuery(r).Get("u")
}

func format(r *http.Request) string {
	return getQuery(r).Get("f")
}

func paramIDReq(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	q := getQuery(r)
	id, ok := paramStrReq(w, r, name)
	if !ok {
		return "", false
	}
	if !crossonic.IDRegex.MatchString(id) {
		responses.EncodeError(w, q.Get("f"), fmt.Sprintf("invalid %s parameter", name), responses.SubsonicErrorGeneric)
		return "", false
	}
	return id, true
}

func paramLimitReq(w http.ResponseWriter, r *http.Request, name string, max *int, def int) (int, bool) {
	q := getQuery(r)
	limitStr := q.Get(name)
	limit := def
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 0 || (max != nil && limit > *max) {
			responses.EncodeError(w, q.Get("f"), fmt.Sprintf("invalid %s parameter", name), responses.SubsonicErrorGeneric)
			return 0, false
		}
	}
	return limit, true
}

func paramLimitOpt(w http.ResponseWriter, r *http.Request, name string, max *int) (*int, bool) {
	q := getQuery(r)
	limitStr := q.Get(name)
	var limit *int
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 0 || (max != nil && l > *max) {
			responses.EncodeError(w, q.Get("f"), fmt.Sprintf("invalid %s parameter", name), responses.SubsonicErrorGeneric)
			return nil, false
		}
		limit = &l
	}
	return limit, true
}

func paramOffset(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	q := getQuery(r)
	offsetStr := q.Get(name)
	var offset int
	var err error
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			responses.EncodeError(w, q.Get("f"), fmt.Sprintf("invalid %s parameter", name), responses.SubsonicErrorGeneric)
			return 0, false
		}
	}
	return offset, true
}

func paramBool(w http.ResponseWriter, r *http.Request, name string, def bool) (value bool, ok bool) {
	q := getQuery(r)
	boolStr := q.Get(name)
	if boolStr == "" {
		return def, true
	}
	value, err := strconv.ParseBool(boolStr)
	if err != nil {
		responses.EncodeError(w, q.Get("f"), fmt.Sprintf("invalid %s parameter", name), responses.SubsonicErrorGeneric)
		return false, false
	}
	return value, true
}

func paramTimeUnixMillis(w http.ResponseWriter, r *http.Request, name string, def time.Time) (time.Time, bool) {
	q := getQuery(r)
	timeStr := q.Get(name)
	if timeStr == "" {
		return def, true
	}
	millis, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		responses.EncodeError(w, q.Get("f"), fmt.Sprintf("invalid %s parameter", name), responses.SubsonicErrorGeneric)
		return time.Time{}, false
	}
	return time.UnixMilli(millis), true
}

func paramStrReq(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	q := getQuery(r)
	v := q.Get(name)
	if v == "" {
		responses.EncodeError(w, q.Get("f"), fmt.Sprintf("missing %s parameter", name), responses.SubsonicErrorRequiredParameterMissing)
		return "", false
	}
	return v, true
}

func registerRoute(r chi.Router, pattern string, handlerFunc func(w http.ResponseWriter, r *http.Request)) {
	r.Get(pattern, handlerFunc)
	r.Post(pattern, handlerFunc)
	r.Get(pattern+".view", handlerFunc)
	r.Post(pattern+".view", handlerFunc)
}
