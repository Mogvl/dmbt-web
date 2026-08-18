// Package rss ports apps/server/src/server/rss: RSS 2.0 generation for
// /feed.xml and collection feeds.
package rss

import (
	"fmt"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/filter"
	"github.com/Mogvl/dmbt-web/server/internal/model"
)

// Item is one RSS item.
type Item struct {
	Title     string
	PubDate   time.Time // Shanghai wall-clock of createdAt
	Link      string
	Enclosure struct {
		URL    string
		Length int64
		Type   string
	}
}

// Feed is the RSS feed payload.
type Feed struct {
	Title       string
	Description string
	Site        string // e.g. https://animes.garden
	TrailingSlash bool
	Items       []Item
}

// Build generates the RSS 2.0 XML string, mirroring the original builder
// (compact, self-closing empty nodes).
func Build(feed Feed) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel>`)
	b.WriteString("<title>")
	xmlEscape(&b, feed.Title)
	b.WriteString("</title><description>")
	xmlEscape(&b, feed.Description)
	b.WriteString("</description><link>")
	xmlEscape(&b, feed.Site)
	b.WriteString("</link>")

	for _, item := range feed.Items {
		b.WriteString("<item><title>")
		xmlEscape(&b, item.Title)
		b.WriteString("</title><link>")
		xmlEscape(&b, item.Link)
		b.WriteString("</link><guid isPermaLink=\"true\">")
		xmlEscape(&b, item.Link)
		b.WriteString("</guid><pubDate>")
		b.WriteString(item.PubDate.In(time.UTC).Format("Mon, 02 Jan 2006 15:04:05 GMT"))
		b.WriteString("</pubDate><enclosure url=\"")
		xmlEscape(&b, item.Enclosure.URL)
		fmt.Fprintf(&b, "\" length=\"%d\" type=\"%s\"/>", item.Enclosure.Length, item.Enclosure.Type)
		b.WriteString("</item>")
	}

	b.WriteString("</channel></rss>")
	return b.String()
}

func xmlEscape(b *strings.Builder, s string) {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	replacer.WriteString(b, s)
}

// GenerateTitleFromFilter ports generateTitleFromFilter from
// server/utils/meta.ts.
func GenerateTitleFromFilter(f filter.FilterOptions, subjectNames func(ids []int64) []string) string {
	if len(f.Subjects) > 0 {
		names := subjectNames(f.Subjects)
		if len(names) > 0 {
			return strings.Join(names, " ") + " 最新动画资源"
		}
	}
	if len(f.Search) > 0 {
		return strings.Join(f.Search, " ") + " 最新动画资源"
	}
	if len(f.Include) > 0 {
		return f.Include[0] + " 最新动画资源"
	}
	if len(f.Fansubs) == 1 {
		return f.Fansubs[0] + " 最新动画资源"
	}
	if len(f.Publishers) == 1 {
		return f.Publishers[0] + " 最新动画资源"
	}
	if len(f.Types) == 1 {
		return "最新" + f.Types[0] + "资源"
	}
	return "所有资源"
}

// ToItem renders one resource as an RSS item.
func ToItem(r model.Resource, site string, withTracker bool) Item {
	item := Item{
		Title:   r.Title,
		PubDate: r.CreatedAt,
		Link:    fmt.Sprintf("%s/detail/%s/%s", site, r.Provider, r.ProviderID),
	}
	item.Enclosure.URL = r.Magnet
	if withTracker && r.Tracker != nil {
		item.Enclosure.URL += *r.Tracker
	}
	item.Enclosure.Length = r.Size
	item.Enclosure.Type = "application/x-bittorrent"
	return item
}

// IsTrackerEnabled ports isTrackerEnabled: reads the typo'd 'trakcer' param;
// absent -> enabled; 'no'/'off'/'false' -> disabled.
func IsTrackerEnabled(values map[string][]string) bool {
	raw, ok := values["trakcer"]
	if !ok || len(raw) == 0 {
		return true
	}
	v := raw[0]
	return !(v == "no" || v == "off" || v == "false")
}