package crossonic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIDRegex(t *testing.T) {
	valid := []string{
		"tr_abcdefghijkl",
		"al_abcdefghijkl",
		"ar_abcdefghijkl",
		"pl_abcdefghijkl",
		"irs_abcdefghijkl",
		// all alphabet characters
		"tr_ABCDEFGHIJKL",
		"tr_0123456789-~",
		"tr_aB3-~aB3-~aB",
	}
	for _, id := range valid {
		assert.True(t, IDRegex.MatchString(id))
	}

	invalid := []string{
		// path traversal
		"tr_../../../etc/passwd",
		"tr/../../../../etc/passwd",
		// prefix without id
		"tr",
		"al",
		"ar",
		"pl",
		// valid id as substring in larger invalid string
		"hello_al_world",
		"normaltext", // contains "ar"
		// wrong prefix
		"xx_abcdefghijkl",
		// missing separator
		"trabcdefghijklm",
		// body too short / too long
		"tr_abcdefghijk",
		"tr_abcdefghijklm",
		// empty string
		"",
		// valid prefix but invalid body characters
		"tr_abcdefghij!@",
		"tr_abcdefghij  ",
		// wrong prefix spelling for irs
		"ir_abcdefghijkl",
	}
	for _, id := range invalid {
		assert.False(t, IDRegex.MatchString(id))
	}
}
