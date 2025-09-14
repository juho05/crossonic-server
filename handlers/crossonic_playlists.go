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
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
)

const maxPlaylistCoverBytes = 15e6 // 15 MB

func (h *Handler) handleSetPlaylistCover(w http.ResponseWriter, r *http.Request) {
	q := getQuery(w, r)

	if r.Body != nil {
		defer r.Body.Close()
	}

	id, ok := q.IDReq("id")
	if !ok {
		return
	}

	_, err := h.DB.Playlist().FindByID(r.Context(), q.User(), id, repos.IncludePlaylistInfoBare())
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("set playlist cover: get playlist: %w", err))
		return
	}

	path := filepath.Join(h.Config.DataDir, "covers", id)
	if r.Body == nil || r.ContentLength <= 0 {
		err = os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			respondInternalErr(w, q.Format(), fmt.Errorf("set playlist cover: delete cover: %w", err))
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
		responses.New().EncodeOrLog(w, q.Format())
		return
	}

	if r.ContentLength > maxPlaylistCoverBytes {
		respondGenericErr(w, q.Format(), "request body too large")
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxPlaylistCoverBytes)
	img, err := imaging.Decode(body, imaging.AutoOrientation(true))
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			respondGenericErr(w, q.Format(), "request body too large")
		} else if errors.Is(err, image.ErrFormat) || errors.Is(err, imaging.ErrUnsupportedFormat) {
			respondGenericErr(w, q.Format(), "unsupported image type")
		} else {
			respondInternalErr(w, q.Format(), fmt.Errorf("set playlist cover: decode image: %w", err))
		}
		return
	}
	targetSize := min(2048, min(img.Bounds().Dx(), img.Bounds().Dy()))
	img = imaging.Thumbnail(img, targetSize, targetSize, imaging.Linear)

	file, err := os.Create(filepath.Join(h.Config.DataDir, "covers", id))
	if err != nil {
		respondInternalErr(w, q.Format(), fmt.Errorf("set playlist cover: save image: create file: %w", err))
		return
	}
	err = imaging.Encode(file, img, imaging.JPEG)
	file.Close()
	if err != nil {
		_ = os.Remove(path)
		respondInternalErr(w, q.Format(), fmt.Errorf("set playlist cover: save image: encode: %w", err))
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

	err = h.DB.Playlist().Update(r.Context(), q.User(), id, repos.UpdatePlaylistParams{})
	if err != nil {
		respondErr(w, q.Format(), fmt.Errorf("set playlist cover: update playlist updated time: %w", err))
		return
	}

	responses.New().EncodeOrLog(w, q.Format())
}
