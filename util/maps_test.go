package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMapKeys(t *testing.T) {
	type testCase struct {
		name string
		m    map[string]any
		want []string
	}
	tests := []testCase{
		{"nil map should return an empty slice", nil, make([]string, 0)},
		{"empty map should return an empty slice", make(map[string]any), make([]string, 0)},
		{"map with one entry should return the key", map[string]any{"asdf": nil}, []string{"asdf"}},
		{"map with three entries should return the keys", map[string]any{"a": nil, "b": nil, "c": nil}, []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := MapKeys(tt.m)
			assert.NotNil(t, keys)
			for _, k := range keys {
				assert.Contains(t, tt.want, k)
			}
		})
	}
}
