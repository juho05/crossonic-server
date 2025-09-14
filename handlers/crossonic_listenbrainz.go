package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/log"
)

func (h *Handler) handleConnectListenbrainz(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	token, ok := q.StrReq("token")
	if !ok {
		return
	}

	var lbUsername *string
	var lbToken *string
	if token != "" {
		con, err := h.ListenBrainz.CheckToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, listenbrainz.ErrUnauthenticated) {
				respondGenericErr(w, q.Format(), "invalid token")
			} else {
				respondInternalErr(w, q.Format(), fmt.Errorf("connect listenbrainz: %w", err))
			}
			return
		}
		lbUsername = &con.LBUsername
		lbToken = &token
	}
	err := h.DB.User().UpdateListenBrainzConnection(r.Context(), q.User(), lbUsername, lbToken)
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("connect listenbrainz: %w", err))
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
	res.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleGetListenbrainzConfig(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)
	user, err := h.DB.User().FindByName(r.Context(), q.User())
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get listenbrainz config: %w", err))
		return
	}
	res := responses.New()
	res.ListenBrainzConfig = &responses.ListenBrainzConfig{
		ListenBrainzUsername: user.ListenBrainzUsername,
	}
	res.EncodeOrLog(w, q.Format())
}
