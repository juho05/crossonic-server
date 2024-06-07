package handlers

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/jackc/pgx/v5"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
)

type ContextKey int

const (
	ContextKeyQuery ContextKey = iota
)

func ignoreExtension(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := path.Base(r.URL.Path)
		parts := strings.Split(name, ".")
		r.URL.Path = strings.TrimSuffix(r.URL.Path, name) + parts[0]
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) subsonicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				respondInternalErr(w, values.Get("f"), err)
				return
			}
			bodyValues, err := url.ParseQuery(string(body))
			if err != nil {
				responses.EncodeError(w, values.Get("f"), "Request body is not a valid query string", responses.SubsonicErrorGeneric)
				return
			}
			for k, v := range bodyValues {
				if values.Has(k) {
					values[k] = append(v, values[k]...)
				} else {
					values[k] = v
				}
			}
		}
		if !values.Has("u") {
			responses.EncodeError(w, values.Get("f"), "missing parameter 'u'", responses.SubsonicErrorRequiredParameterMissing)
			return
		}
		if !values.Has("v") {
			responses.EncodeError(w, values.Get("f"), "missing parameter 'v'", responses.SubsonicErrorRequiredParameterMissing)
			return
		}
		if !values.Has("c") {
			responses.EncodeError(w, values.Get("f"), "missing parameter 'c'", responses.SubsonicErrorRequiredParameterMissing)
			return
		}
		var authenticated bool
		var err error
		if values.Has("p") {
			authenticated, err = h.passwordAuth(r.Context(), values.Get("u"), values.Get("p"))
		} else if values.Has("t") {
			if !values.Has("s") {
				responses.EncodeError(w, values.Get("f"), "missing parameter 's'", responses.SubsonicErrorRequiredParameterMissing)
				return
			}
			authenticated, err = h.tokenAuth(r.Context(), values.Get("u"), values.Get("t"), values.Get("s"))
		} else {
			responses.EncodeError(w, values.Get("f"), "missing parameter 'p' or 't'", responses.SubsonicErrorRequiredParameterMissing)
			return
		}
		if err != nil {
			respondInternalErr(w, values.Get("f"), err)
			return
		}
		if !authenticated {
			responses.EncodeError(w, values.Get("f"), "invalid credentials", responses.SubsonicErrorWrongUsernameOrPassword)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ContextKeyQuery, values)))
	})
}

func (h *Handler) passwordAuth(ctx context.Context, username, password string) (bool, error) {
	user, err := h.Store.FindUser(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("password auth: %w", err)
	}
	dbPassword, err := db.DecryptPassword(user.EncryptedPassword)
	if err != nil {
		return false, fmt.Errorf("password auth: %w", err)
	}
	return subtle.ConstantTimeCompare([]byte(dbPassword), []byte(password)) == 1, nil
}

func (h *Handler) tokenAuth(ctx context.Context, username, token, salt string) (bool, error) {
	user, err := h.Store.FindUser(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("token auth: %w", err)
	}
	dbPassword, err := db.DecryptPassword(user.EncryptedPassword)
	if err != nil {
		return false, fmt.Errorf("token auth: %w", err)
	}
	hash := md5.Sum([]byte(dbPassword + salt))
	dbToken := hex.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(dbToken), []byte(token)) == 1, nil
}
