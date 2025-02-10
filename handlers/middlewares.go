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
	"strings"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
)

type ContextKey int

const (
	ContextKeyQuery ContextKey = iota
)

func (h *Handler) subsonicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		if r.Method == http.MethodPost && r.Body != nil && r.ContentLength > 0 && strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
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
		if !values.Has("v") {
			responses.EncodeError(w, values.Get("f"), "missing parameter 'v'", responses.SubsonicErrorRequiredParameterMissing)
			return
		}
		if !values.Has("c") {
			responses.EncodeError(w, values.Get("f"), "missing parameter 'c'", responses.SubsonicErrorRequiredParameterMissing)
			return
		}

		// disable auth for getOpenSubsonicExtensions
		if r.URL.Path == "/rest/getOpenSubsonicExtensions" {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ContextKeyQuery, values)))
			return
		}

		if !values.Has("u") {
			responses.EncodeError(w, values.Get("f"), "missing parameter 'u'", responses.SubsonicErrorRequiredParameterMissing)
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
	user, err := h.DB.User().FindByName(ctx, username)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("password auth: %w", err)
	}
	dbPassword, err := repos.DecryptPassword(user.EncryptedPassword)
	if err != nil {
		return false, fmt.Errorf("password auth: %w", err)
	}
	return subtle.ConstantTimeCompare([]byte(dbPassword), []byte(password)) == 1, nil
}

func (h *Handler) tokenAuth(ctx context.Context, username, token, salt string) (bool, error) {
	user, err := h.DB.User().FindByName(ctx, username)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("token auth: %w", err)
	}
	dbPassword, err := repos.DecryptPassword(user.EncryptedPassword)
	if err != nil {
		return false, fmt.Errorf("token auth: %w", err)
	}
	hash := md5.Sum([]byte(dbPassword + salt))
	dbToken := hex.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(dbToken), []byte(token)) == 1, nil
}
