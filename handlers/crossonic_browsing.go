package handlers

import (
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
)

func (h *Handler) handleGetAppearsOn(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	artistId, ok := paramIDReq(w, r, "artistId")
	if !ok {
		return
	}

	albums, err := h.DB.Artist().GetAppearsOnAlbums(r.Context(), artistId, repos.IncludeAlbumInfoFull(user))
	if err != nil {
		respondErr(w, format(r), err)
		return
	}

	res := responses.New()
	res.AppearsOn = &responses.AppearsOn{
		Albums: responses.NewAlbums(albums, h.Config),
	}
	res.EncodeOrLog(w, format(r))
}
