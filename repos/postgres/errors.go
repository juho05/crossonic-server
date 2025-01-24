package postgres

import (
	"database/sql"
	"errors"

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
	return repos.ErrGeneral
}
