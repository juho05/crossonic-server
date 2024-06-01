package handlers

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/jackc/pgx/v5"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/ffmpeg"
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

	timeOffsetStr := query.Get("timeOffset")
	var timeOffset int
	if timeOffsetStr != "" {
		timeOffset, err = strconv.Atoi(timeOffsetStr)
		if err != nil || timeOffset < 0 {
			responses.EncodeError(w, query.Get("f"), "invalid timeOffset parameter", responses.SubsonicErrorGeneric)
			return
		}
	}

	info, err := h.Store.GetStreamInfo(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("stream: get info: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}

	if maxBitRate > int(info.BitRate) {
		maxBitRate = int(info.BitRate)
	}

	contentType, bitRate := ffmpeg.GetContentTypeFromFormatString(format, maxBitRate)
	if format == "raw" {
		w.Header().Set("Content-Type", info.ContentType)
		contentType = info.ContentType
	} else {
		w.Header().Set("Content-Type", contentType)
	}

	if format == "raw" || (contentType == info.ContentType && (maxBitRate == 0 || maxBitRate == int(info.BitRate))) {
		path := info.Path
		if timeOffset != 0 {
			path, err = h.Transcoder.SeekRaw(path, time.Duration(timeOffset)*time.Second)
			if err != nil {
				log.Errorf("stream: %s", err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
				return
			}
			defer func() {
				err := os.Remove(path)
				if err != nil {
					log.Errorf("failed to delete seek-raw temp file %s: %s", path, err)
				}
			}()
		}
		log.Tracef("Streaming %s raw (%s %dkbps) to %s (user: %s)...", id, info.ContentType, info.BitRate, query.Get("c"), query.Get("u"))
		http.ServeFile(w, r, path)
		return
	}

	w.Header().Set("Accept-Ranges", "none")

	if estimate, _ := strconv.ParseBool(query.Get("estimateContentLength")); estimate {
		w.Header().Set("Content-Length", fmt.Sprint(int(float64(info.DurationMs)/1000*float64(bitRate)/8*1024)))
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	out, bitRate, err := h.Transcoder.Transcode(info.Path, format, maxBitRate, time.Duration(timeOffset)*time.Second)
	if err != nil {
		log.Errorf("stream: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	log.Tracef("Streaming %s transcoded (%s %dkbps) to %s (user: %s)...", id, contentType, bitRate, query.Get("c"), query.Get("u"))
	io.Copy(w, out)
}

func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	info, err := h.Store.GetStreamInfo(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("stream: get info: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}
	http.ServeFile(w, r, info.Path)
}

var lyricsTimestampRegex = regexp.MustCompile(`^\[([0-9]+[:.]?)+\]`)

func (h *Handler) handleGetLyricsBySongId(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	song, err := h.Store.FindSong(r.Context(), id)
	if err != nil {
		responses.EncodeError(w, query.Get("f"), "song not found", responses.SubsonicErrorNotFound)
		return
	}

	res := responses.New()

	if song.Lyrics == nil {
		res.LyricsList = &responses.LyricsList{
			StructuredLyrics: make([]*responses.StructuredLyrics, 0),
		}
		res.EncodeOrLog(w, query.Get("f"))
		return
	}

	lines := strings.Split(*song.Lyrics, "\n")
	for i, l := range lines {
		l = strings.TrimSpace(l)
		loc := lyricsTimestampRegex.FindStringIndex(l)
		if loc != nil {
			l = strings.TrimSpace(l[loc[1]:])
		}
		lines[i] = l
	}
	first := 0
	for ; first < len(lines); first++ {
		if len(lines[first]) > 0 {
			break
		}
	}
	last := len(lines) - 1
	for ; last >= 0; last-- {
		if len(lines[last]) > 0 {
			break
		}
	}
	lines = lines[first : last+1]
	res.LyricsList = &responses.LyricsList{
		StructuredLyrics: []*responses.StructuredLyrics{
			{
				Lang:   "und",
				Synced: false,
				Line: mapData(lines, func(l string) *responses.Line {
					return &responses.Line{
						Value: l,
					}
				}),
			},
		},
	}
	res.EncodeOrLog(w, query.Get("f"))
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
