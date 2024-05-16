package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/scanner"
)

type Handler struct {
	router  chi.Router
	Store   db.Store
	Scanner *scanner.Scanner
}

func New(store db.Store, scanner *scanner.Scanner) *Handler {
	h := &Handler{
		Store:   store,
		Scanner: scanner,
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

	r.Route("/rest/crossonic", h.registerCrossonicRoutes)
	r.Route("/rest", h.registerSubsonicRoutes)

	h.router = r
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	middleware.StripSlashes(h.router).ServeHTTP(w, r)
}
