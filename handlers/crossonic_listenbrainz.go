package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

func (h *Handler) handleConnectListenbrainz(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	token := q.Str("token")

	var lbUsername *string
	var lbToken *string
	if token != "" {
		lbUser, err := h.ListenBrainz.CheckToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, listenbrainz.ErrUnauthenticated) {
				respondGenericErr(w, q.Format(), "invalid token")
			} else {
				respondInternalErr(w, q.Format(), fmt.Errorf("connect listenbrainz: %w", err))
			}
			return
		}
		lbUsername = &lbUser
		lbToken = &token
	}
	err := h.DB.User().UpdateListenBrainzConnection(r.Context(), q.User(), lbUsername, lbToken)
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("connect listenbrainz: %w", err))
		return
	}

	user, err := h.DB.User().FindByName(r.Context(), q.User())
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("connect listenbrainz: find user: %w", err))
		return
	}

	go func() {
		if user.ListenBrainzUsername == nil {
			return
		}

		if user.ListenBrainzScrobble {
			err = h.ListenBrainz.SubmitMissingListens(context.Background())
			if err != nil {
				log.Error(err)
			}
		}

		if user.ListenBrainzSyncFeedback {
			err = h.ListenBrainz.SyncSongFeedback(context.Background())
			if err != nil {
				log.Error(err)
			}
		}
	}()

	res := responses.New()
	res.ListenBrainzConfig = &responses.ListenBrainzConfig{
		ListenBrainzUsername: user.ListenBrainzUsername,
		Scrobble:             util.ToPtr(user.ListenBrainzScrobble),
		SyncFeedback:         util.ToPtr(user.ListenBrainzSyncFeedback),
	}
	res.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleUpdateListenbrainzConfig(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	var scrobble repos.Optional[bool]
	if q.Has("scrobble") {
		s, ok := q.Bool("scrobble")
		if !ok {
			return
		}
		scrobble = repos.NewOptionalFull(s)
	}

	var syncFeedback repos.Optional[bool]
	if q.Has("syncFeedback") {
		s, ok := q.Bool("syncFeedback")
		if !ok {
			return
		}
		syncFeedback = repos.NewOptionalFull(s)
	}

	err := h.DB.User().UpdateListenBrainzSettings(r.Context(), q.User(), repos.UpdateListenBrainzSettingsParams{
		Scrobble:     scrobble,
		SyncFeedback: syncFeedback,
	})
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("update listenbrainz config: %w", err))
		return
	}

	user, err := h.DB.User().FindByName(r.Context(), q.User())
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("get listenbrainz config: %w", err))
		return
	}

	go func() {
		if scrobble.HasValue() && user.ListenBrainzScrobble {
			err = h.ListenBrainz.SubmitMissingListens(context.Background())
			if err != nil {
				log.Error(err)
			}
		}

		if syncFeedback.HasValue() && user.ListenBrainzSyncFeedback {
			err = h.ListenBrainz.SyncSongFeedback(context.Background())
			if err != nil {
				log.Error(err)
			}
		}
	}()

	res := responses.New()
	res.ListenBrainzConfig = &responses.ListenBrainzConfig{
		ListenBrainzUsername: user.ListenBrainzUsername,
		Scrobble:             util.ToPtr(user.ListenBrainzScrobble),
		SyncFeedback:         util.ToPtr(user.ListenBrainzSyncFeedback),
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
	if user.ListenBrainzUsername != nil {
		res.ListenBrainzConfig = &responses.ListenBrainzConfig{
			ListenBrainzUsername: user.ListenBrainzUsername,
			Scrobble:             util.ToPtr(user.ListenBrainzScrobble),
			SyncFeedback:         util.ToPtr(user.ListenBrainzSyncFeedback),
		}
	} else {
		res.ListenBrainzConfig = &responses.ListenBrainzConfig{}
	}
	res.EncodeOrLog(w, q.Format())
}
