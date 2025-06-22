package util

import (
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var replacementTable = map[rune][]rune{
	'ß': {'s', 's'},
}

func NormalizeText(text string) string {
	nfd := norm.NFD.String(text)
	result := make([]rune, 0, len(text))
	for _, r := range nfd {
		// replace all space characters with ' '
		if unicode.IsSpace(r) {
			if len(result) == 0 || result[len(result)-1] != ' ' {
				result = append(result, ' ')
			}
			continue
		}
		// discard non letter/digit characters
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			continue
		}
		r = unicode.ToLower(r)
		// apply replacements
		if rep, ok := replacementTable[r]; ok {
			result = append(result, rep...)
		} else {
			result = append(result, r)
		}
	}
	return norm.NFC.String(string(result))
}
