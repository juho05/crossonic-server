package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/juho05/crossonic-server/cache"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/ffmpeg"
	"github.com/juho05/crossonic-server/lastfm"
	"github.com/juho05/crossonic-server/listenbrainz"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/scanner"
	"github.com/juho05/log"
)

type Handler struct {
	router       chi.Router
	DB           repos.DB
	Scanner      *scanner.Scanner
	ListenBrainz *listenbrainz.ListenBrainz
	LastFM       *lastfm.LastFm
	Transcoder   *ffmpeg.Transcoder

	CoverCache     *cache.Cache
	TranscodeCache *cache.Cache

	Config config.Config

	// user -> timestamps of failed logins
	authFailures     map[string][]time.Time
	authFailuresLock sync.RWMutex
	authCleanupStop  chan struct{}

	// dummyEncryptedPassword holds a decoy encrypted password used to perform
	// the same cryptographic work for non-existent users as for real ones,
	// preventing username enumeration via timing analysis (see dummyTokenAuth).
	dummyEncryptedPassword []byte
}

func New(conf config.Config, db repos.DB, scanner *scanner.Scanner, listenBrainz *listenbrainz.ListenBrainz, lastFM *lastfm.LastFm, transcoder *ffmpeg.Transcoder, transcodeCache *cache.Cache, coverCache *cache.Cache) (*Handler, error) {
	h := &Handler{
		DB:              db,
		Scanner:         scanner,
		ListenBrainz:    listenBrainz,
		LastFM:          lastFM,
		Transcoder:      transcoder,
		TranscodeCache:  transcodeCache,
		CoverCache:      coverCache,
		Config:          conf,
		authFailures:    make(map[string][]time.Time),
		authCleanupStop: make(chan struct{}),
	}
	var err error
	h.dummyEncryptedPassword, err = repos.EncryptPassword("dummy-password", conf.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt dummy password: %w", err)
	}

	h.registerRoutes()
	go h.cleanAuthFailures()
	return h, nil
}

func (h *Handler) Close() {
	select {
	case <-h.authCleanupStop:
		break
	default:
		close(h.authCleanupStop)
	}
}

func (h *Handler) cleanAuthFailures() {
	ticker := time.NewTicker(authRateWindow)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-authRateWindow)
			h.authFailuresLock.Lock()
			for username, failures := range h.authFailures {
				pruned := failures[:0]
				for _, t := range failures {
					if t.After(cutoff) {
						pruned = append(pruned, t)
					}
				}
				if len(pruned) == 0 {
					delete(h.authFailures, username)
				} else {
					h.authFailures[username] = pruned
				}
			}
			h.authFailuresLock.Unlock()
		case <-h.authCleanupStop:
			return
		}
	}
}

func (h *Handler) registerRoutes() {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Route("/rest/crossonic", h.registerCrossonicRoutes)
	r.Route("/rest", h.registerSubsonicRoutes)
	if h.Config.FrontendDir != "" {
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// required for drift web support:
					// https://drift.simonbinder.eu/platforms/web/#additional-headers
					w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
					w.Header().Set("Cross-Origin-Embedder-Policy", "credentialless")
					next.ServeHTTP(w, r)
				})
			})
			r.Mount("/", http.FileServer(http.Dir(h.Config.FrontendDir)))
		})

		log.Infof("Serving frontend files in %s", h.Config.FrontendDir)
	} else {
		log.Trace("Frontend hosting disabled")
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("crossonic-server is ready to accept connections"))
		})
	}

	h.router = r
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	middleware.StripSlashes(h.router).ServeHTTP(w, r)
}
