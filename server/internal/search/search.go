// Package search implements jieba tokenization for title search,
// equivalent to @node-rs/jieba used by the original AnimeGarden server.
package search

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/Mogvl/dmbt-web/server/internal/search/jiebago"

	"github.com/Mogvl/dmbt-web/server/internal/zh"
)

//go:embed dict.txt
var dictData []byte

// Tokenizer wraps jiebago with the standard jieba dictionary.
type Tokenizer struct {
	mu  sync.RWMutex
	seg *jiebago.Segmenter
}

var (
	once     sync.Once
	instance *Tokenizer
	initErr  error
)

// NewTokenizer loads the embedded dictionary and returns a tokenizer.
func NewTokenizer() (*Tokenizer, error) {
	once.Do(func() {
		seg := &jiebago.Segmenter{}
		if err := seg.LoadDictionaryFromBytes(dictData); err != nil {
			initErr = err
			return
		}
		instance = &Tokenizer{seg: seg}
	})
	if initErr != nil {
		return nil, initErr
	}
	return instance, nil
}

// Cut segments s with HMM disabled, mirroring jieba.cut(s, false) in the
// original. Returns trimmed non-empty tokens.
func (t *Tokenizer) Cut(s string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]string, 0, 8)
	for token := range t.seg.Cut(s, false) {
		token = strings.TrimSpace(token)
		if token != "" {
			out = append(out, token)
		}
	}
	return out
}

// RemovePunctuations replaces every punctuation/symbol rune with a space,
// equivalent to removePunctuations() in @animegarden/shared.
func RemovePunctuations(input string) string {
	if input == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		if isPunctOrSymbol(r) {
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isPunctOrSymbol(r rune) bool {
	// Unicode categories P (punctuation) and S (symbols), including CJK.
	switch {
	case r >= 0x21 && r <= 0x2F,
		r >= 0x3A && r <= 0x40,
		r >= 0x5B && r <= 0x60,
		r >= 0x7B && r <= 0x7E:
		return true
	case r >= 0xA1 && r <= 0xA9,
		r >= 0xAB && r <= 0xB6,
		r >= 0xBB && r <= 0xBF,
		r >= 0xD7, r >= 0xF7:
		return true
	case r >= 0x2000 && r <= 0x206F: // General Punctuation
		return true
	case r >= 0x3000 && r <= 0x303F: // CJK Symbols and Punctuation
		return true
	case r >= 0xFE30 && r <= 0xFE4F: // CJK Compatibility Forms
		return true
	case r >= 0xFE50 && r <= 0xFE6F: // Small Form Variants
		return true
	case r >= 0xFF00 && r <= 0xFF0F, // Fullwidth forms incl. punctuation
		r >= 0xFF1A && r <= 0xFF20,
		r >= 0xFF3B && r <= 0xFF40,
		r >= 0xFF5B && r <= 0xFF65:
		return true
	case r >= 0x1F300 && r <= 0x1FAFF: // Emoji symbols
		return true
	case r >= 0x2600 && r <= 0x27BF: // Misc symbols
		return true
	}
	return false
}

// NormalizeTitle mirrors normalizeTitle() from @animegarden/shared.
func NormalizeTitle(title string) string {
	return zh.NormalizeTitle(title)
}