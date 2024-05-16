package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaevor/go-nanoid"
	"github.com/juho05/crossonic-server"
)

type Store interface {
	Querier
	BeginTransaction(ctx context.Context) (Store, error)
	InstanceID() string
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	FindOrCreateArtistIDs(ctx context.Context, names []string) ([]string, error)
	UpdateAlbumArtists(ctx context.Context, albumID string, artistIDs []string) error
	UpdateAlbumGenres(ctx context.Context, albumID string, genres []string) error
	UpdateSongArtists(ctx context.Context, songID string, artistIDs []string) error
	UpdateSongGenres(ctx context.Context, songID string, genres []string) error
}

type store struct {
	*Queries
	db         *pgxpool.Pool
	instanceID string
}

type transaction struct {
	*Queries
	tx         pgx.Tx
	instanceID string
}

func NewStore(db *pgxpool.Pool) (Store, error) {
	store := &store{
		db:      db,
		Queries: New(db),
	}
	genInstanceID, err := nanoid.CustomUnicode("abcdefghijklmnopqrstuvwxyz0123456789", 16)
	if err != nil {
		return nil, fmt.Errorf("generate instance ID: %w", err)
	}
	instanceID, err := store.InsertSystemValueIfNotExists(context.Background(), InsertSystemValueIfNotExistsParams{
		Key:   "instance_id",
		Value: genInstanceID(),
	})
	if err != nil {
		return nil, fmt.Errorf("new store: create/load instance ID: %w", err)
	}
	store.instanceID = instanceID.Value
	return store, nil
}

func (s *store) BeginTransaction(ctx context.Context) (Store, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	return &transaction{
		Queries:    s.Queries.WithTx(tx),
		tx:         tx,
		instanceID: s.instanceID,
	}, nil
}

func (s *store) InstanceID() string {
	return s.instanceID
}

func (s *store) Commit(ctx context.Context) error {
	return errors.New("store is not a transaction")
}

func (s *store) Rollback(ctx context.Context) error {
	return errors.New("store is not a transaction")
}

func (s *store) FindOrCreateArtistIDs(ctx context.Context, names []string) ([]string, error) {
	return findOrCreateArtistIDs(s, ctx, names)
}

func (s *store) UpdateAlbumArtists(ctx context.Context, albumID string, artistIDs []string) error {
	return updateAlbumArtists(s, ctx, albumID, artistIDs)
}

func (s *store) UpdateAlbumGenres(ctx context.Context, albumID string, genres []string) error {
	return updateAlbumGenres(s, ctx, albumID, genres)
}

func (s *store) UpdateSongArtists(ctx context.Context, songID string, artistIDs []string) error {
	return updateSongArtists(s, ctx, songID, artistIDs)
}

func (s *store) UpdateSongGenres(ctx context.Context, songID string, genres []string) error {
	return updateSongGenres(s, ctx, songID, genres)
}

func (s *transaction) BeginTransaction(ctx context.Context) (Store, error) {
	tx, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	return &transaction{
		Queries: s.Queries.WithTx(tx),
		tx:      tx,
	}, nil
}

func (s *transaction) InstanceID() string {
	return s.instanceID
}

func (s *transaction) Commit(ctx context.Context) error {
	return s.tx.Commit(ctx)
}

func (s *transaction) Rollback(ctx context.Context) error {
	return s.tx.Rollback(ctx)
}

func (s *transaction) FindOrCreateArtistIDs(ctx context.Context, names []string) ([]string, error) {
	return findOrCreateArtistIDs(s, ctx, names)
}

func (s *transaction) UpdateAlbumArtists(ctx context.Context, albumID string, artistIDs []string) error {
	return updateAlbumArtists(s, ctx, albumID, artistIDs)
}

func (s *transaction) UpdateAlbumGenres(ctx context.Context, albumID string, genres []string) error {
	return updateAlbumGenres(s, ctx, albumID, genres)
}

func (s *transaction) UpdateSongArtists(ctx context.Context, songID string, artistIDs []string) error {
	return updateSongArtists(s, ctx, songID, artistIDs)
}

func (s *transaction) UpdateSongGenres(ctx context.Context, songID string, genres []string) error {
	return updateSongGenres(s, ctx, songID, genres)
}

func findOrCreateArtistIDs(db Store, ctx context.Context, names []string) ([]string, error) {
	tx, err := db.BeginTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("find or create artist ids: %w", err)
	}
	defer tx.Rollback(ctx)

	artists, err := tx.FindArtistsByName(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("find or create artist ids: %w", err)
	}

	ids := make([]string, len(names))
	for i := range names {
		var foundID string
		for _, a := range artists {
			if a.Name == names[i] {
				foundID = a.ID
				break
			}
		}
		if foundID != "" {
			ids[i] = foundID
			continue
		}
		a, err := tx.CreateArtist(ctx, CreateArtistParams{
			ID:   "ar_" + crossonic.GenID(),
			Name: names[i],
		})
		if err != nil {
			return nil, fmt.Errorf("find or create artist ids: %w", err)
		}
		ids[i] = a.ID
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("find or create artist ids: %w", err)
	}
	return ids, nil
}

func updateAlbumArtists(db Store, ctx context.Context, albumID string, artistIDs []string) error {
	tx, err := db.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("update album artists: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.DeleteAlbumArtists(ctx, albumID)
	if err != nil {
		return fmt.Errorf("update album artists: delete old connections: %w", err)
	}

	args := make([]CreateAlbumArtistsParams, len(artistIDs))
	for i := range artistIDs {
		args[i] = CreateAlbumArtistsParams{
			ArtistID: artistIDs[i],
			AlbumID:  albumID,
		}
	}
	_, err = tx.CreateAlbumArtists(ctx, args)
	if err != nil {
		return fmt.Errorf("update album artists: create new connections: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("update album artists: %w", err)
	}
	return nil
}

func updateAlbumGenres(db Store, ctx context.Context, albumID string, genres []string) error {
	tx, err := db.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("update album genres: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.DeleteAlbumGenres(ctx, albumID)
	if err != nil {
		return fmt.Errorf("update album genres: delete old connections: %w", err)
	}

	for _, g := range genres {
		err = tx.CreateGenre(ctx, g)
		if err != nil {
			return fmt.Errorf("update song genres: create genre: %w", err)
		}
	}

	args := make([]CreateAlbumGenresParams, len(genres))
	for i := range genres {
		args[i] = CreateAlbumGenresParams{
			GenreName: genres[i],
			AlbumID:   albumID,
		}
	}
	_, err = tx.CreateAlbumGenres(ctx, args)
	if err != nil {
		return fmt.Errorf("update album genres: create new connections: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("update album genres: %w", err)
	}
	return nil
}

func updateSongArtists(db Store, ctx context.Context, songID string, artistIDs []string) error {
	tx, err := db.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("update song artists: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.DeleteSongArtists(ctx, songID)
	if err != nil {
		return fmt.Errorf("update song artists: delete old connections: %w", err)
	}

	args := make([]CreateSongArtistsParams, len(artistIDs))
	for i := range artistIDs {
		args[i] = CreateSongArtistsParams{
			ArtistID: artistIDs[i],
			SongID:   songID,
		}
	}
	_, err = tx.CreateSongArtists(ctx, args)
	if err != nil {
		return fmt.Errorf("update song artists: create new connections: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("update song artists: %w", err)
	}
	return nil
}

func updateSongGenres(db Store, ctx context.Context, songID string, genres []string) error {
	tx, err := db.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("update song genres: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.DeleteSongGenres(ctx, songID)
	if err != nil {
		return fmt.Errorf("update song genres: delete old connections: %w", err)
	}

	for _, g := range genres {
		err = tx.CreateGenre(ctx, g)
		if err != nil {
			return fmt.Errorf("update song genres: create genre: %w", err)
		}
	}

	args := make([]CreateSongGenresParams, len(genres))
	for i := range genres {
		args[i] = CreateSongGenresParams{
			GenreName: genres[i],
			SongID:    songID,
		}
	}
	_, err = tx.CreateSongGenres(ctx, args)
	if err != nil {
		return fmt.Errorf("update song genres: create new connections: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("update song genres: %w", err)
	}
	return nil
}
