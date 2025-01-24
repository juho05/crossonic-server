package handlers

import (
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
)

// https://opensubsonic.netlify.app/docs/endpoints/ping/
func (h *Handler) handlePing(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	res := responses.New()
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/getlicense/
func (h *Handler) handleGetLicense(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	res := responses.New()
	res.License = &responses.License{Valid: true}
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/getopensubsonicextensions/
func (h *Handler) handleGetOpenSubsonicExtensions(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	res := responses.New()
	res.OpenSubsonicExtensions = &responses.OpenSubsonicExtensions{
		responses.OpenSubsonicExtension{Name: "formPost", Versions: []int{1}},
		responses.OpenSubsonicExtension{Name: "transcodeOffset", Versions: []int{1}},
		responses.OpenSubsonicExtension{Name: "songLyrics", Versions: []int{1}},
	}
	res.EncodeOrLog(w, query.Get("f"))
}
