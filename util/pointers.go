package util

// ToPtr returns a pointer pointing to a.
// The returned pointer is never nil.
func ToPtr[T any](a T) *T {
	return &a
}

// EqPtrVals returns true if both values the pointers point to are equal
// or if both pointers are nil.
func EqPtrVals[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
