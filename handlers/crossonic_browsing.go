package handlers

import (
	"fmt"
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
)

func (h *Handler) handleGetAppearsOn(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	artistId, ok := q.IDReq("artistId")
	if !ok {
		return
	}

	albums, err := h.DB.Artist().GetAppearsOnAlbums(r.Context(), artistId, repos.IncludeAlbumInfoFull(q.User()))
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("get appears on albums: %w", err))
		return
	}

	res := responses.New()
	res.AppearsOn = &responses.AppearsOn{
		Albums: responses.NewAlbums(albums, h.Config),
	}
	res.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleGetAlternateAlbumVersions(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	albumId, ok := q.IDReq("albumId")
	if !ok {
		return
	}

	albums, err := h.DB.Album().GetAlternateVersions(r.Context(), albumId, repos.IncludeAlbumInfoFull(q.User()))
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("get album versions: %w", err))
		return
	}

	res := responses.New()
	res.AlbumVersions = &responses.AlbumVersions{
		Albums: responses.NewAlbums(albums, h.Config),
	}
	res.EncodeOrLog(w, q.Format())
}
