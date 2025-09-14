package handlers

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/juho05/crossonic-server"
	"github.com/juho05/crossonic-server/handlers/responses"
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type UrlQuery struct {
	values         url.Values
	responseWriter http.ResponseWriter
}

func getQuery(w http.ResponseWriter, r *http.Request) UrlQuery {
	query, ok := r.Context().Value(ContextKeyQuery).(url.Values)
	if !ok {
		panic("getQuery must be called after subsonicMiddleware")
	}
	return UrlQuery{
		values:         query,
		responseWriter: w,
	}
}

func (q UrlQuery) Has(key string) bool {
	return q.values.Has(key)
}

func (q UrlQuery) Str(key string) string {
	return q.values.Get(key)
}

func (q UrlQuery) Strs(name string) []string {
	return q.values[name]
}
func (q UrlQuery) StrReq(name string) (string, bool) {
	v := q.Str(name)
	if v == "" {
		q.invalidParameter(name)
		return "", false
	}
	return v, true
}

func (q UrlQuery) StrsReq(name string) ([]string, bool) {
	v := q.Strs(name)
	if len(v) == 0 {
		q.missingParameter(name)
		return nil, false
	}
	return v, true
}

func (q UrlQuery) Bool(name string, def bool) (value bool, ok bool) {
	boolStr := q.Str(name)
	if boolStr == "" {
		return def, true
	}
	value, err := strconv.ParseBool(boolStr)
	if err != nil {
		q.invalidParameter(name)
		return false, false
	}
	return value, true
}

func (q UrlQuery) IDReq(name string) (string, bool) {
	id, ok := q.StrReq(name)
	if !ok {
		return "", false
	}
	if !crossonic.IDRegex.MatchString(id) {
		q.invalidParameter(name)
		return "", false
	}
	return id, true
}

func (q UrlQuery) IDTypeReq(name string, allowedIDTypes []crossonic.IDType) (string, bool) {
	id, ok := q.IDReq(name)
	if !ok {
		return "", false
	}
	idType, ok := crossonic.GetIDType(id)
	if !ok {
		q.invalidParameter(name)
		return "", false
	}
	if !slices.Contains(allowedIDTypes, idType) {
		q.invalidParameter(name)
		return "", false
	}
	return id, true
}

func (q UrlQuery) IDs(name string) ([]string, bool) {
	strs := q.Strs(name)
	for _, str := range strs {
		if !crossonic.IDRegex.MatchString(str) {
			q.invalidParameter(name)
			return nil, false
		}
	}
	return strs, true
}

func (q UrlQuery) IDsType(name string, allowedIDTypes []crossonic.IDType) ([]string, bool) {
	ids, ok := q.IDs(name)
	if !ok {
		return nil, false
	}
	for _, id := range ids {
		idType, ok := crossonic.GetIDType(id)
		if !ok {
			q.invalidParameter(name)
			return nil, false
		}
		if !slices.Contains(allowedIDTypes, idType) {
			q.invalidParameter(name)
			return nil, false
		}
	}
	return ids, true
}

func (q UrlQuery) IDsTypeReq(name string, allowedIDTypes []crossonic.IDType) ([]string, bool) {
	ids, ok := q.IDsType(name, allowedIDTypes)
	if !ok {
		return nil, false
	}
	if len(ids) == 0 {
		q.missingParameter(name)
		return nil, false
	}
	return ids, true
}

func (q UrlQuery) Int(name string) (*int, bool) {
	v := q.Str(name)
	if v == "" {
		return nil, true
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		q.invalidParameter(name)
		return nil, false
	}
	return &i, true
}

func (q UrlQuery) IntDef(name string, def int) (int, bool) {
	i, ok := q.Int(name)
	if !ok {
		return 0, false
	}
	if i == nil {
		return def, true
	}
	return *i, true
}

func (q UrlQuery) IntRange(name string, min, max int) (*int, bool) {
	i, ok := q.Int(name)
	if !ok {
		return nil, false
	}
	if i == nil {
		return nil, true
	}
	if *i < min || *i > max {
		q.invalidParameter(name)
		return nil, false
	}
	return i, true
}
func (q UrlQuery) IntRangeReq(name string, min, max int) (int, bool) {
	i, ok := q.IntRange(name, min, max)
	if !ok {
		return 0, false
	}
	if i == nil {
		q.invalidParameter(name)
		return 0, false
	}
	return *i, true
}

func (q UrlQuery) IntPositive(name string) (*int, bool) {
	return q.IntRange(name, 0, math.MaxInt)
}

func (q UrlQuery) IntPositiveDef(name string, def int) (int, bool) {
	i, ok := q.IntRange(name, 0, math.MaxInt)
	if !ok {
		return 0, false
	}
	if i == nil {
		return def, true
	}
	return *i, ok
}

func (q UrlQuery) IntPositiveReq(name string) (int, bool) {
	return q.IntRangeReq(name, 0, math.MaxInt)
}

func (q UrlQuery) Ints(name string) ([]int, bool) {
	strs := q.Strs(name)

	ints := make([]int, len(strs))
	for i, s := range strs {
		integer, err := strconv.Atoi(s)
		if err != nil {
			q.invalidParameter(name)
			return nil, false
		}
		ints[i] = integer
	}

	return ints, true
}

func (q UrlQuery) Int64(name string) (*int64, bool) {
	v := q.Str(name)
	if v == "" {
		return nil, true
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		q.invalidParameter(name)
		return nil, false
	}
	return &i, true
}

func (q UrlQuery) Int64s(name string) ([]int64, bool) {
	strs := q.Strs(name)

	ints := make([]int64, len(strs))
	for i, s := range strs {
		integer, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			q.invalidParameter(name)
			return nil, false
		}
		ints[i] = integer
	}

	return ints, true
}

func (q UrlQuery) Paginate(limitName, offsetName string, defaultLimit int) (repos.Paginate, bool) {
	limit, ok := q.IntRange(limitName, 0, maxListSize)
	if !ok {
		return repos.Paginate{}, false
	}
	if limit == nil {
		limit = &defaultLimit
	}

	offset, ok := q.IntPositiveDef(offsetName, 0)
	if !ok {
		return repos.Paginate{}, false
	}

	return repos.Paginate{
		Limit:  limit,
		Offset: offset,
	}, true
}

func (q UrlQuery) PaginateUnlimited(limitName, offsetName string) (paginate repos.Paginate, ok bool) {
	limit, ok := q.IntPositive(limitName)
	if !ok {
		return repos.Paginate{}, false
	}

	offset, ok := q.IntPositiveDef(offsetName, 0)
	if !ok {
		return repos.Paginate{}, false
	}

	return repos.Paginate{
		Limit:  limit,
		Offset: offset,
	}, true
}

func (q UrlQuery) TimeUnixMillis(name string) (*time.Time, bool) {
	i, ok := q.Int64(name)
	if !ok {
		return nil, false
	}
	if i == nil {
		return nil, true
	}
	return util.ToPtr(time.UnixMilli(*i)), true
}

func (q UrlQuery) Format() string {
	return q.Str("f")
}

func (q UrlQuery) User() string {
	return q.Str("u")
}

func (q UrlQuery) Client() string {
	return q.Str("c")
}

func (q UrlQuery) missingParameter(name string) {
	responses.EncodeError(q.responseWriter, q.Format(), fmt.Sprintf("missing %s parameter", name), responses.SubsonicErrorRequiredParameterMissing)
}

func (q UrlQuery) invalidParameter(name string) {
	responses.EncodeError(q.responseWriter, q.Format(), fmt.Sprintf("invalid %s parameter", name), responses.SubsonicErrorGeneric)
}
