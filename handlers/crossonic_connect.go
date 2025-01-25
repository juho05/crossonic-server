package handlers

import (
	"fmt"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/juho05/crossonic-server/handlers/connect"
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
		respondInternalErr(w, query.Get("f"), fmt.Errorf("upgrade crossonic connect connection: %w", err))
		return
	}
	h.ConnectionManager.Connect(query.Get("u"), name, platform, conn)
}
