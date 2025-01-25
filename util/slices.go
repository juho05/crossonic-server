package util

func Map[T, U any](s []T, mapFn func(T) U) []U {
	if s == nil {
		return nil
	}
	newSlice := make([]U, len(s))
	for i := range s {
		newSlice[i] = mapFn(s[i])
	}
	return newSlice
}

func FirstOrNil[T any](s []T) *T {
	if len(s) > 0 {
		return &s[0]
	}
	return nil
}

func FirstOrNilMap[T, U any](s []T, mapFn func(T) U) *U {
	if len(s) > 0 {
		res := mapFn(s[0])
		return &res
	}
	return nil
}
