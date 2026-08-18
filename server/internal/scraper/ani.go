package scraper

import (
	"crypto/sha1"
	"encoding/base32"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/scraper/htmlx"
	"golang.org/x/net/html"
)

// ani constants, mirroring packages/scraper/src/ani.
const (
	AniName      = "ani"
	AniDisplayName = "ANi"
	AniRSSURL    = "https://api.ani.rip/ani-torrent.xml"
	AniBase      = "https://nyaa.si"
)

type aniRSS struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			PubDate     string `xml:"pubDate"`
			Link        string `xml:"link"`
			Enclosure   struct {
				URL    string `xml:"url,attr"`
				Length string `xml:"length,attr"`
			} `xml:"enclosure"`
		} `xml:"item"`
	} `xml:"channel"`
}

// FetchAniLatest fetches the latest ANi RSS feed, mirroring
// fetchAniPage from packages/scraper/src/ani/index.ts (single feed, no pages).
func FetchAniLatest(retry int) ([]ScrapedResource, error) {
	return retryFn(retry, func() ([]ScrapedResource, error) {
		resp, err := fetchURL("GET", AniRSSURL, MoeUA, nil, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var feed aniRSS
		if err := xml.Unmarshal(body, &feed); err != nil {
			return nil, err
		}
		var out []ScrapedResource
		for _, item := range feed.Channel.Items {
			r, err := parseAniItem(item)
			if err != nil {
				continue
			}
			out = append(out, r)
		}
		return out, nil
	})
}

func parseAniItem(item aniRSSItem) (ScrapedResource, error) {
	if item.Title == "" || item.PubDate == "" || item.Enclosure.Length == "" || !strings.HasSuffix(item.Link, ".torrent") {
		return ScrapedResource{}, fmt.Errorf("invalid ani item")
	}

	pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
	if err != nil {
		pubDate, err = time.Parse(time.RFC1123, item.PubDate)
		if err != nil {
			return ScrapedResource{}, err
		}
	}

	// download .torrent and parse info hash
	torrentData, err := retryFn(5, func() ([]byte, error) {
		resp, err := fetchURL("GET", item.Link, MoeUA, nil, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	})
	if err != nil {
		return ScrapedResource{}, err
	}
	torrent, err := parseTorrent(torrentData)
	if err != nil {
		return ScrapedResource{}, err
	}
	magnet := torrent.toMagnetURI()
	magnetOnly, _ := splitOnce(magnet, "&")

	size := formatSize(parseSize(item.Enclosure.Length))

	providerID := ""
	if strings.HasPrefix(item.Link, "https://tds.ani.rip/") {
		providerID = strings.TrimPrefix(magnetOnly, "magnet:?xt=urn:btih:")
	} else {
		filename := lastSegment(item.Link)
		providerID = filename[:len(filename)-len(".torrent")]
	}

	title := removeExtraSpaces(item.Title)
	title = replaceSuffix(title, map[string]string{
		".torrent": "[MP4]",
		".mp4":     "[MP4]",
		".MP4":     "[MP4]",
		".mkv":     "[MKV]",
		".MKV":     "[MKV]",
	})
	// transformAlias: parse with anipar(fansub=ANi) and rewrite A - B pairs
	title = transformAniAlias(title)

	title = titleCleanup(title, "ANi")

	return ScrapedResource{
		Provider:   AniName,
		ProviderID: providerID,
		Title:      title,
		Href:       item.Link,
		Type:       "动画",
		Magnet:     magnetOnly,
		Tracker:    "",
		Size:       size,
		Publisher:  &Party{ID: "1", Name: "ANi"},
		Fansub:     &Party{ID: "1", Name: "ANi"},
		CreatedAt:  pubDate.UTC(),
	}, nil
}

// transformAniAlias is implemented after the anipar port lands; for now it
// keeps the title unchanged. The anipar package exposes AliasRewrite.
var transformAniAlias = func(title string) string { return title }

// SetAliasRewriter wires the anipar-based alias transform (set at startup).
func SetAliasRewriter(fn func(string) string) {
	if fn != nil {
		transformAniAlias = fn
	}
}

// FetchAniDetail fetches a nyaa.si view page, mirroring fetchAniDetail.
func FetchAniDetail(id string, retry int) (*ScrapedResourceDetail, error) {
	return retryFn(retry, func() (*ScrapedResourceDetail, error) {
		url := fmt.Sprintf("%s/view/%s", AniBase, id)
		resp, err := fetchURL("GET", url, MoeUA, nil, nil)
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

		titleNode := htmlx.First(doc, func(n *html.Node) bool {
			return n.Type == html.ElementNode && htmlx.HasClass(n, "panel-title") && hasAncestorClass(n, "panel-heading")
		})
		if titleNode == nil {
			return nil, nil
		}
		title := strings.TrimSpace(htmlx.Text(titleNode))

		description := ""
		for _, el := range htmlx.ID(doc, "torrent-description") {
			description = htmlx.InnerHTML(el)
			break
		}

		// createdAt: data-timestamp of .panel-body .col-md-5:last-child
		var createdAt time.Time
		panelBodies := htmlx.Classes(doc, "panel-body")
		for _, pb := range panelBodies {
			cols := htmlx.Classes(pb, "col-md-5")
			if len(cols) == 0 {
				continue
			}
			col := cols[len(cols)-1]
			if ts := htmlx.Attr(col, "data-timestamp"); ts != "" {
				if sec, err := strconv.ParseInt(ts, 10, 64); err == nil {
					createdAt = time.Unix(sec, 0).UTC()
				}
			}
		}

		// magnet: .panel-footer a.card-footer-item
		magnet := ""
		for _, pb := range htmlx.Classes(doc, "panel-footer") {
			for _, a := range htmlx.Tag(pb, "a") {
				if htmlx.HasClass(a, "card-footer-item") {
					if href := htmlx.Attr(a, "href"); strings.HasPrefix(href, "magnet:") {
						magnet, _ = splitOnce(href, "&")
					}
				}
			}
		}

		// size: .panel-body .row:nth-child(4) .col-md-5 text
		size := ""
		for _, pb := range panelBodies {
			rows := htmlx.Tag(pb, "row")
			_ = rows
		}
		{
			var rows []*html.Node
			for _, pb := range panelBodies {
				rows = append(rows, htmlx.Tag(pb, "div")...)
			}
			// rows are divs; find the 4th row with class row
			var rowDivs []*html.Node
			for _, n := range rows {
				if htmlx.HasClass(n, "row") {
					rowDivs = append(rowDivs, n)
				}
			}
			if len(rowDivs) >= 4 {
				cols := htmlx.Classes(rowDivs[3], "col-md-5")
				if len(cols) > 0 {
					size = strings.TrimSpace(htmlx.Text(cols[0]))
				}
			}
		}

		// files
		var files []FileEntry
		for _, list := range htmlx.Classes(doc, "torrent-file-list") {
			for _, li := range htmlx.Tag(list, "li") {
				name := ""
				childNodes := []*html.Node{}
				for c := li.FirstChild; c != nil; c = c.NextSibling {
					childNodes = append(childNodes, c)
				}
				if len(childNodes) > 1 {
					name = strings.TrimSpace(htmlx.Text(childNodes[1]))
				} else if len(childNodes) == 1 {
					name = strings.TrimSpace(htmlx.Text(childNodes[0]))
				}
				fSize := ""
				for _, el := range htmlx.Classes(li, "file-size") {
					t := strings.TrimSpace(htmlx.Text(el))
					if len(t) >= 2 {
						t = t[1 : len(t)-1]
					}
					fSize = t
					break
				}
				if name == "" {
					continue
				}
				files = append(files, FileEntry{Name: name, Size: fSize})
			}
			break
		}

		title = titleCleanup(title, "ANi")

		magnets := []MagnetEntry{
			{Name: "种子", URL: fmt.Sprintf("%s/download/%s.torrent", AniBase, id)},
		}
		if magnet != "" {
			magnets = append(magnets, MagnetEntry{Name: "磁力链接", URL: magnet})
		}

		return &ScrapedResourceDetail{
			ScrapedResource: ScrapedResource{
				Provider:   AniName,
				ProviderID: id,
				Title:      title,
				Href:       id,
				Type:       "动画",
				Magnet:     magnet,
				Size:       size,
				Publisher:  &Party{ID: "1", Name: "ANi"},
				Fansub:     &Party{ID: "1", Name: "ANi"},
				CreatedAt:  createdAt,
			},
			Description:  description,
			Files:        files,
			Magnets:      magnets,
			HasMoreFiles: false,
		}, nil
	})
}

type aniRSSItem = struct {
	Title     string `xml:"title"`
	PubDate   string `xml:"pubDate"`
	Link      string `xml:"link"`
	Enclosure struct {
		URL    string `xml:"url,attr"`
		Length string `xml:"length,attr"`
	} `xml:"enclosure"`
}

// --- bencode torrent parsing (minimal, for info hash extraction) ---

type bencodeValue struct {
	kind  byte // 'i' int, 's' string, 'l' list, 'd' dict
	i     int64
	s     []byte
	list  []bencodeValue
	dict  map[string]bencodeValue
}

type bencodeParser struct {
	data []byte
	pos  int
}

func (p *bencodeParser) parse() (bencodeValue, error) {
	if p.pos >= len(p.data) {
		return bencodeValue{}, fmt.Errorf("bencode: unexpected end")
	}
	switch p.data[p.pos] {
	case 'i':
		p.pos++
		start := p.pos
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			p.pos++
		}
		if p.pos >= len(p.data) {
			return bencodeValue{}, fmt.Errorf("bencode: unterminated int")
		}
		n, err := strconv.ParseInt(string(p.data[start:p.pos]), 10, 64)
		if err != nil {
			return bencodeValue{}, err
		}
		p.pos++
		return bencodeValue{kind: 'i', i: n}, nil
	case 'l':
		p.pos++
		var out []bencodeValue
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			v, err := p.parse()
			if err != nil {
				return bencodeValue{}, err
			}
			out = append(out, v)
		}
		p.pos++
		return bencodeValue{kind: 'l', list: out}, nil
	case 'd':
		p.pos++
		out := map[string]bencodeValue{}
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			key, err := p.parse()
			if err != nil {
				return bencodeValue{}, err
			}
			val, err := p.parse()
			if err != nil {
				return bencodeValue{}, err
			}
			out[string(key.s)] = val
		}
		p.pos++
		return bencodeValue{kind: 'd', dict: out}, nil
	default:
		// string: <len>:<data>
		colon := -1
		for i := p.pos; i < len(p.data); i++ {
			if p.data[i] == ':' {
				colon = i
				break
			}
		}
		if colon == -1 {
			return bencodeValue{}, fmt.Errorf("bencode: invalid string")
		}
		n, err := strconv.Atoi(string(p.data[p.pos:colon]))
		if err != nil {
			return bencodeValue{}, err
		}
		if colon+1+n > len(p.data) {
			return bencodeValue{}, fmt.Errorf("bencode: truncated string")
		}
		s := p.data[colon+1 : colon+1+n]
		p.pos = colon + 1 + n
		return bencodeValue{kind: 's', s: s}, nil
	}
}

// torrentInfo holds the parsed torrent metadata.
type torrentInfo struct {
	name      string
	infoHash  string // lowercase hex
}

// parseTorrent parses bencode and computes the v1 info hash.
func parseTorrent(data []byte) (*torrentInfo, error) {
	p := &bencodeParser{data: data}
	root, err := p.parse()
	if err != nil {
		return nil, err
	}
	if root.kind != 'd' {
		return nil, fmt.Errorf("bencode: root not a dict")
	}
	info, ok := root.dict["info"]
	if !ok || info.kind != 'd' {
		return nil, fmt.Errorf("bencode: missing info dict")
	}
	// re-serialize the info dict bytes for hashing
	infoStart, infoEnd, err := findInfoRange(data)
	if err != nil {
		return nil, err
	}
	sum := sha1.Sum(data[infoStart:infoEnd])

	t := &torrentInfo{infoHash: hex.EncodeToString(sum[:])}
	if name, ok := info.dict["name"]; ok {
		t.name = string(name.s)
	}
	return t, nil
}

// findInfoRange locates the byte range of the "info" value in the top-level
// dict (first occurrence of 4:info at the root level).
func findInfoRange(data []byte) (int, int, error) {
	// scan for "4:info" — the top-level dict starts with 'd'
	idx := 0
	for i := 0; i+6 <= len(data); i++ {
		if data[i] == '4' && data[i+1] == ':' && string(data[i+2:i+6]) == "info" {
			// must be followed by a value
			valStart := i + 6
			if valStart >= len(data) {
				return 0, 0, fmt.Errorf("bencode: no info value")
			}
			// parse the value to find its end
			p := &bencodeParser{data: data, pos: valStart}
			if _, err := p.parse(); err != nil {
				return 0, 0, err
			}
			_ = idx
			return valStart, p.pos, nil
		}
	}
	return 0, 0, fmt.Errorf("bencode: info key not found")
}

var base32NoPad = base32.StdEncoding.WithPadding(base32.NoPadding)

// toMagnetURI mirrors parse-torrent's toMagnetURI: xt=urn:btih:<lower hex>,
// &dn=<encoded name> (name empty -> omitted per parse-torrent behavior:
// parse-torrent includes dn only when the name exists).
func (t *torrentInfo) toMagnetURI() string {
	var b strings.Builder
	b.WriteString("magnet:?xt=urn:btih:")
	b.WriteString(t.infoHash)
	if t.name != "" {
		b.WriteString("&dn=")
		b.WriteString(url.QueryEscape(t.name))
	}
	return b.String()
}

// extractBtihFromMagnet ports extractBtihFromMagnet: returns the first btih
// found among multiple xt= params, normalized-aware.
func extractBtihFromMagnet(magnetURL string) (format, value string, ok bool) {
	magnetURL = strings.TrimSpace(magnetURL)
	if !strings.HasPrefix(strings.ToLower(magnetURL), "magnet:?") {
		return "", "", false
	}
	query := magnetURL[len("magnet:?"):]
	values, err := url.ParseQuery(query)
	if err != nil {
		return "", "", false
	}
	for _, xt := range values["xt"] {
		m := regexp.MustCompile(`^urn:btih:([a-zA-Z0-9]+)$`).FindStringSubmatch(xt)
		if m == nil {
			continue
		}
		v := m[1]
		if hexRe.MatchString(v) {
			return "hex", strings.ToLower(v), true
		}
		if b32Re.MatchString(v) {
			return "base32", strings.ToUpper(v), true
		}
		return "", "", false
	}
	return "", "", false
}

var hexRe = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
var b32Re = regexp.MustCompile(`^[A-Z2-7]{32}$`)

// normalizeBtihToHex ports normalizeBtihToHex: base32 input is converted and
// the magnet URI is rebuilt with only the xt param.
func NormalizeBtihToHex(magnetURL string) string {
	r, v, ok := extractBtihFromMagnet(magnetURL)
	if !ok {
		return magnetURL
	}
	if r == "hex" {
		return magnetURL
	}
	raw, err := base32NoPad.DecodeString(v)
	if err != nil {
		return magnetURL
	}
	return "magnet:?xt=urn:btih:" + hex.EncodeToString(raw)
}

// normalizeBtihToBase32 ports normalizeBtihToBase32.
func NormalizeBtihToBase32(magnetURL string) string {
	r, v, ok := extractBtihFromMagnet(magnetURL)
	if !ok {
		return magnetURL
	}
	if r == "base32" {
		return magnetURL
	}
	raw, err := hex.DecodeString(v)
	if err != nil {
		return magnetURL
	}
	return "magnet:?xt=urn:btih:" + base32NoPad.EncodeToString(raw)
}
