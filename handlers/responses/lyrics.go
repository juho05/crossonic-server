package responses

import (
	"bufio"
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/juho05/log"
)

type Lyrics struct {
	Title  string  `xml:"title,attr" json:"title"`
	Artist *string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Value  string  `xml:",chardata" json:"value"`
}

type LyricsList struct {
	StructuredLyrics []*StructuredLyrics `xml:"structuredLyrics" json:"structuredLyrics"`
}

type StructuredLyrics struct {
	Lang          string  `xml:"lang,attr" json:"lang"`
	Synced        bool    `xml:"synced,attr" json:"synced"`
	DisplayArtist string  `xml:"displayArtist,attr,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string  `xml:"displayTitle,attr,omitempty" json:"displayTitle,omitempty"`
	Offset        int     `xml:"offset,attr,omitempty" json:"offset,omitempty"`
	Line          []*Line `xml:"line" json:"line"`
}

type Line struct {
	Value string `xml:",chardata" json:"value"`
	Start *int   `xml:"start,attr,omitempty" json:"start,omitempty"`
}

func NewLyrics(lyrics *string) *string {
	if lyrics == nil {
		return nil
	}
	result := stripLRCMetadata(*lyrics)
	if result == "" {
		return nil
	}
	return &result
}

func NewLyricsList(lyrics *string) *LyricsList {
	if lyrics == nil {
		return &LyricsList{
			StructuredLyrics: make([]*StructuredLyrics, 0),
		}
	}

	structuredLyrics := parseStructuredLyrics(*lyrics)
	if structuredLyrics == nil || len(structuredLyrics.Line) == 0 {
		return &LyricsList{
			StructuredLyrics: make([]*StructuredLyrics, 0),
		}
	}

	return &LyricsList{
		StructuredLyrics: []*StructuredLyrics{
			structuredLyrics,
		},
	}
}

func parseStructuredLyrics(lyrics string) *StructuredLyrics {
	lines := make([]*Line, 0, 50)
	lang := "und"
	synced := false
	artist := ""
	title := ""
	offset := 0

	scanner := bufio.NewScanner(strings.NewReader(lyrics))
	for scanner.Scan() {
		initialLine := scanner.Text()

		var builder strings.Builder
		var start *int

		textStarted := false

		readingTag := false
		var tagContentBuilder strings.Builder
		for _, r := range initialLine {
			if readingTag {
				if r == ']' {
					readingTag = false
					continue
				}
				tagContentBuilder.WriteRune(r)
				continue
			}

			if !textStarted && unicode.IsSpace(r) {
				continue
			}

			if r == '[' {
				readingTag = true
				continue
			}

			textStarted = true
			builder.WriteRune(r)
		}

		tagContent := tagContentBuilder.String()
		if tagContent != "" {
			if slices.Contains([]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}, tagContent[0]) {
				start = parseLRCTimeTag(tagContent)
				if start != nil {
					synced = true
				}
			} else {
				parts := strings.Split(tagContent, ":")
				if len(parts) > 1 {
					tag := strings.ToLower(parts[0])
					content := strings.TrimSpace(strings.Join(parts[1:], ":"))
					switch tag {
					case "ti":
						title = content
					case "ar":
						artist = content
					case "offset":
						var err error
						offset, err = strconv.Atoi(content)
						log.Warnf("invalid offset in lyrics data: %s", err)
					}
				}
			}
		}

		lineText := strings.TrimSpace(builder.String())

		if lineText == "" && start == nil && (len(lines) == 0 || lines[len(lines)-1].Value == "") {
			continue
		}

		lines = append(lines, &Line{
			Value: lineText,
			Start: start,
		})
	}

	if synced {
		slices.SortFunc(lines, func(a, b *Line) int {
			if a == nil || b == nil || a.Start == nil || b.Start == nil {
				return 0
			}
			return cmp.Compare(*a.Start, *b.Start)
		})
	}

	return &StructuredLyrics{
		Lang:          lang,
		Synced:        synced,
		DisplayArtist: artist,
		DisplayTitle:  title,
		Offset:        offset,
		Line:          lines,
	}
}

func parseLRCTimeTag(tag string) *int {
	dotParts := strings.Split(tag, ".")
	if len(dotParts) > 2 {
		log.Warnf("invalid lyrics line time: %s", tag)
		return nil
	}

	var result int
	if len(dotParts) == 2 {
		if len(dotParts[1]) > 3 {
			dotParts[1] = dotParts[1][:3]
		}
		if len(dotParts[1]) == 0 {
			dotParts[1] = "0"
		}

		value, err := strconv.Atoi(dotParts[1])
		if err != nil {
			log.Warnf("invalid lyrics line time: %s", tag)
			return nil
		}
		if len(dotParts[1]) == 3 {
			result += value
		} else if len(dotParts[1]) == 2 {
			result += value * 10
		} else if len(dotParts[1]) == 1 {
			result += value * 100
		}
	}

	parts := strings.Split(dotParts[0], ":")
	if len(parts) > 3 || len(parts) == 0 {
		log.Warnf("invalid lyrics line time: %s", tag)
		return nil
	}

	multiplier := 1000
	for i := len(parts) - 1; i >= 0; i-- {
		value, err := strconv.Atoi(parts[i])
		if err != nil {
			log.Warnf("invalid lyrics line time: %s", tag)
			return nil
		}

		result += value * multiplier

		multiplier *= 60
	}

	return &result
}

func stripLRCMetadata(lyrics string) string {
	var builder strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(lyrics))
	wasEmpty := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		squareBracket := false
		angleBracket := false

		wasSpace := true

		empty := true

		for i, r := range line {
			if r == '\r' {
				continue
			}

			if i == 0 && r == '[' {
				squareBracket = true
				continue
			}
			if squareBracket {
				if r == ']' {
					squareBracket = false
				}
				continue
			}

			if i == 0 && r == '<' {
				angleBracket = true
				continue
			}
			if angleBracket {
				if r == '>' {
					angleBracket = false
				}
				continue
			}

			if unicode.IsSpace(r) {
				if !wasSpace {
					builder.WriteRune(r)
				}
				wasSpace = true
				continue
			}
			wasSpace = false
			empty = false
			builder.WriteRune(r)
		}

		if empty && wasEmpty {
			continue
		}
		wasEmpty = empty

		builder.WriteRune('\n')
	}

	return strings.TrimSpace(builder.String())
}
