package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestNilIfEmpty(t *testing.T) {
	t.Run("empty string should return nil", func(t *testing.T) {
		assert.Nil(t, NilIfEmpty(""))
	})
	t.Run("0 should return nil", func(t *testing.T) {
		assert.Nil(t, NilIfEmpty(0))
	})
	t.Run("0.0 should return nil", func(t *testing.T) {
		assert.Nil(t, NilIfEmpty(0.0))
	})
	t.Run("false should return nil", func(t *testing.T) {
		assert.Nil(t, NilIfEmpty(false))
	})
	t.Run("struct with default values should return nil", func(t *testing.T) {
		assert.Nil(t, NilIfEmpty(struct {
			A string
			B int
			C bool
		}{}))
	})

	t.Run("\"x\" should return ptr", func(t *testing.T) {
		result := NilIfEmpty("x")
		assert.NotNil(t, result)
		assert.Equal(t, "x", *result)
	})
	t.Run("1 should return ptr", func(t *testing.T) {
		result := NilIfEmpty(1)
		assert.NotNil(t, result)
		assert.Equal(t, 1, *result)
	})
	t.Run("0.125 should return ptr", func(t *testing.T) {
		result := NilIfEmpty(0.125)
		assert.NotNil(t, result)
		assert.Equal(t, 0.125, *result)
	})
	t.Run("true should return ptr", func(t *testing.T) {
		result := NilIfEmpty(true)
		assert.NotNil(t, result)
		assert.Equal(t, true, *result)
	})
	t.Run("struct with non-default values should return ptr", func(t *testing.T) {
		value := struct {
			A string
			B int
			C bool
		}{
			B: 1,
		}
		result := NilIfEmpty(value)
		assert.NotNil(t, result)
		assert.Equal(t, value, *result)
	})
}
