package util

// Map calls mapFn for every element in s and returns a slice
// with each element in s replaced by the value returned by the corresponding call to mapFn.
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

// FirstOrNil returns the first element in s or nil if s is nil or empty.
func FirstOrNil[T any](s []T) *T {
	if len(s) > 0 {
		return &s[0]
	}
	return nil
}

// FirstOrNilMap returns the result of mapFn when passed the first element in s or nil
// if s is empty or nil.
func FirstOrNilMap[T, U any](s []T, mapFn func(T) U) *U {
	if len(s) > 0 {
		res := mapFn(s[0])
		return &res
	}
	return nil
}
