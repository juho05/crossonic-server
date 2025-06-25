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
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/lastfm"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
)

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	id, ok := paramIDReq(w, r, "id")
	if !ok {
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

	info, err := h.DB.Song().GetStreamInfo(r.Context(), id)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
		} else {
			log.Errorf("stream: get info: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		}
		return
	}

	if maxBitRate > info.BitRate {
		maxBitRate = info.BitRate
	}

	fileFormat, bitRate := h.Transcoder.SelectFormat(format, info.ChannelCount, maxBitRate)

	if format == "raw" || (fileFormat.Mime == info.ContentType && (maxBitRate == 0 || maxBitRate >= info.BitRate)) {
		format = "raw"
		fileFormat.Mime = info.ContentType
		fileFormat.Name = strings.TrimPrefix(filepath.Ext(info.Path), ".")
		if timeOffset == 0 {
			log.Tracef("Streaming %s raw (%s %dkbps) to %s (user: %s) (range: %s)...", id, info.ContentType, info.BitRate, query.Get("c"), query.Get("u"), r.Header.Get("Range"))
			http.ServeFile(w, r, info.Path)
			return
		}
	}
	w.Header().Set("Content-Type", fileFormat.Mime)

	if estimate, _ := strconv.ParseBool(query.Get("estimateContentLength")); estimate {
		w.Header().Set("Content-Length", fmt.Sprint(int(float64(info.Duration.ToStd().Milliseconds()-int64(timeOffset*1000))/1000*float64(bitRate)/8*1024)))
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	if timeOffset != 0 {
		done := make(chan struct{})
		w.Header().Set("Accept-Ranges", "none")
		if format == "raw" {
			err = h.Transcoder.SeekRaw(info.Path, time.Duration(timeOffset)*time.Second, w, func() {
				close(done)
			})
			log.Tracef("Streaming %s with offset (%ds) (%s %dkbps) to %s (user: %s)...", id, timeOffset, info.ContentType, info.BitRate, query.Get("c"), query.Get("u"))
		} else {
			bitRate, err = h.Transcoder.Transcode(info.Path, info.ChannelCount, fileFormat, bitRate, time.Duration(timeOffset)*time.Second, w, func() {
				close(done)
			})
			log.Tracef("Streaming %s with transcoded offset (%ds) (%s %dkbps) to %s (user: %s)...", id, timeOffset, fileFormat.Name, bitRate, query.Get("c"), query.Get("u"))
		}
		if err != nil {
			log.Errorf("stream: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		<-done
		return
	}

	cacheKey := fmt.Sprintf("%s-%s-%d", id, fileFormat.Name, bitRate)

	cacheObj, exists := h.TranscodeCache.GetObject(cacheKey)
	if !exists {
		cacheObj, err = h.TranscodeCache.CreateObject(cacheKey)
		if err != nil {
			respondErr(w, query.Get("f"), fmt.Errorf("stream: %w", err))
			return
		}
		bitRate, err = h.Transcoder.Transcode(info.Path, info.ChannelCount, fileFormat, bitRate, 0, cacheObj, func() {
			err := cacheObj.SetComplete()
			if err != nil {
				log.Errorf("ffmpeg: transcode: %s", err)
			}
		})
		if err != nil {
			log.Errorf("stream: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			err = h.TranscodeCache.DeleteObject(cacheKey)
			if err != nil {
				log.Errorf("stream: %s", err)
			}
			return
		}
		log.Tracef("Streaming %s transcoded (%s %dkbps) to %s (user: %s) (new transcode)...", id, fileFormat.Name, bitRate, query.Get("c"), query.Get("u"))
	} else {
		log.Tracef("Streaming %s transcoded (%s %dkbps) to %s (user: %s) (cached (complete: %t)) (range: %s)...", id, fileFormat.Name, bitRate, query.Get("c"), query.Get("u"), cacheObj.IsComplete(), r.Header.Get("Range"))
	}

	cacheReader, err := cacheObj.Reader()
	if err != nil {
		log.Errorf("stream: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer func() {
		err = cacheReader.Close()
		if err != nil {
			log.Errorf("stream: %s", err)
		}
	}()

	if cacheObj.IsComplete() {
		http.ServeContent(w, r, id, cacheObj.Modified(), cacheReader)
	} else {
		w.Header().Set("Accept-Ranges", "none")
		_, _ = io.Copy(w, cacheReader)
	}
}

func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}
	info, err := h.DB.Song().GetStreamInfo(r.Context(), id)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
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

func (h *Handler) handleGetLyrics(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)

	title := query.Get("title")
	artist := query.Get("artist")

	if title == "" {
		responses.EncodeError(w, format(r), "song not found", responses.SubsonicErrorNotFound)
		return
	}

	songs, err := h.DB.Song().FindByTitle(r.Context(), title, repos.IncludeSongInfo{
		Lists: true,
	})
	if err != nil {
		respondErr(w, format(r), fmt.Errorf("get lyrics: find songs by title: %w", err))
		return
	}

	var song *repos.CompleteSong

	if artist != "" {
	songLoop:
		for _, s := range songs {
			for _, a := range s.Artists {
				if a.Name == artist {
					song = s
					break songLoop
				}
			}
		}
	} else if len(songs) > 0 {
		song = songs[0]
	}

	if song == nil {
		responses.EncodeError(w, format(r), "song not found", responses.SubsonicErrorNotFound)
		return
	}

	if song.Lyrics == nil {
		responses.EncodeError(w, format(r), "no lyrics available", responses.SubsonicErrorNotFound)
		return
	}

	lines := getLyricsLines(*song.Lyrics)

	if artist == "" && len(song.Artists) > 0 {
		artist = song.Artists[0].Name
	}

	res := responses.New()
	res.Lyrics = &responses.Lyrics{
		Title:  song.Title,
		Artist: &artist,
		Value:  strings.Join(lines, "\n"),
	}
	res.EncodeOrLog(w, format(r))
}

func (h *Handler) handleGetLyricsBySongId(w http.ResponseWriter, r *http.Request) {
	id, ok := paramIDReq(w, r, "id")
	if !ok {
		return
	}
	song, err := h.DB.Song().FindByID(r.Context(), id, repos.IncludeSongInfoBare())
	if err != nil {
		responses.EncodeError(w, format(r), "song not found", responses.SubsonicErrorNotFound)
		return
	}

	res := responses.New()

	if song.Lyrics == nil {
		res.LyricsList = &responses.LyricsList{
			StructuredLyrics: make([]*responses.StructuredLyrics, 0),
		}
		res.EncodeOrLog(w, format(r))
		return
	}

	lines := getLyricsLines(*song.Lyrics)

	res.LyricsList = &responses.LyricsList{
		StructuredLyrics: []*responses.StructuredLyrics{
			{
				Lang:   "und",
				Synced: false,
				Line: util.Map(lines, func(l string) *responses.Line {
					return &responses.Line{
						Value: l,
					}
				}),
			},
		},
	}
	res.EncodeOrLog(w, format(r))
}

func (h *Handler) handleGetCoverArt(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	id, ok := paramIDReq(w, r, "id")
	if !ok {
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

	cacheKey := fmt.Sprintf("%s-%d", id, size)

	cacheObj, exists := h.CoverCache.GetObject(cacheKey)
	if exists {
		cacheReader, err := cacheObj.Reader()
		if err != nil {
			log.Errorf("get cover art: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
		defer func() {
			err = cacheReader.Close()
			if err != nil {
				log.Errorf("get cover art: %s", err)
			}
		}()
		w.Header().Set("Cache-Control", "max-age=10080") // 3h
		w.Header().Set("Content-Type", "image/jpeg")
		if cacheObj.IsComplete() {
			http.ServeContent(w, r, id+".jpg", time.Now(), cacheReader)
		} else {
			_, _ = io.Copy(w, cacheReader)
		}
		return
	}

	dir := filepath.Join(h.Config.DataDir, "covers")
	fileFS := os.DirFS(dir)
	file, err := fileFS.Open(id)
	if errors.Is(err, fs.ErrNotExist) {
		if h.Config.LastFMApiKey == "" {
			responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
			return
		}
		err = h.loadArtistCoverFromLastFMByID(r.Context(), id)
		if errors.Is(err, repos.ErrNotFound) {
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
	w.Header().Set("Cache-Control", "max-age=10080") // 3h
	w.WriteHeader(http.StatusOK)
	cacheObj, err = h.CoverCache.CreateObject(cacheKey)
	if err != nil {
		respondErr(w, format(r), fmt.Errorf("get cover art: %w", err))
		return
	}
	go func() {
		err = imaging.Encode(cacheObj, img, imaging.JPEG)
		if err != nil {
			log.Errorf("get cover art: encode %s: %s", id, err)
			err = h.CoverCache.DeleteObject(cacheKey)
			if err != nil {
				log.Errorf("get cover art: %s", err)
			}
			return
		}
		err = cacheObj.SetComplete()
		if err != nil {
			log.Errorf("get cover art: %s", err)
		}
	}()
	cacheReader, err := cacheObj.Reader()
	if err != nil {
		respondErr(w, format(r), fmt.Errorf("get cover art: %w", err))
		return
	}
	defer cacheObj.Close()
	_, _ = io.Copy(w, cacheReader)
}

func (h *Handler) loadArtistCoverFromLastFMByID(ctx context.Context, id string) error {
	artist, err := h.DB.Artist().FindByID(ctx, id, repos.IncludeArtistInfoBare())
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: %w", err)
	}
	info, err := h.LastFM.GetArtistInfo(ctx, artist.Name, artist.MusicBrainzID)
	if err != nil {
		return fmt.Errorf("load artist cover from last fm by id: %w", err)
	}
	imageURL, err := h.LastFM.GetArtistImageURL(*info.URL)
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
	file, err := os.Create(filepath.Join(h.Config.DataDir, "covers", id))
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

func getLyricsLines(lyrics string) []string {
	lines := strings.Split(lyrics, "\n")
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
	return lines
}
