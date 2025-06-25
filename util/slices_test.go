package util

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestMap(t *testing.T) {
	type testCase struct {
		name string
		s    []int
		want []string
	}
	tests := []testCase{
		{"nil slice should result in a nil slice", nil, nil},
		{"empty slice should result in an empty slice", make([]int, 0), make([]string, 0)},
		{"should map slice with one element", []int{42}, []string{"42"}},
		{"should map slice with five element", []int{1, 2, 3, 4, 5}, []string{"1", "2", "3", "4", "5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Map(tt.s, func(t int) string {
				return strconv.Itoa(t)
			}))
		})
	}
}

func TestFirstOrNil(t *testing.T) {
	type testCase struct {
		name string
		s    []int
		want *int
	}
	tests := []testCase{
		{"nil slice should return nil", nil, nil},
		{"empty slice should return nil", make([]int, 0), nil},
		{"slice with one element should return that element", []int{1}, ToPtr(1)},
		{"slice with three elements should return the first element", []int{3, 2, 1}, ToPtr(3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FirstOrNil(tt.s))
		})
	}
}

func TestFirstOrNilMap(t *testing.T) {
	type testCase struct {
		name string
		s    []int
		want *string
	}
	tests := []testCase{
		{"nil slice should return nil", nil, nil},
		{"empty slice should return nil", make([]int, 0), nil},
		{"slice with one element should return that element", []int{1}, ToPtr("1")},
		{"slice with three elements should return the first element", []int{3, 2, 1}, ToPtr("3")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FirstOrNilMap(tt.s, func(t int) string {
				return strconv.Itoa(t)
			}))
		})
	}
}
