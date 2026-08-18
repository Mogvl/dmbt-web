package anipar

import (
	"strings"
)

type splitOptions struct {
	space      bool
	separators []string
}

func defaultSplitOptions() splitOptions {
	return splitOptions{space: true, separators: []string{"/", "-"}}
}

func (o splitOptions) withSpace(space bool) splitOptions {
	o.space = space
	return o
}

func (o splitOptions) withSeparators(seps []string) splitOptions {
	o.separators = seps
	return o
}

// splitMultipleTitles mirrors splitMultipleTitles in title.ts.
func splitMultipleTitles(ctx *Context, options splitOptions) []string {
	rest := ctx.tokens[ctx.left : ctx.right+1]
	if len(rest) == 0 {
		return nil
	}

	// [xxx][yyy] 已经被分割好了
	if len(rest) > 1 {
		allWrapped := true
		for _, t := range rest {
			if !t.IsWrapped() {
				allWrapped = false
				break
			}
		}
		if allWrapped {
			result := make([]string, 0, len(rest))
			for _, r := range rest {
				result = append(result, jsTrim(r.text))
			}
			return result
		}
	}

	// "xxx" 或者 "[xxx]" 内的内容为一个整体 或者 "xxx [yyy] zzz" 被当成一个整体
	var fullText string
	if len(rest) == 1 {
		fullText = jsTrim(rest[0].text)
	} else {
		var b strings.Builder
		for _, t := range rest {
			b.WriteString(t.String())
		}
		fullText = jsTrim(b.String())
	}

	if fullText == "" {
		return nil
	}

	for _, separator := range options.separators {
		var parts []string
		if options.space {
			parts = strings.Split(fullText, " "+separator+" ")
		} else {
			parts = strings.Split(fullText, separator)
		}

		if strings.HasPrefix(fullText, separator) && len(parts) > 0 {
			parts[0] += separator
		}
		if strings.HasSuffix(fullText, separator) && len(parts) > 0 {
			parts[len(parts)-1] += separator
		}

		result := []string{}
		for i := 0; i < len(parts); i++ {
			if !options.space && separator == "/" && i+1 < len(parts) {
				if strings.HasSuffix(strings.ToUpper(parts[i]), "FATE") ||
					strings.HasSuffix(strings.ToUpper(parts[i]), "命运") ||
					strings.HasSuffix(strings.ToUpper(parts[i]), "命運") ||
					(endsWithDigit(parts[i]) && startsWithDigit(parts[i+1])) {
					result = append(result, parts[i]+separator+parts[i+1])
					i++
					continue
				}
			}
			result = append(result, parts[i])
		}

		if len(result) > 1 {
			trimmed := make([]string, 0, len(result))
			for _, r := range result {
				trimmed = append(trimmed, jsTrim(r))
			}
			return trimmed
		}
	}

	return []string{jsTrim(fullText)}
}

func endsWithDigit(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[len([]rune(s))-1]
	return '0' <= r && r <= '9'
}

func startsWithDigit(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	return '0' <= r && r <= '9'
}

// parseSingleTitleText mirrors parseSingleTitleText in title.ts.
func parseSingleTitleText(ctx *Context, text string) string {
	text = jsTrimRight(parseSuffixTextInlineTags(ctx, text))
	text = jsTrimRight(parseSuffixTextInlineSeason(ctx, text))
	text = jsTrimRight(parseSuffixTextInlineTags(ctx, text))
	return text
}

// parseMultipleTitles mirrors parseMultipleTitles in title.ts.
func parseMultipleTitles(ctx *Context, options splitOptions) []string {
	titles := splitMultipleTitles(ctx, options)
	if len(titles) == 0 {
		return nil
	}

	var trimmedTitles []string
	for _, t := range titles {
		t = parseSingleTitleText(ctx, t)
		if t != "" {
			trimmedTitles = append(trimmedTitles, t)
		}
	}
	if len(trimmedTitles) == 0 {
		// TS tolerates trimmedTitles[0] === undefined; mirror that by
		// leaving the title unset (normalize() will return nil).
		return nil
	}
	trimmedTitle := trimmedTitles[0]

	ctx.update("title", trimmedTitle)

	otherTitles := []string{}
	for _, t := range dedupeStrings(trimmedTitles) {
		if t != trimmedTitle {
			otherTitles = append(otherTitles, t)
		}
	}
	if len(otherTitles) > 0 {
		ctx.update("titles", otherTitles)
	}

	return append([]string{trimmedTitle}, otherTitles...)
}

// parseSingleTitle mirrors parseSingleTitle in title.ts.
func parseSingleTitle(ctx *Context) string {
	rest := ctx.tokens[ctx.left : ctx.right+1]
	if len(rest) == 0 {
		return ""
	}

	var fullText string
	if len(rest) == 1 {
		fullText = jsTrim(rest[0].text)
	} else {
		var b strings.Builder
		for _, t := range rest {
			b.WriteString(t.String())
		}
		fullText = jsTrim(b.String())
	}

	title := parseSingleTitleText(ctx, fullText)
	if title != "" {
		ctx.update("title", title)
	}

	return title
}
