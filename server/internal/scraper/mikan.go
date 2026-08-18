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

// mikan constants, mirroring packages/scraper/src/mikan.
const (
	MikanBase      = "https://mikanani.kas.pub"
	MikanName      = "mikan"
	MikanDisplayName = "蜜柑计划"
)

// parseMikanDate ports parseMikanDate from the original: supports
// 今天/昨天/absolute/yearless formats; all in Asia/Shanghai.
func parseMikanDate(raw string, now time.Time) (time.Time, error) {
	text := strings.TrimSpace(raw)
	text = regexp.MustCompile(`^发布日期[:：]\s*`).ReplaceAllString(text, "")
	loc := shanghaiLocation

	if m := regexp.MustCompile(`^(今天|昨天)\s+(\d{1,2}):(\d{2})$`).FindStringSubmatch(text); m != nil {
		var hh, mm int
		fmt.Sscanf(m[2], "%d", &hh)
		fmt.Sscanf(m[3], "%d", &mm)
		day := now.In(loc)
		if m[1] == "昨天" {
			day = day.AddDate(0, 0, -1)
		}
		return time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, loc).UTC(), nil
	}
	if m := regexp.MustCompile(`^(\d{4})[-/.年](\d{1,2})[-/.月](\d{1,2})(?:日)?(?:\s+(\d{1,2}):(\d{2}))?$`).FindStringSubmatch(text); m != nil {
		var y, mo, d, hh, mm int
		fmt.Sscanf(m[1], "%d", &y)
		fmt.Sscanf(m[2], "%d", &mo)
		fmt.Sscanf(m[3], "%d", &d)
		if m[4] != "" {
			fmt.Sscanf(m[4], "%d", &hh)
			fmt.Sscanf(m[5], "%d", &mm)
		}
		return time.Date(y, time.Month(mo), d, hh, mm, 0, 0, loc).UTC(), nil
	}
	if m := regexp.MustCompile(`^(\d{1,2})[-/.月](\d{1,2})(?:日)?\s+(\d{1,2}):(\d{2})$`).FindStringSubmatch(text); m != nil {
		var mo, d, hh, mm int
		fmt.Sscanf(m[1], "%d", &mo)
		fmt.Sscanf(m[2], "%d", &d)
		fmt.Sscanf(m[3], "%d", &hh)
		fmt.Sscanf(m[4], "%d", &mm)
		year := now.In(loc).Year()
		return time.Date(year, time.Month(mo), d, hh, mm, 0, 0, loc).UTC(), nil
	}
	t, err := time.Parse(time.RFC3339Nano, text)
	if err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse mikan date: %s", raw)
}

// FetchMikanPage fetches one mikan classic list page, mirroring
// fetchMikanPage from packages/scraper/src/mikan/index.ts.
func FetchMikanPage(page int, retry int) ([]ScrapedResource, error) {
	return retryFn(retry, func() ([]ScrapedResource, error) {
		url := fmt.Sprintf("%s/Home/Classic/%d", MikanBase, page)
		resp, err := fetchURL("GET", url, BrowserUA, nil, nil)
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

		now := time.Now()
		var out []ScrapedResource
		for _, table := range htmlx.Tag(doc, "table") {
			if !htmlx.HasClass(table, "table") {
				continue
			}
			for _, tr := range htmlx.Tag(table, "tr") {
				tds := htmlx.Tag(tr, "td")
				if len(tds) < 4 {
					continue
				}
				r, ok := parseMikanRow(tds, now)
				if ok && r.Fansub != nil && r.Publisher != nil {
					out = append(out, r)
				}
			}
			break
		}
		return out, nil
	})
}

func parseMikanRow(tds []*html.Node, now time.Time) (ScrapedResource, bool) {
	// td[0] date
	createdAt, err := parseMikanDate(htmlx.Text(tds[0]), now)
	if err != nil {
		return ScrapedResource{}, false
	}

	// td[1] publish group -> publisher + fansub
	var party *Party
	for _, a := range htmlx.Tag(tds[1], "a") {
		href := htmlx.Attr(a, "href")
		if !strings.Contains(href, "/Home/PublishGroup/") {
			continue
		}
		m := regexp.MustCompile(`^/Home/PublishGroup/([^/?#]+)`).FindStringSubmatch(href)
		if m == nil {
			continue
		}
		party = &Party{ID: m[1], Name: strings.TrimSpace(htmlx.Text(a))}
		break
	}
	if party == nil {
		return ScrapedResource{}, false
	}

	// td[2] episode link
	var title, providerID, magnet string
	for _, a := range htmlx.Tag(tds[2], "a") {
		href := htmlx.Attr(a, "href")
		if !strings.Contains(href, "/Home/Episode/") {
			continue
		}
		title = strings.TrimSpace(htmlx.Text(a))
		rest := href[strings.Index(href, "/Home/Episode/")+len("/Home/Episode/"):]
		rest = strings.TrimPrefix(rest, "/")
		providerID = rest
		break
	}
	// the magnet is carried on an element with data-clipboard-text (an <a>)
	for _, el := range htmlx.Filter(tds[2], func(n *html.Node) bool {
		return n.Type == html.ElementNode && htmlx.Attr(n, "data-clipboard-text") != ""
	}) {
		if v := htmlx.Attr(el, "data-clipboard-text"); v != "" {
			magnet, _ = splitOnce(v, "&")
			break
		}
	}
	if title == "" || providerID == "" {
		return ScrapedResource{}, false
	}

	// td[3] size
	size := strings.TrimSpace(htmlx.Text(tds[3]))

	title = titleCleanup(title, party.Name)

	return ScrapedResource{
		Provider:   MikanName,
		ProviderID: providerID,
		Title:      title,
		Href:       providerID,
		Type:       "动画",
		Magnet:     magnet,
		Size:       size,
		Publisher:  &Party{ID: party.ID, Name: party.Name},
		Fansub:     &Party{ID: party.ID, Name: party.Name},
		CreatedAt:  createdAt,
	}, true
}

// FetchMikanDetail fetches a mikan episode detail page, mirroring
// fetchMikanDetail from packages/scraper/src/mikan/index.ts.
func FetchMikanDetail(id string, retry int) (*ScrapedResourceDetail, error) {
	return retryFn(retry, func() (*ScrapedResourceDetail, error) {
		url := fmt.Sprintf("%s/Home/Episode/%s", MikanBase, id)
		resp, err := fetchURL("GET", url, BrowserUA, nil, nil)
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

		// title from <title>
		title := ""
		titles := htmlx.Tag(doc, "title")
		if len(titles) > 0 {
			title = strings.TrimSpace(htmlx.Text(titles[0]))
			title = strings.TrimSuffix(title, " - Mikan Project")
		}
		if title == "" {
			for _, el := range htmlx.Classes(doc, "episode-title") {
				title = strings.TrimSpace(htmlx.Text(el))
				break
			}
		}
		if title == "" {
			return nil, nil
		}

		// description
		description := ""
		for _, el := range htmlx.Classes(doc, "episode-desc") {
			// remove ad divs
			for _, div := range htmlx.Tag(el, "div") {
				remove := false
				for _, img := range htmlx.Tag(div, "img") {
					if strings.Contains(htmlx.Attr(img, "src"), "/images/SSWJ/") {
						remove = true
					}
				}
				for _, a := range htmlx.Tag(div, "a") {
					if strings.Contains(htmlx.Attr(a, "href"), "equity.tmall.com") {
						remove = true
					}
				}
				if remove {
					div.Parent.RemoveChild(div)
				}
			}
			description = htmlx.InnerHTML(el)
			break
		}

		// createdAt / size from .bangumi-info
		var createdAt time.Time
		var size string
		for _, el := range htmlx.Classes(doc, "bangumi-info") {
			text := strings.TrimSpace(htmlx.Text(el))
			if strings.HasPrefix(text, "发布日期") {
				if t, err := parseMikanDate(text, time.Now()); err == nil {
					createdAt = t
				}
			} else if strings.HasPrefix(text, "文件大小") {
				size = text
			}
		}

		// group from .leftbar-container
		var party *Party
		leftbar := htmlx.First(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && htmlx.HasClass(n, "leftbar-container") })
		root := doc
		if leftbar != nil {
			root = leftbar
		}
		for _, a := range htmlx.Tag(root, "a") {
			href := htmlx.Attr(a, "href")
			if !strings.Contains(href, "/Home/PublishGroup/") {
				continue
			}
			m := regexp.MustCompile(`^/Home/PublishGroup/([^/?#]+)`).FindStringSubmatch(href)
			if m == nil {
				continue
			}
			party = &Party{ID: m[1], Name: strings.TrimSpace(htmlx.Text(a))}
			break
		}

		// magnet / torrent
		var magnets []MagnetEntry
		var magnet string
		for _, a := range htmlx.Tag(root, "a") {
			href := htmlx.Attr(a, "href")
			if strings.HasPrefix(href, "magnet:") {
				magnet, _ = splitOnce(href, "&")
				continue
			}
			if strings.HasSuffix(href, ".torrent") {
				magnetLink := strings.TrimPrefix(href, "/")
				if !strings.HasPrefix(magnetLink, "http") {
					magnetLink = MikanBase + "/" + magnetLink
				}
				magnets = append(magnets, MagnetEntry{Name: "种子", URL: magnetLink})
			}
		}
		if magnet != "" {
			magnets = append(magnets, MagnetEntry{Name: "磁力链接", URL: magnet})
		}

		title = titleCleanup(title, partyName(party))

		return &ScrapedResourceDetail{
			ScrapedResource: ScrapedResource{
				Provider:   MikanName,
				ProviderID: id,
				Title:      title,
				Href:       id,
				Type:       "动画",
				Magnet:     magnet,
				Size:       size,
				Publisher:  party,
				Fansub:     party,
				CreatedAt:  createdAt,
			},
			Description:  description,
			Files:        nil,
			Magnets:      magnets,
			HasMoreFiles: false,
		}, nil
	})
}

func partyName(p *Party) string {
	if p == nil {
		return ""
	}
	return p.Name
}

