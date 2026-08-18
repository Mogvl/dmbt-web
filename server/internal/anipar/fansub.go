package anipar

import "strings"

// Fansub names mirror the Fansub enum in fansub.ts.
const (
	FansubKiraraFantasia  = "Kirara Fantasia"
	FansubANi             = "ANi"
	FansubLoliHouse       = "LoliHouse"
	Fansub绿茶字幕组           = "绿茶字幕组"
	Fansub桜都字幕组           = "桜都字幕组"
	FansubPrejudiceStudio = "Prejudice-Studio"
	Fansub沸班亚马制作组         = "沸班亚马制作组"
	Fansub喵萌奶茶屋           = "喵萌奶茶屋"
	Fansub猎户发布组           = "猎户发布组"
	Fansub爱恋字幕社           = "爱恋字幕社"
	Fansub拨雪寻春            = "拨雪寻春"
	Fansub雪飄工作室           = "雪飄工作室(FLsnow)"
	Fansub幻樱字幕组           = "幻樱字幕组"
	FansubGMTeam          = "GMTeam"
	Fansub三明治摆烂组          = "三明治摆烂组"
	Fansub星空字幕组           = "星空字幕组"
	Fansub北宇治字幕组          = "北宇治字幕组"
	Fansub极影字幕社           = "极影字幕社"
	FansubMingYSub        = "MingYSub"
	Fansub黑白字幕组           = "黑白字幕组"
	FansubS1百综字幕组         = "S1百综字幕组"
)

var fansubTags = map[string]bool{
	"個人製作合集": true,
	"代发":     true,
	"羊圈个人译制": true,
}

var fansubSep = []string{"&", "＆", "·", "，"}

// parseFansub mirrors parseFansub in fansub.ts.
func parseFansub(ctx *Context) bool {
	// [個人製作合集][fansub] ...
	var tags []string
	for ctx.left <= ctx.right && fansubTags[ctx.tokens[ctx.left].text] {
		tags = append(tags, ctx.tokens[ctx.left].text)
		ctx.left++
	}
	if len(tags) > 0 {
		ctx.update2("fansub", "tags", tags)
	}

	if ctx.left+1 > ctx.right {
		return false
	}

	// [fansub] ...
	token := ctx.tokens[ctx.left]
	if token.IsWrapped() {
		text := token.text

		// @hack jibaketa
		if strings.HasPrefix(text, "jibaketa合成&") || text == "jibaketa" {
			ctx.update2("fansub", "name", "jibaketa")
			if text != "jibaketa" {
				ctx.update2("fansub", "alias", text)
			}
			ctx.left++
			return true
		}

		found := false
		for _, sep := range fansubSep {
			rawParts := strings.Split(text, sep)
			var parts []string
			for _, p := range rawParts {
				p = jsTrim(p)
				if p != "" {
					parts = append(parts, p)
				}
			}
			name := parts[0]
			collab := parts[1:]

			if len(collab) > 0 {
				found = true
				if ctx.result.Fansub != nil && ctx.result.Fansub.Name != "" &&
					ctx.result.Fansub.Name != name && !containsString(collab, ctx.result.Fansub.Name) {
					ctx.update2("fansub", "alias", name)
				} else {
					ctx.update2("fansub", "name", name)
				}
				ctx.update2("fansub", "collab", collab)
				break
			}
		}

		if !found {
			if ctx.result.Fansub != nil && ctx.result.Fansub.Name != "" && ctx.result.Fansub.Name != text {
				ctx.update2("fansub", "alias", text)
			} else {
				ctx.update2("fansub", "name", text)
			}
		}

		ctx.left++

		return true
	}

	return false
}
