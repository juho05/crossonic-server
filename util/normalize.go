package util

import (
	"github.com/mozillazg/go-unidecode"
	"unicode"
)

// NormalizeText converts text into a form that is suitable for comparison with user input.
// It removes accents/diacritics, converts neighboring space characters to a single space,
// converts text into lowercase and removes all non-letter/non-digit characters.
// Additionally, the text is transliterated into ASCII.
func NormalizeText(text string) string {
	ascii := unidecode.Unidecode(text)
	result := make([]rune, 0, len(ascii))
	for _, r := range ascii {
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
		result = append(result, r)
	}
	return string(result)
}
