package handlers

import (
	"fmt"
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

func (h *Handler) handleGetInternetRadioStations(w http.ResponseWriter, r *http.Request) {
	stations, err := h.DB.InternetRadioStation().FindAll(r.Context(), user(r))
	if err != nil {
		respondErr(w, format(r), fmt.Errorf("get internet radio stations: %w", err))
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
	res.EncodeOrLog(w, format(r))
}

func (h *Handler) handleCreateInternetRadioStation(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)

	streamURL, ok := paramStrReq(w, r, "streamUrl")
	if !ok {
		return
	}

	name, ok := paramStrReq(w, r, "name")
	if !ok {
		return
	}

	homepageURL := query.Get("homepageUrl")
	var homepageURLPtr *string
	if homepageURL != "" {
		homepageURLPtr = &homepageURL
	}

	_, err := h.DB.InternetRadioStation().Create(r.Context(), user(r), repos.CreateInternetRadioStationParams{
		Name:        name,
		StreamURL:   streamURL,
		HomepageURL: homepageURLPtr,
	})
	if err != nil {
		respondErr(w, format(r), fmt.Errorf("create internet radio station: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, format(r))
}

func (h *Handler) handleUpdateInternetRadioStation(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)

	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}

	streamURL, ok := paramStrReq(w, r, "streamUrl")
	if !ok {
		return
	}

	name, ok := paramStrReq(w, r, "name")
	if !ok {
		return
	}

	homepageURL := query.Get("homepageUrl")
	var homepageURLOpt repos.Optional[*string]
	if query.Has("homepageUrl") {
		if homepageURL != "" {
			homepageURLOpt = repos.NewOptionalFull(&homepageURL)
		} else {
			homepageURLOpt = repos.NewOptionalFull[*string](nil)
		}
	}

	err := h.DB.InternetRadioStation().Update(r.Context(), user(r), id, repos.UpdateInternetRadioStationParams{
		Name:        repos.NewOptionalFull(name),
		StreamURL:   repos.NewOptionalFull(streamURL),
		HomepageURL: homepageURLOpt,
	})
	if err != nil {
		respondErr(w, format(r), fmt.Errorf("update internet radio station: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, format(r))
}

func (h *Handler) handleDeleteInternetRadioStation(w http.ResponseWriter, r *http.Request) {
	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}

	err := h.DB.InternetRadioStation().Delete(r.Context(), user(r), id)
	if err != nil {
		respondErr(w, format(r), fmt.Errorf("delete internet radio station: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, format(r))
}
