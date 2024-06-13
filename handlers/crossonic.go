package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/log"
)

func (h *Handler) registerCrossonicRoutes(r chi.Router) {
	r.Use(h.subsonicMiddleware)
	registerRoute(r, "/connectListenBrainz", h.handleConnectListenbrainz)
	registerRoute(r, "/getListenBrainzConfig", h.handleGetListenbrainzConfig)
	registerRoute(r, "/connect", h.handleConnect)
}

func (h *Handler) handleConnectListenbrainz(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	username := query.Get("u")
	token := query.Get("token")
	if !query.Has("token") {
		responses.EncodeError(w, query.Get("f"), "missing token parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	var lbUsername *string
	var encryptedListenbrainzToken []byte
	if token != "" {
		con, err := h.ListenBrainz.CheckToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, listenbrainz.ErrUnauthenticated) {
				responses.EncodeError(w, query.Get("f"), "invalid token", responses.SubsonicErrorGeneric)
			} else {
				log.Errorf("connect listenbrainz: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			}
			return
		}
		lbUsername = &con.LBUsername
		encryptedListenbrainzToken, err = db.EncryptPassword(token)
		if err != nil {
			log.Errorf("connect listenbrainz: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	}
	_, err := h.Store.UpdateUserListenBrainzConnection(r.Context(), db.UpdateUserListenBrainzConnectionParams{
		Name:                       username,
		EncryptedListenbrainzToken: encryptedListenbrainzToken,
		ListenbrainzUsername:       lbUsername,
	})
	if err != nil {
		log.Errorf("connect listenbrainz: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	err = h.ListenBrainz.SubmitMissingListens(r.Context())
	if err != nil {
		log.Error(err)
	}

	err = h.ListenBrainz.SyncSongFeedback(r.Context())
	if err != nil {
		log.Error(err)
	}

	res := responses.New()
	res.ListenBrainzConfig = &responses.ListenBrainzConfig{
		ListenBrainzUsername: lbUsername,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

func (h *Handler) handleGetListenbrainzConfig(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	username := query.Get("u")
	user, err := h.Store.FindUser(r.Context(), username)
	if err != nil {
		log.Errorf("get listenbrainz config: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	res := responses.New()
	res.ListenBrainzConfig = &responses.ListenBrainzConfig{
		ListenBrainzUsername: user.ListenbrainzUsername,
	}
	res.EncodeOrLog(w, query.Get("f"))
}
