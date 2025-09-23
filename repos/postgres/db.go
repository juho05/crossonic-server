package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/config"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
	migrate "github.com/rubenv/sql-migrate"
)

type DB struct {
	db     *sqlx.DB
	tx     *sqlx.Tx
	config config.Config
}

func NewDB(dsn string, conf config.Config) (*DB, error) {
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: postgres: %w", err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("open db: postgres: %w", err)
	}

	if conf.AutoMigrate {
		err = autoMigrate(db.DB)
		if err != nil {
			return nil, fmt.Errorf("open db: postgres: %w", err)
		}
	}

	return &DB{
		db:     db,
		config: conf,
	}, nil
}

func autoMigrate(db *sql.DB) error {
	migrations := &migrate.HttpFileSystemMigrationSource{
		FileSystem: http.FS(crossonic.MigrationsFS),
	}
	log.Trace("Migrating database...")
	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	log.Tracef("Applied %d migrations!", n)
	return nil
}

func (d *DB) User() repos.UserRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return userRepository{
		db: exec,
		tx: newTransactionFn(d, func(tx executer) userRepository {
			return userRepository{
				db:   tx,
				conf: d.config,
			}
		}),
		conf: d.config,
	}
}

func (d *DB) System() repos.SystemRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return systemRepository{
		db: exec,
		tx: newTransactionFn(d, func(tx executer) systemRepository {
			return systemRepository{
				db: tx,
			}
		}),
	}
}

func (d *DB) Song() repos.SongRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return songRepository{
		db: exec,
		tx: newTransactionFn(d, func(tx executer) songRepository {
			return songRepository{
				db: tx,
			}
		}),
	}
}

func (d *DB) Scrobble() repos.ScrobbleRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return scrobbleRepository{
		db: exec,
		tx: newTransactionFn(d, func(tx executer) scrobbleRepository {
			return scrobbleRepository{
				db: tx,
			}
		}),
	}
}

func (d *DB) Album() repos.AlbumRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return albumRepository{
		db: exec,
		tx: newTransactionFn(d, func(tx executer) albumRepository {
			return albumRepository{
				db: tx,
			}
		}),
	}
}

func (d *DB) Artist() repos.ArtistRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return artistRepository{
		db: exec,
		tx: newTransactionFn(d, func(tx executer) artistRepository {
			return artistRepository{
				db: tx,
			}
		}),
	}
}

func (d *DB) Genre() repos.GenreRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return genreRepository{
		db: exec,
		tx: newTransactionFn(d, func(tx executer) genreRepository {
			return genreRepository{
				db: tx,
			}
		}),
	}
}

func (d *DB) Playlist() repos.PlaylistRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return playlistRepository{
		db: exec,
		tx: newTransactionFn(d, func(tx executer) playlistRepository {
			return playlistRepository{
				db: tx,
			}
		}),
	}
}

func (d *DB) InternetRadioStation() repos.InternetRadioStationRepository {
	exec := executer(d.db)
	if d.tx != nil {
		exec = d.tx
	}
	return internetRadioStationRepository{
		db: exec,
	}
}

func (d *DB) Transaction(ctx context.Context, fn func(tx repos.Tx) error) error {
	if d.db == nil {
		return repos.NewError("create transaction", repos.ErrNestedTransaction, nil)
	}
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapErr("begin transaction", err)
	}
	defer func() {
		err = tx.Rollback()
		if err != nil {
			if errors.Is(err, sql.ErrTxDone) {
				return
			}
			log.Errorf("rollback transaction: %s", err)
		}
	}()
	err = fn(&DB{
		tx: tx,
	})
	if err != nil {
		return err
	}
	return wrapErr("commit transaction", tx.Commit())
}

func newTransactionFn[R any](db *DB, newRepo func(tx executer) R) func(ctx context.Context, fn func(R) error) error {
	return func(ctx context.Context, fn func(R) error) error {
		if db.tx != nil {
			return fn(newRepo(db.tx))
		}
		tx, err := db.db.BeginTxx(ctx, nil)
		if err != nil {
			return wrapErr("", fmt.Errorf("begin transaction: %w", err))
		}
		defer func() {
			err = tx.Rollback()
			if err != nil {
				if errors.Is(err, sql.ErrTxDone) {
					return
				}
				log.Errorf("rollback transaction: %s", err)
			}
		}()
		err = fn(newRepo(tx))
		if err != nil {
			return wrapErr("", err)
		}
		err = tx.Commit()
		if err != nil {
			return wrapErr("", fmt.Errorf("commit transaction: %w", err))
		}
		return nil
	}
}

func (d *DB) NewTransaction(ctx context.Context) (repos.Transaction, error) {
	if d.db == nil {
		return nil, repos.NewError("create transaction", repos.ErrNestedTransaction, nil)
	}
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, wrapErr("begin transaction", err)
	}
	return &DB{
		tx: tx,
	}, nil
}

func (d *DB) Commit() error {
	if d.tx != nil {
		return d.tx.Commit()
	}
	return nil
}

func (d *DB) Rollback() error {
	if d.tx != nil {
		err := d.tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			return err
		}
	}
	return nil
}

func (d *DB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
