package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/log"
	"github.com/nullism/bqb"
)

func executeQuery(ctx context.Context, db executer, query *bqb.Query) error {
	sql, args, err := query.ToPgsql()
	if err != nil {
		return wrapErr("build query", err)
	}
	_, err = db.ExecContext(ctx, sql, args...)
	printQueryOnErr(sql, err)
	return wrapErr("execute exec query", err)
}

func executeQueryCountAffectedRows(ctx context.Context, db executer, query *bqb.Query) (int, error) {
	sql, args, err := query.ToPgsql()
	if err != nil {
		return 0, wrapErr("build query", err)
	}
	res, err := db.ExecContext(ctx, sql, args...)
	printQueryOnErr(sql, err)
	var count int64
	if err == nil {
		count, _ = res.RowsAffected()
	}
	return int(count), wrapErr("execute exec query", err)
}

func executeQueryExpectAffectedRows(ctx context.Context, db executer, query *bqb.Query) error {
	sql, args, err := query.ToPgsql()
	if err != nil {
		return wrapErr("build query", err)
	}

	res, err := db.ExecContext(ctx, sql, args...)
	printQueryOnErr(sql, err)
	return wrapResErr("execute exec query (expect affected rows)", res, err)
}

func getQuery[T any](ctx context.Context, db executer, query *bqb.Query) (T, error) {
	var result T
	results, err := selectQuery[T](ctx, db, query)
	if err != nil {
		return result, err
	}
	if len(results) == 0 {
		return result, repos.NewError("", repos.ErrNotFound, nil)
	}
	if len(results) > 1 {
		return result, repos.NewError("", repos.ErrTooMany, nil)
	}
	result = results[0]
	return result, nil
}

func selectQuery[T any](ctx context.Context, db executer, query *bqb.Query) ([]T, error) {
	sql, args, err := query.ToPgsql()
	if err != nil {
		return nil, wrapErr("build query", err)
	}

	result := []T{}
	err = db.SelectContext(ctx, &result, sql, args...)
	printQueryOnErr(sql, err)
	return result, wrapErr("execute select query", err)
}

func printQueryOnErr(query string, err error) {
	if err == nil {
		return
	}
	if sqlErrToErrType(err) == repos.ErrGeneral && !errors.Is(err, context.Canceled) {
		log.Errorf("error on query: %s: %s", query, err)
	}
}

func genUpdateList(values map[string]repos.OptionalGetter, updatedField bool) *bqb.Query {
	q := bqb.Optional("")
	for name, value := range values {
		if value.HasValue() {
			q.Comma(fmt.Sprintf("%s=?", name), value.Get())
		}
	}
	if updatedField {
		q.Comma("updated=NOW()")
	}
	return q
}
