package anipar

import (
	"regexp"
	"strings"
)

// Types mirrors the Types set in episodes.ts.
var Types = []string{
	"GEKIJOUBAN",
	"MOVIE",
	"OAD",
	"OAV",
	"ONA",
	"OVA",
	"SPECIAL",
	"SPECIALS",
	"TV",
	"开播纪念特别篇",
	"開播紀念特別篇",
	"开篇纪念特别篇",
	"開篇紀念特別篇",
	"特别篇",
	"特別篇",
	"特別編",
	"特别话",
	"特別话",
	"特別話",
	"番外篇",
	"番外編",
	"剧场版",
	"劇場版",
	"总集篇",
	"總集篇",
	//
	"广播剧",
	"朗读剧",
	//
	"SP",
	//
	"ED",
	"ENDING",
	"NCED",
	"NCOP",
	"OP",
	"OPENING",
	"PREVIEW",
	"PV",
	"特别篇PV",
	//
	"合集",
	"修正合集",
}

var typesSet = func() map[string]bool {
	m := make(map[string]bool, len(Types))
	for _, t := range Types {
		m[t] = true
	}
	return m
}()

var versionSuffixRE = jsRegex(`[vV](\d+)$`, false)

// MARK: wrapped-token episode regexes

// wrappedEpisodeRE1: "01", "1.5", "3v2", "OVA01", "第01话", "S01E01",
// "S01E01(END)". Extra branch for the SxxExx ep-type is group 10.
var wrappedEpisodeRE1 = jsRegex(`^(TV|OVA|OAD|SP)?(\d+)(?:\.(\d))?(?:[vV](\d+))?(?:(?:\s+|_|-)?([^\d集]+))?$|^第(\d+)[集话話]$|^S(\d+)E(\d+)((?:[_|（])?([^\d][^）]*)(?:）)?)?$`, false)

// wrappedEpisodeRE2: "06,07" / "07+ES07".
var wrappedEpisodeRE2 = jsRegex(`^(\d+)(?:\+([^\d]+)|,|&|、)(\d+)$`, false)

var wrappedSeasonRE = jsRegex(`^(?:S|Season)(\d+)\s*(Fin|End)?$`, false)

var wrappedMovieRE = jsRegex(`^Movie [vV](\d+)$`, false)

// wrappedEpisodesRange1: "01-26", "TV01-13 修正合集", "12.5-23".
var wrappedEpisodesRange1 = jsRegex(`^(TV|OVA|OAD|SP)?(\d+)(?:\.(\d))?[-~](\d+)(?:\.(\d))?\s*[_]?(.*)$`, false)

var wrappedEpisodesRange2 = jsRegex(`^全(\d+)集$`, false)

var wrappedSeasonsRE = jsRegex(`^S(\d)\+S(\d)$`, false)

var wrappedSeasonsRangeRE = jsRegex(`^S(\d)-S(\d)$`, false)

var wrappedVolumeRE = jsRegex(`^(?:Vol|vol|Volume|volume)\.?\s*(\d+)$`, false)

var wrappedVolumesRangeRE = jsRegex(`^(?:Vol|vol|Volume|volume)\.?\s*(\d+)-(\d+)\s+(.*)$`, false)

// matchEpiodes mirrors matchEpiodes in episodes.ts.
func matchEpiodes(ctx *Context, text string) bool {
	text = jsTrimRight(text)

	// 1. Single episode
	{
		res := wrappedEpisodeRE1.FindStringSubmatch(text)
		if res != nil {
			// groups: 1=type, 2=ep1, 3=sub, 4=version, 5=ep_type,
			// 6=ep2(第N话), 7=season, 8=ep3, 9=ep_type-outer, 10=ep_type
			epText := res[2]
			if epText == "" {
				epText = res[6]
			}
			if epText == "" {
				epText = res[8]
			}
			if ep, ok := strToInt(epText); ok {
				// Handle year: [2024]
				if 1949 <= ep && ep <= 2099 && text == epText && ctx.HasEpisode() {
					ctx.update("year", ep)
					return true
				}

				ctx.update2("episode", "number", ep)

				// 1.5
				if res[3] != "" {
					if sub, ok := strToInt(res[3]); ok {
						ctx.update2("episode", "numberSub", sub)
					}
				}

				// 3v2
				if res[4] != "" {
					if version, ok := strToInt(res[4]); ok {
						ctx.update("version", version)
					}
				}

				// SP01 OVA01
				if res[1] != "" {
					ctx.update("type", jsTrim(res[1]))
				}

				// 01 END / S01E01(END)
				epType := res[5]
				if epType == "" {
					epType = res[10]
				}
				if epType != "" {
					ctx.update2("episode", "type", jsTrim(epType))
				}

				// S01E01
				if res[7] != "" {
					if season, ok := strToInt(res[7]); ok {
						ctx.update2("season", "number", season)
					}
				}

				return true
			}
		}
	}
	{
		// 06,07 07+ES07
		res := wrappedEpisodeRE2.FindStringSubmatch(text)
		if res != nil {
			ep1, ok1 := strToInt(res[1])
			ep2, ok2 := strToInt(res[3])
			if ok1 && ok2 {
				if res[2] != "" {
					ctx.update2("episode", "number", ep1)
					ctx.update("episodes", []EpisodeInfo{{Number: ep2, Type: res[2]}})
				} else {
					ctx.update("episodes", []EpisodeInfo{{Number: ep1}, {Number: ep2}})
				}
				return true
			}
		}
	}
	{
		// [Movie v2]
		res := wrappedMovieRE.FindStringSubmatch(text)
		if res != nil {
			if res[1] != "" {
				if version, ok := strToInt(res[1]); ok {
					ctx.update("version", version)
				}
			}
			ctx.update("type", "Movie")
			return true
		}
	}

	// 2. Episodes range
	{
		// 01-26
		res := wrappedEpisodesRange1.FindStringSubmatch(text)
		if res != nil {
			// 1=type, 2=ep1, 3=sub1, 4=ep2, 5=sub2, 6=range_type
			from, ok1 := strToInt(res[2])
			to, ok2 := strToInt(res[4])
			if ok1 && ok2 {
				ctx.update2("episodesRange", "from", from)
				ctx.update2("episodesRange", "to", to)

				if res[1] != "" {
					ctx.update("type", jsTrim(res[1]))
				}

				episodesRangeType := ""
				if res[6] != "" {
					episodesRangeType = jsTrim(res[6])
				}
				if episodesRangeType != "" {
					exec2 := versionSuffixRE.FindStringSubmatch(episodesRangeType)
					if exec2 != nil {
						if version, ok := strToInt(exec2[1]); ok {
							ctx.update("version", version)
							ctx.update2("episodesRange", "type", episodesRangeType[:len(episodesRangeType)-len(exec2[0])])
						} else {
							ctx.update2("episodesRange", "type", episodesRangeType)
						}
					} else {
						ctx.update2("episodesRange", "type", episodesRangeType)
					}
				}

				// 12.5-23
				if res[3] != "" {
					if sub1, ok := strToInt(res[3]); ok {
						ctx.update2("episodesRange", "fromSub", sub1)
					}
				}
				if res[5] != "" {
					if sub2, ok := strToInt(res[5]); ok {
						ctx.update2("episodesRange", "toSub", sub2)
					}
				}

				return true
			}
		}
	}
	{
		// 全26集
		res := wrappedEpisodesRange2.FindStringSubmatch(text)
		if res != nil {
			if to, ok := strToInt(res[1]); ok {
				ctx.update2("episodesRange", "from", 1)
				ctx.update2("episodesRange", "to", to)
				return true
			}
		}
	}

	// 3. Season
	{
		res := wrappedSeasonRE.FindStringSubmatch(text)
		if res != nil {
			if season, ok := strToInt(res[1]); ok {
				ctx.update2("season", "number", season)
				if res[2] != "" {
					ctx.tags = append(ctx.tags, jsTrim(res[2]))
				}
				return true
			}
		}
	}

	// 4. Seasons Range
	{
		res := wrappedSeasonsRE.FindStringSubmatch(text)
		if res != nil {
			season1, ok1 := strToInt(res[1])
			season2, ok2 := strToInt(res[2])
			if ok1 && ok2 {
				ctx.update("seasons", []SeasonInfo{{Number: season1}, {Number: season2}})
			}
			return true
		}
	}
	{
		// S1-S2
		res := wrappedSeasonsRangeRE.FindStringSubmatch(text)
		if res != nil {
			season1, ok1 := strToInt(res[1])
			season2, ok2 := strToInt(res[2])
			if ok1 && ok2 {
				ctx.update2("seasonsRange", "from", season1)
				ctx.update2("seasonsRange", "to", season2)
			}
			return true
		}
	}

	// 5. Volume
	{
		res := wrappedVolumeRE.FindStringSubmatch(text)
		if res != nil {
			if vol, ok := strToInt(res[1]); ok {
				ctx.update2("volume", "number", vol)
				return true
			}
		}
	}
	{
		res := wrappedVolumesRangeRE.FindStringSubmatch(text)
		if res != nil {
			vol1, ok1 := strToInt(res[1])
			_, ok2 := strToInt(res[2])
			if ok1 && ok2 {
				ctx.update2("volumesRange", "from", vol1)
				// NOTE: the original writes 'to: vol1' (a source bug), ported as-is.
				ctx.update2("volumesRange", "to", vol1)
				if res[3] != "" {
					ctx.update2("volumesRange", "type", res[3])
				}
				return true
			}
		}
	}

	return false
}

// parseWrappedEpisodes mirrors parseWrappedEpisodes in episodes.ts.
func parseWrappedEpisodes(ctx *Context) bool {
	if ctx.HasEpisode() {
		return true
	}
	token := ctx.tokens[ctx.right]
	if token.IsWrapped() && matchEpiodes(ctx, token.text) {
		ctx.right--
		return true
	}
	return false
}

// SuffixEpisodeREs: inline text suffix episode regexes.
var (
	suffixEpisodeRE1 = jsRegex(`\s*- (SP|OVA)?(\d+)(?:\.(\d))?(?:[vV](\d+))?(\s+[^\-]*)?(?:\s*-)?$`, false)
	suffixEpisodeRE2 = jsRegex(`\s+S(\d+)E(\d+)$`, false)
	suffixEpisodeRE3 = jsRegex(`\s*第(\d+)(?:\.(\d))?[集话話]$`, false)
	suffixEpisodeRE4 = jsRegex(`\s+S(\d+)-S(\d+)$`, false)
)

var suffixEpisodesRangeRE = jsRegex(`- (\d+)-(\d+)(?:\s+(.+))?$`, false)

// parseSuffixTextInlineEpisodes mirrors parseSuffixTextInlineEpisodes.
func parseSuffixTextInlineEpisodes(ctx *Context, text string) string {
	if ctx.HasEpisode() {
		return text
	}

	text = jsTrimRight(text)

	// - 01-24 修正合集
	{
		res := suffixEpisodesRangeRE.FindStringSubmatch(text)
		if res != nil {
			from, ok1 := strToInt(res[1])
			to, ok2 := strToInt(res[2])
			if ok1 && ok2 {
				ctx.update2("episodesRange", "from", from)
				ctx.update2("episodesRange", "to", to)

				typ := ""
				if res[3] != "" {
					typ = jsTrim(res[3])
				}
				if typ != "" {
					version := versionSuffixRE.FindStringSubmatch(typ)
					if version != nil {
						versionNumber, ok := strToInt(version[1])
						if ok {
							ctx.update("version", versionNumber)
							typeWithoutVersion := jsTrim(typ[:len(typ)-len(version[0])])
							if typeWithoutVersion != "" {
								ctx.update2("episodesRange", "type", typeWithoutVersion)
							}
						} else {
							ctx.update2("episodesRange", "type", typ)
						}
					} else {
						ctx.update2("episodesRange", "type", typ)
					}
				}

				return jsTrimRight(text[:len(text)-len(res[0])])
			}
		}
	}

	// episodes
	{
		res := suffixEpisodeRE1.FindStringSubmatch(text)
		if res != nil {
			// 1=type, 2=ep1, 3=sub, 4=version, 5=ep_type
			if ep, ok := strToInt(res[2]); ok {
				if res[1] != "" {
					ctx.update("type", res[1])
				}
				ctx.update2("episode", "number", ep)
				if res[3] != "" {
					if numberSub, ok := strToInt(res[3]); ok && numberSub != 0 {
						ctx.update2("episode", "numberSub", numberSub)
					}
				}
				if res[4] != "" {
					if version, ok := strToInt(res[4]); ok && version != 0 {
						ctx.update("version", version)
					}
				}
				return jsTrimRight(text[:len(text)-len(res[0])])
			}
		}
	}
	{
		res := suffixEpisodeRE2.FindStringSubmatch(text)
		if res != nil {
			// 1=season, 2=ep1
			if ep, ok := strToInt(res[2]); ok {
				ctx.update2("episode", "number", ep)
				if season, ok := strToInt(res[1]); ok {
					ctx.update2("season", "number", season)
				}
				return jsTrimRight(text[:len(text)-len(res[0])])
			}
		}
	}
	{
		res := suffixEpisodeRE3.FindStringSubmatch(text)
		if res != nil {
			// 1=ep1, 2=sub
			if ep, ok := strToInt(res[1]); ok {
				ctx.update2("episode", "number", ep)
				if res[2] != "" {
					if numberSub, ok := strToInt(res[2]); ok && numberSub != 0 {
						ctx.update2("episode", "numberSub", numberSub)
					}
				}
				return jsTrimRight(text[:len(text)-len(res[0])])
			}
		}
	}
	{
		res := suffixEpisodeRE4.FindStringSubmatch(text)
		if res != nil {
			// S1-S2
			season1, ok1 := strToInt(res[1])
			season2, ok2 := strToInt(res[2])
			if ok1 && ok2 && season1 < season2 {
				ctx.update2("seasonsRange", "from", season1)
				ctx.update2("seasonsRange", "to", season2)
				return jsTrimRight(text[:len(text)-len(res[0])])
			}
		}
	}

	// - 特别篇
	for _, typ := range Types {
		toMatch := " - " + typ
		if strings.HasSuffix(text, toMatch) {
			ctx.update("type", typ)
			return jsTrimRight(text[:len(text)-len(toMatch)])
		}
	}

	return text
}

// parseSuffixEpisodes mirrors parseSuffixEpisodes.
func parseSuffixEpisodes(ctx *Context) bool {
	if ctx.HasEpisode() {
		return true
	}

	token := ctx.tokens[ctx.right]
	text := jsTrimRight(token.text)

	if token.IsWrapped() {
		return parseWrappedEpisodes(ctx)
	}
	trimmed := parseSuffixTextInlineEpisodes(ctx, text)
	if trimmed != text {
		ctx.tokens[ctx.right] = NewToken(trimmed, token.left, token.right)
		return true
	}

	return false
}

// MARK: suffix season / part matching

type suffixSeasonFn func(res []string, ctx *Context) bool

type suffixSeasonEntry struct {
	re *regexp.Regexp
	fn suffixSeasonFn
}

// seasonUnset mirrors `!ctx.result.season?.number` (season absent or 0).
func (c *Context) seasonUnset() bool {
	return c.result.Season == nil || c.result.Season.Number == 0
}

// seasonMatches mirrors `!ctx.result.season?.number || ctx.result.season.number === season`.
func (c *Context) seasonMatches(season int) bool {
	return c.seasonUnset() || c.result.Season.Number == season
}

var suffixSeasonOrEpisodesRes []suffixSeasonEntry

func init() {
	suffixSeasonOrEpisodesRes = []suffixSeasonEntry{
		{
			re: jsRegex(`Parts? (\d+)$`, false),
			fn: func(res []string, ctx *Context) bool {
				if part, ok := strToInt(res[1]); ok {
					ctx.update2("part", "number", part)
					return true
				}
				return false
			},
		},
		{
			re: jsRegex(`第\s*(\d+)\s*部分$`, false),
			fn: func(res []string, ctx *Context) bool {
				if part, ok := strToInt(res[1]); ok {
					ctx.update2("part", "number", part)
					return true
				}
				return false
			},
		},
		{
			re: jsRegex(`(?:S|Season\s?)(\d+)$`, false),
			fn: func(res []string, ctx *Context) bool {
				if season, ok := strToInt(res[1]); ok {
					if ctx.seasonMatches(season) {
						ctx.update2("season", "number", season)
						return true
					}
				}
				return false
			},
		},
		{
			re: jsRegex(`(1st|2nd|3rd|[456789]th) Season$`, false),
			fn: func(res []string, ctx *Context) bool {
				season := parseIntPrefix(res[1])
				if ctx.seasonMatches(season) {
					ctx.update2("season", "number", season)
					return true
				}
				return false
			},
		},
		{
			re: jsRegex(`(?:-\s+)(Third) Season$`, false),
			fn: func(res []string, ctx *Context) bool {
				season := 0
				if res[1] == "Third" {
					season = 3
				}
				if season != 0 && ctx.seasonMatches(season) {
					ctx.update2("season", "number", season)
					return true
				}
				return false
			},
		},
		{
			re: jsRegex(`第?(\d+)[季期]$`, false),
			fn: func(res []string, ctx *Context) bool {
				if season, ok := strToInt(res[1]); ok {
					if ctx.seasonMatches(season) {
						ctx.update2("season", "number", season)
						return true
					}
				}
				return false
			},
		},
		{
			re: jsRegex(`第?((?:[零一二三四五六七八九]十)?[零一二三四五六七八九])[季期]$`, false),
			fn: func(res []string, ctx *Context) bool {
				text := res[1]
				cnMap := map[rune]int{'零': 0, '一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
				runes := []rune(text)
				base := 0
				if len(runes) == 2 && runes[0] == '十' {
					base = 1
				} else if len(runes) == 3 {
					base = cnMap[runes[0]]
				}
				offset := cnMap[runes[len(runes)-1]]
				season := base*10 + offset
				if ctx.seasonMatches(season) {
					ctx.update2("season", "number", season)
					return true
				}
				return false
			},
		},
		{
			re: jsRegex(`(?:Vol|vol|Volume|volume)\.?\s*(\d+)$`, false),
			fn: func(res []string, ctx *Context) bool {
				if vol, ok := strToInt(res[1]); ok {
					ctx.update2("volume", "number", vol)
					return true
				}
				return false
			},
		},
		{
			re: jsRegex(`\s+[vV](\d+)$`, false),
			fn: func(res []string, ctx *Context) bool {
				if version, ok := strToInt(res[1]); ok {
					ctx.update("version", version)
					return true
				}
				return false
			},
		},
		{
			re: jsRegex(`\s+(\d+)$|\((\d+)\)$`, false),
			fn: func(res []string, ctx *Context) bool {
				text := res[1]
				if text == "" {
					text = res[2]
				}
				if season, ok := strToInt(text); ok && ctx.HasEpisode() {
					if ctx.seasonMatches(season) {
						ctx.update2("season", "number", season)
						return true
					} else if (ctx.result.Year == 0 || ctx.result.Year == season) && 1949 <= season && season <= 2099 {
						ctx.update("year", season)
						return true
					}
				}
				return false
			},
		},
	}
}

// parseSuffixTextInlineSeason mirrors parseSuffixTextInlineSeason.
func parseSuffixTextInlineSeason(ctx *Context, text string) string {
	changed := false
	for {
		changed = false
		text = jsTrimRight(text)
		for _, entry := range suffixSeasonOrEpisodesRes {
			res := entry.re.FindStringSubmatch(text)
			if res != nil && entry.fn(res, ctx) {
				changed = true
				text = jsTrimRight(text[:len(text)-len(res[0])])
			}
		}
		if !changed {
			break
		}
	}
	return text
}
