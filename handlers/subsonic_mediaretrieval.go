package handlers

import (
	"context"
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
	"github.com/juho05/crossonic-server/lastfm"
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

	if !crossonic.IDRegex.MatchString(id) {
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

	var dir string
	switch idType {
	case crossonic.IDTypeSong:
		dir = filepath.Join(config.DataDir(), "covers", "songs")
	case crossonic.IDTypeAlbum:
		dir = filepath.Join(config.DataDir(), "covers", "albums")
	case crossonic.IDTypeArtist:
		dir = filepath.Join(config.DataDir(), "covers", "artists")
	}
	fileFS := os.DirFS(dir)
	file, err := fileFS.Open(id)
	if errors.Is(err, fs.ErrNotExist) {
		if config.LastFMApiKey() == "" {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			return
		}
		err = h.loadArtistCoverFromLastFMByID(r.Context(), id)
		if errors.Is(err, pgx.ErrNoRows) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			return
		}
		if errors.Is(err, lastfm.ErrNotFound) {
			file, err := os.Create(filepath.Join(dir, id))
			if err != nil {
				log.Errorf("get cover art: create placeholder for %s: %s", id, err)
				responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
				return
			}
			file.Close()
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			return
		}
		if err == nil {
			file, err = fileFS.Open(id)
		}
	}
	if err != nil {
		log.Errorf("get cover art: open %s: %s", id, err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		log.Errorf("get cover art: stat %s: %s", id, err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	if stat.Size() == 0 || stat.IsDir() {
		file.Close()
		responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
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

func (h *Handler) loadArtistCoverFromLastFMByID(ctx context.Context, id string) error {
	artist, err := h.Store.FindArtistSimple(ctx, id)
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: %w", err)
	}
	info, err := h.LastFM.GetArtistInfo(ctx, artist.Name, artist.MusicBrainzID)
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: %w", err)
	}
	imageURL, err := h.LastFM.GetArtistImageURL(ctx, info.URL)
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: %w", err)
	}
	if imageURL == "" {
		return lastfm.ErrNotFound
	}
	res, err := http.Get(imageURL)
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: download image: %w", err)
	}
	defer res.Body.Close()
	img, err := imaging.Decode(res.Body, imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: decode image: %w", err)
	}
	if img.Bounds().Dx() != img.Bounds().Dy() {
		size := min(img.Bounds().Dx(), img.Bounds().Dy())
		img = imaging.CropCenter(img, size, size)
	}
	file, err := os.Create(filepath.Join(config.DataDir(), "covers", "artists", id))
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: create file: %w", err)
	}
	defer file.Close()
	err = imaging.Encode(file, img, imaging.JPEG)
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: encode image: %w", err)
	}
	return nil
}
