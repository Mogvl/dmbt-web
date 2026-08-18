package scraper

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/scraper/htmlx"
	"golang.org/x/net/html"
)

// dmhy constants, mirroring packages/scraper/src/dmhy.
const (
	DmhyBase   = "https://share.dmhy.org"
	DmhyName   = "dmhy"
	DmhyDisplayName = "动漫花园"
)

// DisplayType maps traditional dmhy category names to simplified.
var dmhyDisplayType = map[string]string{
	"動畫":     "动画",
	"季度全集":   "季度全集",
	"音樂":     "音乐",
	"動漫音樂":   "动漫音乐",
	"同人音樂":   "同人音乐",
	"流行音樂":   "流行音乐",
	"日劇":     "日剧",
	"ＲＡＷ":    "RAW",
	"其他":     "其他",
	"漫畫":     "漫画",
	"港台原版":   "港台原版",
	"日文原版":   "日文原版",
	"遊戲":     "游戏",
	"電腦遊戲":   "电脑游戏",
	"電視遊戲":   "主机游戏",
	"掌機遊戲":   "掌机游戏",
	"網絡遊戲":   "网络游戏",
	"遊戲周邊":   "游戏周边",
	"特攝":     "特摄",
}

// SimpleType canonicalizes a type to one of the canonical set.
var dmhySimpleType = map[string]string{
	"动画":  "动画",
	"季度全集": "合集",
	"音乐":  "音乐",
	"动漫音乐": "音乐",
	"同人音乐": "音乐",
	"流行音乐": "音乐",
	"日剧":  "日剧",
	"RAW": "RAW",
	"其他":  "其他",
	"漫画":  "漫画",
	"港台原版": "漫画",
	"日文原版": "漫画",
	"游戏":  "游戏",
	"电脑游戏": "游戏",
	"主机游戏": "游戏",
	"掌机游戏": "游戏",
	"网络游戏": "游戏",
	"游戏周边": "游戏",
	"特摄":  "特摄",
}

// dmhyType converts a raw dmhy category text to a canonical type.
func dmhyType(raw string) string {
	t := dmhyDisplayType[raw]
	if t == "" {
		t = raw
	}
	if s, ok := dmhySimpleType[t]; ok {
		return s
	}
	return "动画"
}

// toShanghai ports the original toShanghai: treats the parsed wall time as
// Asia/Shanghai and returns the correct UTC instant.
func toShanghai(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, shanghaiLocation).UTC()
}

var dmhyIDRe = regexp.MustCompile(`^(\d+)`)

// FetchDmhyPage fetches one dmhy list page and parses the rows.
// Mirrors fetchDmhyPage from packages/scraper/src/dmhy/index.ts.
func FetchDmhyPage(page int, retry int) ([]ScrapedResource, error) {
	return retryFn(retry, func() ([]ScrapedResource, error) {
		url := fmt.Sprintf("%s/topics/list/page/%d", DmhyBase, page)
		resp, err := fetchURL("GET", url, "", nil, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		doc, err := htmlx.Parse(string(body))
		if err != nil {
			return nil, err
		}
		// .ui-state-error -> NetworkError
		if len(htmlx.Classes(doc, "ui-state-error")) > 0 {
			return nil, &NetworkError{Provider: DmhyName, URL: url, Status: 0}
		}

		var out []ScrapedResource
		tables := htmlx.Tag(doc, "table")
		for _, table := range tables {
			if htmlx.Attr(table, "id") != "topic_list" {
				continue
			}
			for _, tr := range htmlx.Tag(table, "tr") {
				tds := htmlx.Tag(tr, "td")
				if len(tds) < 9 {
					continue
				}
				r, ok := parseDmhyRow(tds)
				if ok {
					out = append(out, r)
				}
			}
			break
		}
		return out, nil
	})
}

func parseDmhyRow(tds []*html.Node) (ScrapedResource, bool) {
	// td[0] time
	rawTime := strings.TrimSpace(htmlx.Text(tds[0]))
	t, err := time.Parse("2006/01/02 15:04", rawTime)
	if err != nil {
		return ScrapedResource{}, false
	}
	createdAt := toShanghai(t)

	// td[1] category
	rawType := strings.TrimSpace(htmlx.Text(tds[1]))
	typ := dmhyType(rawType)

	// td[2] title + href + fansub
	var title, href string
	var fansub *Party
	as := htmlx.Tag(tds[2], "a")
	if len(as) > 0 {
		title = strings.TrimSpace(htmlx.Text(as[0]))
		href = htmlx.Attr(as[0], "href")
	}
	for _, span := range htmlx.Tag(tds[2], "span") {
		if !htmlx.HasClass(span, "tag") {
			continue
		}
		for _, a := range htmlx.Tag(span, "a") {
			id := lastSegment(htmlx.Attr(a, "href"))
			name := strings.TrimSpace(htmlx.Text(a))
			if name != "" {
				fansub = &Party{ID: id, Name: name}
			}
		}
	}
	if title == "" || href == "" {
		return ScrapedResource{}, false
	}
	absHref := DmhyBase + href

	// td[3] magnet + tracker
	magnetFull := ""
	if len(htmlx.Tag(tds[3], "a")) > 0 {
		magnetFull = htmlx.Attr(htmlx.Tag(tds[3], "a")[0], "href")
	}
	magnet, tracker := splitOnce(magnetFull, "&")

	// td[4] size
	size := strings.TrimSpace(htmlx.Text(tds[4]))

	// td[8] publisher
	var publisher *Party
	if len(tds) > 8 {
		as := htmlx.Tag(tds[8], "a")
		if len(as) > 0 {
			publisher = &Party{
				ID:   lastSegment(htmlx.Attr(as[0], "href")),
				Name: strings.TrimSpace(htmlx.Text(as[0])),
			}
		}
	}

	providerID := ""
	if m := dmhyIDRe.FindStringSubmatch(lastSegment(absHref)); m != nil {
		providerID = m[1]
	}

	title = titleCleanup(title, fansubName(fansub))

	// publisher / fansub rewrites
	if publisher != nil && publisher.Name == "ANiTorrent" {
		publisher.Name = "ANi"
	}
	if publisher != nil && publisher.Name == "悠哈C9字幕社" {
		publisher.Name = "悠哈璃羽字幕社"
	}
	if fansub != nil && fansub.Name == "悠哈C9字幕社" {
		fansub.Name = "悠哈璃羽字幕社"
	}
	if publisher != nil && publisher.Name == "灼眼のシャナ" && publisher.ID == "110897" {
		publisher = &Party{ID: "747291", Name: "ANi"}
		fansub = &Party{ID: "816", Name: "ANi"}
		title = strings.TrimPrefix(title, "[搬運]")
	}

	return ScrapedResource{
		Provider:   DmhyName,
		ProviderID: providerID,
		Title:      title,
		Href:       lastSegment(absHref),
		Type:       typ,
		Magnet:     magnet,
		Tracker:    tracker,
		Size:       size,
		Publisher:  publisher,
		Fansub:     fansub,
		CreatedAt:  createdAt,
	}, true
}

func fansubName(p *Party) string {
	if p == nil {
		return ""
	}
	return p.Name
}

// lastSegment returns the last path segment of a URL-ish string.
func lastSegment(href string) string {
	idx := strings.LastIndex(href, "/")
	if idx == -1 {
		return href
	}
	return href[idx+1:]
}

// FetchDmhyDetail fetches and parses a dmhy detail page.
// Mirrors fetchDmhyDetail from packages/scraper/src/dmhy/index.ts.
func FetchDmhyDetail(id string, retry int) (*ScrapedResourceDetail, error) {
	return retryFn(retry, func() (*ScrapedResourceDetail, error) {
		url := fmt.Sprintf("%s/topics/view/%s", DmhyBase, id)
		resp, err := fetchURL("GET", url, "", nil, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		doc, err := htmlx.Parse(string(body))
		if err != nil {
			return nil, err
		}

		// title
		titleNode := htmlx.First(doc, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "h3" && hasAncestorClass(n, "topic-title")
		})
		if titleNode == nil {
			return nil, nil // deleted resource
		}
		title := strings.TrimSpace(htmlx.Text(titleNode))

		// type: .topic-title li:first-child a:last-of-type
		var typ string
		lis := htmlx.Tag(titleNode, "li")
		if len(lis) > 0 {
			as := htmlx.Tag(lis[0], "a")
			if len(as) > 0 {
				typ = dmhyType(strings.TrimSpace(htmlx.Text(as[len(as)-1])))
			}
		}

		// size: li:nth-child(5) span
		var size string
		if len(lis) >= 5 {
			size = strings.TrimSpace(htmlx.Text(lis[4]))
		}

		// createdAt: li:nth-child(2) span
		var createdAt time.Time
		if len(lis) >= 2 {
			raw := strings.TrimSpace(htmlx.Text(lis[1]))
			if t, err := time.Parse("2006/01/02 15:04", raw); err == nil {
				createdAt = toShanghai(t)
			}
		}

		// publisher / fansub from .topics_bk
		var publisher, fansub *Party
		for _, bk := range htmlx.Classes(doc, "topics_bk") {
			avatars := htmlx.Classes(bk, "avatar")
			ps := htmlx.Tag(bk, "p")
			for i, av := range avatars {
				if len(ps) < 2 {
					break
				}
				var avatar string
				imgs := htmlx.Tag(av, "img")
				if len(imgs) > 0 {
					avatar = htmlx.Attr(imgs[0], "src")
				}
				as := htmlx.Tag(ps[1], "a")
				if len(as) == 0 {
					break
				}
				party := &Party{
					ID:     lastSegment(htmlx.Attr(as[0], "href")),
					Name:   strings.TrimSpace(htmlx.Text(as[0])),
					Avatar: avatar,
				}
				if i == 0 {
					if party.Avatar == "" {
						party.Avatar = "https://share.dmhy.org/images/defaultUser.png"
					}
					publisher = party
				} else {
					if party.Avatar == "" {
						party.Avatar = "https://share.dmhy.org/images/defaultTeam.gif"
					}
					fansub = party
				}
			}
		}

		// description
		var description string
		for _, nfo := range htmlx.Classes(doc, "topic-nfo") {
			description = htmlx.InnerHTML(nfo)
			break
		}

		// magnets
		var magnets []MagnetEntry
		magnet1 := cssFind(doc, "#resource-tabs #tabs-1 p:nth-child(1) a")
		if len(magnet1) > 0 {
			href := htmlx.Attr(magnet1[0], "href")
			if !strings.HasPrefix(href, "https://") {
				href = "https:" + href
			}
			magnets = append(magnets, MagnetEntry{Name: "会员专用链接", URL: href})
		}
		if a := htmlx.First(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && htmlx.Attr(n, "id") == "a_magnet" }); a != nil {
			magnets = append(magnets, MagnetEntry{Name: "磁力链接", URL: htmlx.Attr(a, "href")})
		}
		if a := htmlx.First(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && htmlx.Attr(n, "id") == "magnet2" }); a != nil {
			magnets = append(magnets, MagnetEntry{Name: "磁力链接 type II", URL: htmlx.Attr(a, "href")})
		}
		if a := htmlx.First(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && htmlx.Attr(n, "id") == "ddplay" }); a != nil {
			magnets = append(magnets, MagnetEntry{Name: "弹幕播放链接", URL: strings.TrimSpace(htmlx.Text(a))})
		}

		// files
		var files []FileEntry
		hasMoreFiles := false
		for _, li := range htmlx.Classes(doc, "file_list") {
			for _, item := range htmlx.Tag(li, "li") {
				spans := htmlx.Tag(item, "span")
				if len(spans) == 0 {
					continue
				}
				sizeText := strings.TrimSpace(htmlx.Text(spans[0]))
				if sizeText == "種子可能不存在" || sizeText == "Bytes" {
					continue
				}
				name := strings.TrimSpace(htmlx.Text(item))
				name = strings.TrimSuffix(name, sizeText)
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				if strings.Contains(name, "More Than ") && strings.Contains(name, "Files") {
					hasMoreFiles = true
					continue
				}
				files = append(files, FileEntry{Name: name, Size: sizeText})
			}
			break
		}

		title = titleCleanup(title, fansubName(fansub))

		return &ScrapedResourceDetail{
			ScrapedResource: ScrapedResource{
				Provider:   DmhyName,
				ProviderID: id,
				Title:      title,
				Href:       id,
				Type:       typ,
				Size:       size,
				Publisher:  publisher,
				Fansub:     fansub,
				CreatedAt:  createdAt,
			},
			Description:  description,
			Files:        files,
			Magnets:      magnets,
			HasMoreFiles: hasMoreFiles,
		}, nil
	})
}

// cssFind implements a tiny subset of querySelectorAll for the selectors used
// by the dmhy detail parser.
func cssFind(doc *html.Node, selector string) []*html.Node {
	parts := strings.Fields(selector)
	if len(parts) == 0 {
		return nil
	}
	candidates := []*html.Node{doc}
	for _, part := range parts {
		var next []*html.Node
		for _, c := range candidates {
			var nodes []*html.Node
			if strings.HasPrefix(part, "#") {
				nodes = htmlx.ID(c, part[1:])
			} else if strings.HasPrefix(part, ".") {
				nodes = htmlx.Classes(c, part[1:])
			} else {
				tag := part
				var pseudo string
				if idx := strings.IndexByte(part, ':'); idx != -1 {
					tag = part[:idx]
					pseudo = part[idx:]
				}
				nodes = htmlx.Tag(c, tag)
				if pseudo == ":first-child" {
					nodes = nodes[:1]
				} else if pseudo == ":last-of-type" {
					if len(nodes) > 0 {
						nodes = nodes[len(nodes)-1:]
					}
				}
			}
			next = append(next, nodes...)
		}
		candidates = next
	}
	return candidates
}

func hasAncestorClass(n *html.Node, class string) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode {
			for _, a := range p.Attr {
				if a.Key == "class" {
					for _, c := range strings.Fields(a.Val) {
						if c == class {
							return true
						}
					}
				}
			}
		}
	}
	return false
}


