package db

import "github.com/jackc/pgx/v5"

type Store interface {
	Querier
}

type store struct {
	*Queries
	db *pgx.Conn
}

func NewStore(db *pgx.Conn) Store {
	return &store{
		db:      db,
		Queries: New(db),
	}
}
