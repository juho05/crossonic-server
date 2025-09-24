package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/scanner"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

// https://opensubsonic.netlify.app/docs/endpoints/startscan
func (h *Handler) handleStartScan(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	res := responses.New()

	fullScan, ok := q.BoolDef("fullScan", false)
	if !ok {
		return
	}

	startTime := time.Now()

	var lastScan *time.Time
	ls, err := h.DB.System().LastScan(r.Context())
	if err != nil {
		if !errors.Is(err, repos.ErrNotFound) {
			respondErr(w, q.Format(), err)
			return
		}
		lastScan = nil
	} else {
		lastScan = &ls
	}

	if h.Scanner.Scanning() {
		res.ScanStatus = &responses.ScanStatus{
			Scanning:  true,
			Count:     util.ToPtr(h.Scanner.Count()),
			LastScan:  lastScan,
			FullScan:  h.Scanner.IsFullScan(),
			StartTime: util.ToPtr(h.Scanner.ScanStart()),
		}
		res.EncodeOrLog(w, q.Format())
		return
	}

	go func() {
		if fullScan {
			log.Infof("manual full scan triggered by %s", q.User())
		} else {
			log.Infof("manual quick scan triggered by %s", q.User())
		}
		err := h.Scanner.Scan(h.DB, fullScan)
		if err != nil && !errors.Is(err, scanner.ErrAlreadyScanning) {
			log.Errorf("scan media full: %s", err)
		}
	}()

	res.ScanStatus = &responses.ScanStatus{
		Scanning:  true,
		Count:     util.ToPtr(0),
		LastScan:  lastScan,
		FullScan:  fullScan,
		StartTime: &startTime,
	}
	res.EncodeOrLog(w, q.Format())
}

// https://opensubsonic.netlify.app/docs/endpoints/getscanstatus
func (h *Handler) handleGetScanStatus(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)
	res := responses.New()

	var lastScan *time.Time
	ls, err := h.DB.System().LastScan(r.Context())
	if err != nil {
		if !errors.Is(err, repos.ErrNotFound) {
			respondErr(w, q.Format(), err)
			return
		}
		lastScan = nil
	} else {
		lastScan = &ls
	}

	if h.Scanner.Scanning() {
		res.ScanStatus = &responses.ScanStatus{
			Scanning:  true,
			Count:     util.ToPtr(h.Scanner.Count()),
			LastScan:  lastScan,
			FullScan:  h.Scanner.IsFullScan(),
			StartTime: util.ToPtr(h.Scanner.ScanStart()),
		}
	} else {
		res.ScanStatus = &responses.ScanStatus{
			Scanning: false,
			Count:    util.ToPtr(h.Scanner.Count()),
			LastScan: lastScan,
		}
	}

	res.EncodeOrLog(w, q.Format())
}
