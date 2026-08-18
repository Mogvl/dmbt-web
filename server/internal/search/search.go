// Package search implements jieba tokenization for title search,
// equivalent to @node-rs/jieba used by the original AnimeGarden server.
package search

import (
	_ "embed"
	"strings"
	"sync"
	"unicode"

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
	// Equivalent to JS \p{P} (punctuation) and \p{S} (symbols).
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

// NormalizeTitle mirrors normalizeTitle() from @animegarden/shared.
func NormalizeTitle(title string) string {
	return zh.NormalizeTitle(title)
}
