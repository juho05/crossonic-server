package handlers

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

const (
	authRateWindow   = 5 * time.Second
	authRateMaxFails = 3
)

type ContextKey int

const (
	ContextKeyQuery ContextKey = iota
)

var errAuthTypeNotSupported = errors.New("auth type not supported")

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
		if r.URL.Path == "/rest/getOpenSubsonicExtensions" || r.URL.Path == "/rest/getOpenSubsonicExtensions.view" {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ContextKeyQuery, values)))
			return
		}

		if values.Has("apiKey") {
			apiKey := values.Get("apiKey")

			if values.Has("u") || values.Has("p") || values.Has("t") || values.Has("s") {
				responses.EncodeError(w, values.Get("f"), "multiple conflicting authentication mechanisms provided", responses.SubsonicErrorMultipleConflictingAuthenticationMechanismsProvided)
				return
			}

			// There is no rate limiting on apiKey because the search space is big enough that brute-force is infeasible.

			user, err := h.DB.User().FindUserNameByAPIKey(r.Context(), apiKey)
			if err != nil {
				if errors.Is(err, repos.ErrNotFound) {
					responses.EncodeError(w, values.Get("f"), "invalid api key", responses.SubsonicErrorInvalidAPIKey)
				} else {
					respondInternalErr(w, values.Get("f"), err)
				}
				return
			}

			values.Set("u", user)
		} else {
			var authenticated bool
			if !values.Has("u") {
				responses.EncodeError(w, values.Get("f"), "missing parameter 'u'", responses.SubsonicErrorRequiredParameterMissing)
				return
			}
			username := values.Get("u")
			if limited, retryAfter := h.isAuthRateLimited(username); limited {
				seconds := int(math.Ceil(retryAfter.Seconds()))
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				w.WriteHeader(http.StatusTooManyRequests)
				responses.EncodeError(w, values.Get("f"), "too many failed login attempts", responses.SubsonicErrorGeneric)
				return
			}
			var err error
			if values.Has("p") {
				if values.Has("t") || values.Has("s") {
					responses.EncodeError(w, values.Get("f"), "multiple conflicting authentication mechanisms provided", responses.SubsonicErrorMultipleConflictingAuthenticationMechanismsProvided)
					return
				}
				authenticated, err = h.passwordAuth(r.Context(), username, values.Get("p"))
			} else if values.Has("t") {
				// This allows empty salt values if the parameter is specified but empty. It's the clients responsibility
				// to properly use this parameter. We do not validate this value any further to prevent client incompatibilities.
				if !values.Has("s") {
					responses.EncodeError(w, values.Get("f"), "missing parameter 's'", responses.SubsonicErrorRequiredParameterMissing)
					return
				}
				authenticated, err = h.tokenAuth(r.Context(), username, values.Get("t"), values.Get("s"))
			} else {
				responses.EncodeError(w, values.Get("f"), "missing authentication parameter(s)", responses.SubsonicErrorRequiredParameterMissing)
				return
			}
			if err != nil {
				if errors.Is(err, errAuthTypeNotSupported) {
					if values.Has("t") {
						responses.EncodeError(w, values.Get("f"), "token authentication not supported", responses.SubsonicErrorTokenAuthenticationNotSupported)
					} else {
						responses.EncodeError(w, values.Get("f"), "provided authentication mechanism not supported", responses.SubsonicErrorProvidedAuthenticationMechanismNotSupported)
					}
					return
				}
				respondInternalErr(w, values.Get("f"), err)
				return
			}
			if !authenticated {
				h.recordAuthFailure(username)
				responses.EncodeError(w, values.Get("f"), "invalid credentials", responses.SubsonicErrorWrongUsernameOrPassword)
				return
			}
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ContextKeyQuery, values)))
	})
}

func (h *Handler) passwordAuth(ctx context.Context, username, password string) (bool, error) {
	if strings.HasPrefix(password, "enc:") {
		decoded, err := hex.DecodeString(strings.TrimPrefix(password, "enc:"))
		if err != nil {
			return false, fmt.Errorf("failed to decode hex encoded password: %w", err)
		}
		password = string(decoded)
	}

	user, err := h.DB.User().FindByName(ctx, username)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			// dummy computation to prevent username enumerations
			_, _ = repos.VerifyPassword("$argon2id$v=19$m=3072,t=1,p=4$N2LCBe1+JO5/S5bE2HGRcg$MeQXxb6r9d0GhBbwMvc7MmkJyrbv8+EkY1Aj6SFpIeE", password)
			return false, nil
		}
		return false, fmt.Errorf("password auth: %w", err)
	}

	if user.HashedPassword != nil {
		return repos.VerifyPassword(*user.HashedPassword, password)
	}

	if user.EncryptedPassword == nil {
		return false, errAuthTypeNotSupported
	}

	dbPassword, err := repos.DecryptPassword(user.EncryptedPassword, h.Config.EncryptionKey)
	if err != nil {
		return false, fmt.Errorf("password auth: %w", err)
	}

	if subtle.ConstantTimeCompare([]byte(dbPassword), []byte(password)) == 0 {
		return false, nil
	}

	setHashedPasswordIfNil(ctx, h.DB, user, password)

	return true, nil
}

func (h *Handler) tokenAuth(ctx context.Context, username, token, salt string) (bool, error) {
	user, err := h.DB.User().FindByName(ctx, username)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("token auth: %w", err)
	}

	if user.EncryptedPassword == nil {
		return false, errAuthTypeNotSupported
	}

	dbPassword, err := repos.DecryptPassword(user.EncryptedPassword, h.Config.EncryptionKey)
	if err != nil {
		return false, fmt.Errorf("token auth: %w", err)
	}
	hash := md5.Sum([]byte(dbPassword + salt))
	dbToken := hex.EncodeToString(hash[:])

	if subtle.ConstantTimeCompare([]byte(dbToken), []byte(token)) == 0 {
		return false, nil
	}

	setHashedPasswordIfNil(ctx, h.DB, user, dbPassword)

	return true, nil
}

func (h *Handler) isAuthRateLimited(username string) (bool, time.Duration) {
	cutoff := time.Now().Add(-authRateWindow)
	h.authFailuresLock.RLock()
	defer h.authFailuresLock.RUnlock()
	failures := h.authFailures[username]
	// Failures are appended in chronological order, so those within the window
	// stay sorted ascending (oldest first).
	inWindow := make([]time.Time, 0, len(failures))
	for _, t := range failures {
		if t.After(cutoff) {
			inWindow = append(inWindow, t)
		}
	}
	if len(inWindow) < authRateMaxFails {
		return false, 0
	}
	expiresAt := inWindow[0].Add(authRateWindow)
	return true, time.Until(expiresAt)
}

func (h *Handler) recordAuthFailure(username string) {
	now := time.Now()
	cutoff := now.Add(-authRateWindow)
	h.authFailuresLock.Lock()
	defer h.authFailuresLock.Unlock()
	prev := h.authFailures[username]
	pruned := prev[:0]
	for _, t := range prev {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	h.authFailures[username] = append(pruned, now)
}

func setHashedPasswordIfNil(ctx context.Context, db repos.DB, user *repos.User, password string) {
	if user.HashedPassword == nil {
		err := db.User().Update(ctx, user.Name, repos.UpdateUserParams{
			Password: repos.NewOptionalFull(password),
		})
		if err != nil {
			log.Errorf("failed to set user hashed password: %v", err)
		}
	}
}
