package anipar

import "strings"

// Token mirrors the TypeScript Token class.
type Token struct {
	text  string
	left  string
	right string
}

func NewToken(text, left, right string) *Token {
	return &Token{text: text, left: left, right: right}
}

func (t *Token) IsWrapped() bool {
	return t.left != "" && t.right != ""
}

func (t *Token) Trim() *Token {
	return &Token{text: jsTrim(t.text), left: t.left, right: t.right}
}

func (t *Token) Text() string { return t.text }

func (t *Token) String() string {
	var b strings.Builder
	b.WriteString(t.left)
	b.WriteString(t.text)
	b.WriteString(t.right)
	return b.String()
}

// Wrappers maps opening to closing bracket characters.
var wrappers = map[rune]string{
	'[': "]",
	'【': "】",
	'(': ")",
	'（': "）",
	'{': "}",
}

var reverseWrappers = func() map[string]rune {
	m := make(map[string]rune, len(wrappers))
	for k, v := range wrappers {
		m[v] = k
	}
	return m
}()

// tokenize splits a title on bracket pairs with nested-wrapper support,
// mirroring tokenize() in tokenizer.ts.
func tokenize(text string) []*Token {
	var tokens []*Token

	runes := []rune(text)
	cursor := 0
	var cur []rune
	left := rune(0)
	right := ""

	flush := func() {
		if len(cur) > 0 {
			tokens = append(tokens, &Token{text: string(cur)})
			cur = nil
		}
	}

	for cursor < len(runes) {
		r := runes[cursor]
		if w, ok := wrappers[r]; ok {
			if left == 0 {
				flush()
				left = r
				right = w
			} else {
				// nest wrapper
				cur = append(cur, r)
			}
		} else if left != 0 && right != "" && string(r) == right {
			if len(cur) > 0 {
				tokens = append(tokens, &Token{text: string(cur), left: string(left), right: right})
			}
			cur = nil
			left = 0
			right = ""
		} else {
			cur = append(cur, r)
		}
		cursor++
	}
	flush()

	// filter tokens with empty text
	filtered := tokens[:0]
	for _, t := range tokens {
		if t.text != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
