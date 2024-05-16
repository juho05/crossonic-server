package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/juho05/crossonic-server"
	db "github.com/juho05/crossonic-server/db/sqlc"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/log"
)

// https://opensubsonic.netlify.app/docs/endpoints/setrating/
func (h *Handler) handleSetRating(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	id := query.Get("id")
	if id == "" {
		responses.EncodeError(w, query.Get("f"), "missing id parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}
	ratingStr := query.Get("rating")
	if ratingStr == "" {
		responses.EncodeError(w, query.Get("f"), "missing rating parameter", responses.SubsonicErrorRequiredParameterMissing)
		return
	}

	idType, ok := crossonic.GetIDType(id)
	if !ok {
		responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
		return
	}

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 0 || rating > 5 {
		responses.EncodeError(w, query.Get("f"), "invalid rating parameter", responses.SubsonicErrorNotFound)
		return
	}

	switch idType {
	case crossonic.IDTypeSong:
		if rating == 0 {
			err = h.Store.RemoveSongRating(r.Context(), db.RemoveSongRatingParams{
				UserName: user,
				SongID:   id,
			})
		} else {
			err = h.Store.SetSongRating(r.Context(), db.SetSongRatingParams{
				SongID:   id,
				UserName: user,
				Rating:   int32(rating),
			})
		}
	case crossonic.IDTypeAlbum:
		if rating == 0 {
			err = h.Store.RemoveAlbumRating(r.Context(), db.RemoveAlbumRatingParams{
				UserName: user,
				AlbumID:  id,
			})
		} else {
			err = h.Store.SetAlbumRating(r.Context(), db.SetAlbumRatingParams{
				AlbumID:  id,
				UserName: user,
				Rating:   int32(rating),
			})
		}
	case crossonic.IDTypeArtist:
		if rating == 0 {
			err = h.Store.RemoveArtistRating(r.Context(), db.RemoveArtistRatingParams{
				UserName: user,
				ArtistID: id,
			})
		} else {
			err = h.Store.SetArtistRating(r.Context(), db.SetArtistRatingParams{
				ArtistID: id,
				UserName: user,
				Rating:   int32(rating),
			})
		}
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
				return
			}
		}
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	res := responses.New()
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/star/
func (h *Handler) handleStar(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	var ids []string
	ids = append(ids, query["id"]...)
	ids = append(ids, query["albumId"]...)
	ids = append(ids, query["artistId"]...)

	tx, err := h.Store.BeginTransaction(r.Context())
	if err != nil {
		log.Errorf("star: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer tx.Rollback(r.Context())
	for _, id := range ids {
		idType, ok := crossonic.GetIDType(id)
		if !ok {
			responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
			return
		}
		var err error
		switch idType {
		case crossonic.IDTypeSong:
			err = tx.StarSong(r.Context(), db.StarSongParams{
				SongID:   id,
				UserName: user,
			})
		case crossonic.IDTypeAlbum:
			err = tx.StarAlbum(r.Context(), db.StarAlbumParams{
				AlbumID:  id,
				UserName: user,
			})
		case crossonic.IDTypeArtist:
			err = tx.StarArtist(r.Context(), db.StarArtistParams{
				ArtistID: id,
				UserName: user,
			})
		}
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgErr.Code == pgerrcode.ForeignKeyViolation {
					responses.EncodeError(w, query.Get("f"), "not found", responses.SubsonicErrorNotFound)
					return
				}
			}
			log.Errorf("star: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	}

	err = tx.Commit(r.Context())
	if err != nil {
		log.Errorf("star: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	res := responses.New()
	res.EncodeOrLog(w, query.Get("f"))
}

// https://opensubsonic.netlify.app/docs/endpoints/unstar/
func (h *Handler) handleUnstar(w http.ResponseWriter, r *http.Request) {
	query := getQuery(r)
	user := query.Get("u")

	var ids []string
	ids = append(ids, query["id"]...)
	ids = append(ids, query["albumId"]...)
	ids = append(ids, query["artistId"]...)

	tx, err := h.Store.BeginTransaction(r.Context())
	if err != nil {
		log.Errorf("unstar: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}
	defer tx.Rollback(r.Context())
	for _, id := range ids {
		idType, ok := crossonic.GetIDType(id)
		if !ok {
			responses.EncodeError(w, query.Get("f"), "unknown id type", responses.SubsonicErrorNotFound)
			return
		}
		var err error
		switch idType {
		case crossonic.IDTypeSong:
			err = tx.UnstarSong(r.Context(), db.UnstarSongParams{
				SongID:   id,
				UserName: user,
			})
		case crossonic.IDTypeAlbum:
			err = tx.UnstarAlbum(r.Context(), db.UnstarAlbumParams{
				AlbumID:  id,
				UserName: user,
			})
		case crossonic.IDTypeArtist:
			err = tx.UnstarArtist(r.Context(), db.UnstarArtistParams{
				ArtistID: id,
				UserName: user,
			})
		}
		if err != nil {
			log.Errorf("unstar: %s", err)
			responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
			return
		}
	}

	err = tx.Commit(r.Context())
	if err != nil {
		log.Errorf("unstar: %s", err)
		responses.EncodeError(w, query.Get("f"), "internal server error", responses.SubsonicErrorGeneric)
		return
	}

	res := responses.New()
	res.EncodeOrLog(w, query.Get("f"))
}
