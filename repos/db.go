package repos

import "context"

type Tx interface {
	User() UserRepository
	System() SystemRepository
	Song() SongRepository
	Scrobble() ScrobbleRepository
	Album() AlbumRepository
	Artist() ArtistRepository
	Genre() GenreRepository
	Playlist() PlaylistRepository
	InternetRadioStation() InternetRadioStationRepository
}

type Transaction interface {
	Tx
	Commit() error
	Rollback() error
}

type DB interface {
	Tx
	Transaction(ctx context.Context, fn func(tx Tx) error) error

	NewTransaction(ctx context.Context) (Transaction, error)

	Close() error
}
