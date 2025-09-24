package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/juho05/log"
	"github.com/nullism/bqb"
)

const execBatchSize = 512
const selectBatchSize = 4096

func execBatch[T any](data []T, fn func(data []T) error) error {
	if len(data) == 0 {
		return nil
	}
	for i := 0; i < len(data); i += execBatchSize {
		err := fn(data[i:min(i+execBatchSize, len(data))])
		if err != nil {
			return err
		}
	}
	return nil
}

func selectBatch[T, U any](data []T, fn func(data []T) ([]U, error)) ([]U, error) {
	if len(data) == 0 {
		return make([]U, 0), nil
	}
	var result []U
	for i := 0; i < len(data); i += selectBatchSize {
		slice, err := fn(data[i:min(i+selectBatchSize, len(data))])
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = slice
		} else {
			result = append(result, slice...)
		}
	}
	return result, nil
}

func selectBatch2[T, U, V any](data1 []T, data2 []U, fn func(data1 []T, data2 []U) ([]V, error)) ([]V, error) {
	if len(data1) == 0 && len(data2) == 0 {
		return make([]V, 0), nil
	}
	var result []V
	for i := 0; i < max(len(data1), len(data2)); i += selectBatchSize {
		d1 := make([]T, 0)
		if i < len(data1) {
			d1 = data1[i:min(i+selectBatchSize, len(data1))]
		}
		d2 := make([]U, 0)
		if i < len(data2) {
			d2 = data2[i:min(i+selectBatchSize, len(data2))]
		}

		slice, err := fn(d1, d2)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = slice
		} else {
			result = append(result, slice...)
		}
	}
	return result, nil
}

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

	result := make([]T, 0)
	err = db.SelectContext(ctx, &result, sql, args...)
	printQueryOnErr(sql, err)
	return result, wrapErr("execute select query", err)
}

func printQueryOnErr(query string, err error) {
	if err == nil {
		return
	}
	if errors.Is(sqlErrToErrType(err), repos.ErrGeneral) && !errors.Is(err, context.Canceled) {
		log.Errorf("error on query: %s: %s", query, err)
	}
}

func genUpdateList(values map[string]repos.OptionalGetter, updatedField bool) (query *bqb.Query, empty bool) {
	q := bqb.Optional("")
	empty = true
	for name, value := range values {
		if value.HasValue() {
			empty = false
			q.Comma(fmt.Sprintf("%s=?", name), value.Get())
		}
	}
	if updatedField {
		q.Comma("updated=NOW()")
		empty = false
	}
	return q, empty
}

func genSearch(query, searchColumn, titleCol string) (conditions *bqb.Query, orderBy *bqb.Query) {
	conditions = bqb.New("true")
	orderBy = bqb.Optional("")
	searchTokens := strings.Split(util.NormalizeText(query), " ")
	tokenCount := 0
	for _, token := range searchTokens {
		if token == "" || token == " " {
			continue
		}
		token = " " + token
		conditions.And(fmt.Sprintf("position(? in %s) > 0", searchColumn), token)
		if tokenCount < 3 {
			orderBy.Comma(fmt.Sprintf("position(? in %s)", searchColumn), token)
		}
		tokenCount++
	}
	orderBy.Comma(fmt.Sprintf("lower(%s)", titleCol))
	return conditions, orderBy
}
