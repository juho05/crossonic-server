package handlers

import (
	"errors"
	"net/http"

	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/scanner"
	"github.com/juho05/log"
)

// https://opensubsonic.netlify.app/docs/endpoints/startscan
func (h *Handler) handleStartScan(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	res := responses.New()

	go func() {
		err := h.Scanner.ScanMediaFull()
		if err != nil && !errors.Is(err, scanner.ErrAlreadyScanning) {
			log.Errorf("scan media full: %s", err)
		}
	}()

	res.ScanStatus = &responses.ScanStatus{
		Scanning: true,
	}
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/getscanstatus
func (h *Handler) handleGetScanStatus(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	res := responses.New()
	res.ScanStatus = &responses.ScanStatus{
		Scanning: h.Scanner.Scanning,
		Count:    &h.Scanner.Count,
	}
	res.EncodeOrLog(w, query.Get("f"))
}
