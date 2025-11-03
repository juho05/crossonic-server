package postgres

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/juho05/crossonic-server/repos"
)

func wrapErr(message string, err error) error {
	if err == nil {
		return nil
	}
	return repos.NewError(message, sqlErrToErrType(err), err)
}

func wrapResErr(message string, result sql.Result, err error) error {
	if err == nil {
		if rows, err2 := result.RowsAffected(); err2 == nil && rows == 0 {
			return repos.NewError(message, repos.ErrNotFound, errors.New("no rows affected"))
		}
		return nil
	}
	return repos.NewError(message, sqlErrToErrType(err), err)
}

func sqlErrToErrType(err error) repos.ErrorType {
	var errType repos.ErrorType
	if errors.As(err, &errType) {
		return errType
	}
	if errors.Is(err, sql.ErrNoRows) {
		return repos.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if strings.Contains(pgErr.ConstraintName, "_pkey") || strings.Contains(pgErr.ConstraintName, "_key") {
			return repos.ErrExists
		}
	}
	return repos.ErrGeneral
}
