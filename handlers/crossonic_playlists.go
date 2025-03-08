package handlers

import (
	"errors"
	"fmt"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

const maxPlaylistCoverBytes = 15e6 // 15 MB

func (h *Handler) handleSetPlaylistCover(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	if r.Body != nil {
		defer r.Body.Close()
	}

	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}

	_, err := h.DB.Playlist().FindByID(r.Context(), user, id, repos.IncludePlaylistInfoBare())
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("set playlist cover: get playlist: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}

	path := filepath.Join(config.DataDir(), "covers", id)
	if r.Body == nil || r.ContentLength <= 0 {
		err = os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Errorf("set playlist cover: delete cover: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		for _, k := range h.CoverCache.Keys() {
			parts := strings.Split(k, "-")
			pID := strings.Join(parts[:len(parts)-1], "-")
			if pID == id {
				err := h.CoverCache.DeleteObject(k)
				if err != nil {
					log.Errorf("set playlist cover: invalidate cache: %s", err)
				}
			}
		}
		responses.New().EncodeOrLog(w, query.Get("f"))
		return
	}

	if r.ContentLength > maxPlaylistCoverBytes {
		responses.EncodeError(w, query.Get("f"), "request body too large", responses.SubsonicErrorGeneric)
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxPlaylistCoverBytes)
	img, err := imaging.Decode(body, imaging.AutoOrientation(true))
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			responses.EncodeError(w, query.Get("f"), "request body too large", responses.SubsonicErrorGeneric)
		} else if errors.Is(err, image.ErrFormat) || errors.Is(err, imaging.ErrUnsupportedFormat) {
			responses.EncodeError(w, query.Get("f"), "unsupported image type", responses.SubsonicErrorGeneric)
		} else {
			log.Errorf("set playlist cover: decode image: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}
	targetSize := min(2048, min(img.Bounds().Dx(), img.Bounds().Dy()))
	img = imaging.Thumbnail(img, targetSize, targetSize, imaging.Linear)

	file, err := os.Create(filepath.Join(config.DataDir(), "covers", id))
	if err != nil {
		log.Errorf("set playlist cover: save image: create file: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	err = imaging.Encode(file, img, imaging.JPEG)
	file.Close()
	if err != nil {
		os.Remove(path)
		log.Errorf("set playlist cover: save image: encode: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	for _, k := range h.CoverCache.Keys() {
		parts := strings.Split(k, "-")
		pID := strings.Join(parts[:len(parts)-1], "-")
		if pID == id {
			err := h.CoverCache.DeleteObject(k)
			if err != nil {
				log.Errorf("set playlist cover: invalidate cache: %s", err)
			}
		}
	}

	err = h.DB.Playlist().Update(r.Context(), user, id, repos.UpdatePlaylistParams{})
	if err != nil {
		respondErr(w, format(r), fmt.Errorf("set playlist cover: update playlist updated time: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, query.Get("f"))
}
