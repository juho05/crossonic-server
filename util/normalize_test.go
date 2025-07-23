package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"the empty string stays the empty string", "", ""},
		{"whitespace is correctly normalized", "  asdf\t  test   bla\r\n", " asdf test bla "},
		{"text is converted to lowercase", "AaBbCcDd", "aabbccdd"},
		{"accents are removed", "öäüàêÇ", "oauaec"},
		{"ß is handled", "Soße auf der STRAẞE", "sosse auf der strasse"},
		{"special characters are removed", "Hello, world!", "hello world"},
		{"cfk characters are correctly normalized", "안녕하세요 세상아", "annyeonghaseyo sesanga"},
		{"Ænima", "Ænima", "aenima"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := NormalizeText(tt.text)
			assert.Equalf(t, tt.want, normalized, "normalized bytes: %v, wanted: %v", []byte(normalized), []byte(tt.want))
		})
	}
}
