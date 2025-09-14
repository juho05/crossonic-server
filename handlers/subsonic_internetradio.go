package handlers

import (
	"fmt"
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

func (h *Handler) handleGetInternetRadioStations(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	stations, err := h.DB.InternetRadioStation().FindAll(r.Context(), q.User())
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("get internet radio stations: %w", err))
		return
	}

	res := responses.New()
	res.InternetRadioStations = &responses.InternetRadioStations{
		Stations: util.Map(stations, func(s *repos.InternetRadioStation) responses.InternetRadioStation {
			return responses.InternetRadioStation{
				ID:          s.ID,
				Name:        s.Name,
				StreamURL:   s.StreamURL,
				HomepageURL: s.HomepageURL,
			}
		}),
	}
	res.EncodeOrLog(w, q.Format())
}

func (h *Handler) handleCreateInternetRadioStation(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	streamURL, ok := q.StrReq("streamUrl")
	if !ok {
		return
	}

	name, ok := q.StrReq("name")
	if !ok {
		return
	}

	homepageURL := util.NilIfEmpty(q.Str("homepageUrl"))

	_, err := h.DB.InternetRadioStation().Create(r.Context(), q.User(), repos.CreateInternetRadioStationParams{
		Name:        name,
		StreamURL:   streamURL,
		HomepageURL: homepageURL,
	})
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("create internet radio station: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, q.Format())
}

func (h *Handler) handleUpdateInternetRadioStation(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	id, ok := q.IDReq("id")
	if !ok {
		return
	}

	streamURL, ok := q.StrReq("streamUrl")
	if !ok {
		return
	}

	name, ok := q.StrReq("name")
	if !ok {
		return
	}

	var homepageURL repos.Optional[*string]
	if q.Has("homepageUrl") {
		homepageURL = repos.NewOptionalFull(util.NilIfEmpty(q.Str("homepageUrl")))
	}

	err := h.DB.InternetRadioStation().Update(r.Context(), q.User(), id, repos.UpdateInternetRadioStationParams{
		Name:        repos.NewOptionalFull(name),
		StreamURL:   repos.NewOptionalFull(streamURL),
		HomepageURL: homepageURL,
	})
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("update internet radio station: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, q.Format())
}

func (h *Handler) handleDeleteInternetRadioStation(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	id, ok := q.IDReq("id")
	if !ok {
		return
	}

	err := h.DB.InternetRadioStation().Delete(r.Context(), q.Format(), id)
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("delete internet radio station: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, q.Format())
}
