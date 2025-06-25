package util

import (
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var replacementTable = map[rune][]rune{
	'ß': {'s', 's'},
}

// NormalizeText converts text into a form that is suitable for comparison with user input.
// It removes accents/diacritics, converts neighboring space characters to a single space,
// converts text into lowercase and removes all non-letter/non-digit characters.
// Additionally, some characters are replaced according to the following table:
//   - ß -> ss
//
// The result is returned in Unicode NFKD form.
func NormalizeText(text string) string {
	nfd := norm.NFKD.String(text)
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
	return string(result)
}
