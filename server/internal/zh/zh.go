// Package zh implements Chinese text normalization used by the search index,
// ported 1:1 from the original AnimeGarden TypeScript code (simptrad library).
package zh

import "strings"

var tradToSimp = func() map[rune]rune {
	m := make(map[rune]rune, len(tradChars))
	simps := []rune(simpChars)
	trads := []rune(tradChars)
	for i := 0; i < len(trads); i++ {
		m[trads[i]] = simps[i]
	}
	return m
}()

// TradToSimple converts traditional Chinese characters to simplified ones.
// Unknown characters are kept as-is.
func TradToSimple(text string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if s, ok := tradToSimp[r]; ok {
			b.WriteRune(s)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

const (
	fullSpaceCode = 12288
	widthOffset   = 65248

	fullAsciiStart = 65281
	fullAsciiEnd   = 65374
)

// fullPunctuations maps full-width punctuation to half-width, used by
// FullToHalf with punctuation enabled. Order per original code.
var fullPunctuations = map[rune]string{
	'。': ".",
	'～': "~",
	'─': "-",
	'・': "·",
	'【': "[",
	'】': "]",
	'“': "\"",
	'”': "\"",
	'‘': "'",
	'’': "'",
	'、': ",",
}

// FullToHalf converts full-width ASCII characters and (optionally)
// full-width punctuation to half-width, matching simptrad behavior.
func FullToHalf(text string, punctuation bool) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if punctuation {
			if p, ok := fullPunctuations[r]; ok {
				b.WriteString(p)
				continue
			}
		}
		if r == fullSpaceCode {
			// space unchanged unless enabled (space=false in original)
			b.WriteRune(r)
		} else if fullAsciiStart <= r && r <= fullAsciiEnd {
			b.WriteRune(r - widthOffset)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeTitle produces titleAlt: fullwidth to halfwidth (with
// punctuation), traditional to simplified. Matches normalizeTitle() in
// @animegarden/shared.
func NormalizeTitle(title string) string {
	return FullToHalf(TradToSimple(title), true)
}