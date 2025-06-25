package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToPtr(t *testing.T) {
	type testCase struct {
		name  string
		value int
	}
	tests := []testCase{
		{"0 should result in a pointer to 0", 0},
		{"-1 should result in a pointer to -1", -1},
		{"2 should result in a pointer to 2", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr := ToPtr(tt.value)
			assert.NotNil(t, ptr)
			assert.Equal(t, tt.value, *ptr)
		})
	}
}

func TestEqPtrVals(t *testing.T) {
	type testCase struct {
		name string
		a    *int
		b    *int
		want bool
	}

	tests := []testCase{
		{"nil should be equal to nil", nil, nil, true},
		{"0 should not be equal to nil", ToPtr(1), nil, false},
		{"nil should not be equal to 0", nil, ToPtr(1), false},
		{"0 should not be equal to 1", ToPtr(0), ToPtr(1), false},
		{"0 should be equal to 0", ToPtr(0), ToPtr(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EqPtrVals(tt.a, tt.b))
		})
	}
}
