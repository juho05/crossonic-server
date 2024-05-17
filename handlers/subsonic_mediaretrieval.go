package handlers

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/jackc/pgx/v5"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	format := query.Get("format")
	if format == "" {
		format = "raw"
	}
	if format != "raw" {
		responses.EncodeError(w, query.Get("f"), "transcoding is currently not supported", responses.SubsonicErrorGeneric)
		return
	}

	maxBitRateStr := query.Get("maxBitRate")
	var maxBitRate int
	var err error
	if maxBitRateStr != "" {
		maxBitRate, err = strconv.Atoi(maxBitRateStr)
		if err != nil || maxBitRate < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid maxBitRate parameter", responses.SubsonicErrorGeneric)
			return
		}
	}
	if maxBitRate != 0 {
		responses.EncodeError(w, query.Get("f"), "transcoding is currently not supported", responses.SubsonicErrorGeneric)
		return
	}

	timeOffsetStr := query.Get("timeOffset")
	var timeOffset int
	if timeOffsetStr != "" {
		timeOffset, err = strconv.Atoi(timeOffsetStr)
		if err != nil || timeOffset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid timeOffset parameter", responses.SubsonicErrorGeneric)
			return
		}
	}
	if timeOffset != 0 {
		responses.EncodeError(w, query.Get("f"), "time offset is currently not supported", responses.SubsonicErrorGeneric)
		return
	}

	path, err := h.Store.GetSongPath(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("stream: get path: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}

	http.ServeFile(w, r, path)
}

func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	path, err := h.Store.GetSongPath(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("stream: get path: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}
	http.ServeFile(w, r, path)
}

func (h *Handler) handleGetCoverArt(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	idType, ok := crossonic.GetIDType(id)
	if !ok {
		responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
		return
	}

	if strings.Contains(id, "/") || strings.Contains(id, ".") {
		responses.EncodeError(w, query.Get("f"), "invalid id", responses.SubsonicErrorNotFound)
		return
	}

	size := 2048
	var err error
	sizeStr := query.Get("size")
	if sizeStr != "" {
		size, err = strconv.Atoi(sizeStr)
		if err != nil || size < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid size parameter", responses.SubsonicErrorGeneric)
			return
		}
	}

	var fileFS fs.FS
	switch idType {
	case crossonic.IDTypeSong:
		fileFS = os.DirFS(filepath.Join(config.DataDir(), "covers", "songs"))
	case crossonic.IDTypeAlbum:
		fileFS = os.DirFS(filepath.Join(config.DataDir(), "covers", "albums"))
	case crossonic.IDTypeArtist:
		fileFS = os.DirFS(filepath.Join(config.DataDir(), "covers", "artists"))
	}
	file, err := fileFS.Open(id)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}
	img, err := imaging.Decode(file, imaging.AutoOrientation(true))
	file.Close()
	if err != nil {
		log.Errorf("get cover art: decode %s: %s", id, err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	size = min(size, min(img.Bounds().Dx(), img.Bounds().Dy()))
	img = imaging.Thumbnail(img, size, size, imaging.Linear)
	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	err = imaging.Encode(w, img, imaging.JPEG)
	if err != nil {
		if !strings.Contains(err.Error(), "broken pipe") {
			log.Errorf("get cover art: encode %s: %s", id, err)
		}
	}
}
