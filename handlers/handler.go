package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/juho05/crossonic-server/cache"
	"github.com/juho05/crossonic-server/config"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/ffmpeg"
	"github.com/juho05/crossonic-server/handlers/connect"
	"github.com/juho05/crossonic-server/lastfm"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/crossonic-server/scanner"
	"github.com/juho05/log"
)

type Handler struct {
	router            chi.Router
	Store             db.Store
	Scanner           *scanner.Scanner
	ListenBrainz      *listenbrainz.ListenBrainz
	LastFM            *lastfm.LastFm
	Transcoder        *ffmpeg.Transcoder
	ConnectionManager *connect.ConnectionManager

	CoverCache     *cache.Cache
	TranscodeCache *cache.Cache
}

func New(store db.Store, scanner *scanner.Scanner, listenBrainz *listenbrainz.ListenBrainz, lastFM *lastfm.LastFm, transcoder *ffmpeg.Transcoder, transcodeCache *cache.Cache, coverCache *cache.Cache) *Handler {
	h := &Handler{
		Store:             store,
		Scanner:           scanner,
		ListenBrainz:      listenBrainz,
		LastFM:            lastFM,
		Transcoder:        transcoder,
		ConnectionManager: connect.NewConnectionManager(store),
		TranscodeCache:    transcodeCache,
		CoverCache:        coverCache,
	}
	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.With(ignoreExtension).Route("/rest/crossonic", h.registerCrossonicRoutes)
	r.With(ignoreExtension).Route("/rest", h.registerSubsonicRoutes)
	if config.FrontendDir() != "" {
		r.Mount("/", http.FileServer(http.Dir(config.FrontendDir())))
		log.Infof("Serving frontend files in %s", config.FrontendDir())
	} else {
		log.Trace("Frontend hosting disabled")
	}

	h.router = r
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	middleware.StripSlashes(h.router).ServeHTTP(w, r)
}
