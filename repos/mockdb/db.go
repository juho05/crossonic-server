package mockdb

import (
	"context"
	"github.com/juho05/crossonic-server/repos"
)

type DB struct {
	UserRepository                 UserRepository
	SystemRepository               SystemRepository
	SongRepository                 SongRepository
	ScrobbleRepository             ScrobbleRepository
	AlbumRepository                AlbumRepository
	ArtistRepository               ArtistRepository
	GenreRepository                GenreRepository
	PlaylistRepository             PlaylistRepository
	InternetRadioStationRepository InternetRadioStationRepository

	TransactionMock    func(ctx context.Context, fn func(tx repos.Tx) error) error
	NewTransactionMock func(ctx context.Context) (repos.Transaction, error)
	CommitMock         func() error
	RollbackMock       func() error
	CloseMock          func() error
}

func (d *DB) User() repos.UserRepository {
	return d.UserRepository
}

func (d *DB) System() repos.SystemRepository {
	return d.SystemRepository
}

func (d *DB) Song() repos.SongRepository {
	return d.SongRepository
}

func (d *DB) Scrobble() repos.ScrobbleRepository {
	return d.ScrobbleRepository
}

func (d *DB) Album() repos.AlbumRepository {
	return d.AlbumRepository
}

func (d *DB) Artist() repos.ArtistRepository {
	return d.ArtistRepository
}

func (d *DB) Genre() repos.GenreRepository {
	return d.GenreRepository
}

func (d *DB) Playlist() repos.PlaylistRepository {
	return d.PlaylistRepository
}

func (d *DB) InternetRadioStation() repos.InternetRadioStationRepository {
	return d.InternetRadioStationRepository
}

func (d *DB) Transaction(ctx context.Context, fn func(tx repos.Tx) error) error {
	if d.TransactionMock != nil {
		return d.TransactionMock(ctx, fn)
	}
	return fn(d)
}

func (d *DB) NewTransaction(ctx context.Context) (repos.Transaction, error) {
	if d.NewTransactionMock != nil {
		return d.NewTransactionMock(ctx)
	}
	return d, nil
}

func (d *DB) Commit() error {
	if d.CommitMock != nil {
		return d.CommitMock()
	}
	return nil
}

func (d *DB) Rollback() error {
	if d.RollbackMock != nil {
		return d.RollbackMock()
	}
	return nil
}

func (d *DB) Close() error {
	if d.CloseMock != nil {
		return d.CloseMock()
	}
	return nil
}
