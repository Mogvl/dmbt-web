package anipar

import (
	"regexp"
	"strings"
)

// jsWhiteSpace is the exact set of characters trimmed by JavaScript's
// String.prototype.trim()/trimEnd()/trimStart() (WhiteSpace + LineTerminator).
const jsWhiteSpace = "\u0009\u000a\u000b\u000c\u000d\u0020\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"

func jsTrim(s string) string {
	return strings.Trim(s, jsWhiteSpace)
}

func jsTrimLeft(s string) string {
	return strings.TrimLeft(s, jsWhiteSpace)
}

func jsTrimRight(s string) string {
	return strings.TrimRight(s, jsWhiteSpace)
}

// jsWhitespaceClass is the RE2 character class equivalent to JavaScript's \s
// (which also matches several Unicode space characters Go's \s does not).
const jsWhitespaceClass = "[\t\n\v\f\r \u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000\ufeff]"

// jsRegex compiles a JavaScript regex pattern (with or without an "i" flag)
// into a Go regexp. It rewrites \s to a JS-compatible whitespace class. The
// original patterns use no lookahead/backreferences, so a direct RE2
// translation is behavior-preserving. Named groups in the original are
// translated to numbered groups at the call sites.
func jsRegex(pattern string, caseInsensitive bool) *regexp.Regexp {
	if caseInsensitive {
		pattern = "(?i)" + pattern
	}
	pattern = strings.ReplaceAll(pattern, `\s`, jsWhitespaceClass)
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic("anipar: invalid regex " + pattern + ": " + err.Error())
	}
	return re
}

// strToInt converts a string to int; returns ok=false when the string cannot
// be parsed (mirrors Number.isNaN(+str)).
func strToInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// parseIntPrefix mirrors Number.parseInt for strings like "1st".
func parseIntPrefix(s string) int {
	n := 0
	seen := false
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		seen = true
		n = n*10 + int(r-'0')
	}
	if !seen {
		return 0
	}
	return n
}

// dedupeStrings removes duplicates preserving first-occurrence order.
func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// containsString reports whether s is in list.
func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// appendStringSet appends s to list unless already present.
func appendStringSet(list []string, s string) []string {
	if containsString(list, s) {
		return list
	}
	return append(list, s)
}
