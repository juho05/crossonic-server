package handlers

import (
	"net/http"

	"github.com/gobwas/ws"
	"github.com/juho05/crossonic-server/handlers/connect"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func (h *Handler) handleConnect(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	name := query.Get("name")
	if name == "" {
		name = "Unnamed device"
	}
	platform := connect.DevicePlatform(query.Get("platform"))
	if !platform.Valid() {
		platform = connect.DevicePlatformUnknown
	}
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Errorf("upgrade crossonic connect connection: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	h.ConnectionManager.Connect(query.Get("u"), name, platform, conn)
}
