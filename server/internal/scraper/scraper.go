// Package scraper ports packages/scraper from the original AnimeGarden:
// fetch + parse functions for the four providers (dmhy, moe, mikan, ani).
package scraper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ScrapedResource mirrors ScrapedResource in @animegarden/client.
type ScrapedResource struct {
	Provider   string
	ProviderID string
	Title      string
	Href       string
	Type       string
	Magnet     string
	Tracker    string
	Size       string // always a string here
	Publisher  *Party
	Fansub     *Party
	CreatedAt  time.Time
}

// Party mirrors the publisher/fansub shape of a scraped resource.
type Party struct {
	ID     string
	Name   string
	Avatar string
}

// ScrapedResourceDetail mirrors ScrapedResourceDetail.
type ScrapedResourceDetail struct {
	ScrapedResource
	Description  string
	Files        []FileEntry
	Magnets      []MagnetEntry
	HasMoreFiles bool
}

type FileEntry struct {
	Name string
	Size string
}

type MagnetEntry struct {
	Name string
	URL  string
}

// NetworkError marks a failed upstream request (bad status), mirroring the
// original NetworkError used to drive provider inactive-state handling.
type NetworkError struct {
	Provider string
	URL      string
	Status   int
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error (%s): GET %s -> %d", e.Provider, e.URL, e.Status)
}

// IsNetworkError reports whether err is a NetworkError.
func IsNetworkError(err error) bool {
	var ne *NetworkError
	return errors.As(err, &ne)
}

// Client is the shared HTTP client. No rate limiting, mirroring the original.
var Client = &http.Client{Timeout: 60 * time.Second}

// BrowserUA mirrors the moe scraper UA; used across providers so Cloudflare
// does not treat bare Go requests as bots.
const BrowserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0"

// fetchURL performs a GET with the given UA and returns the response body
// reader; it converts bad statuses to NetworkError.
func fetchURL(method, url, ua string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, &NetworkError{URL: url, Status: resp.StatusCode}
	}
	return resp, nil
}

// retryFn mirrors the original retryFn: re-invokes, count+1 times total, on
// error. A short pause is inserted between network-error retries so that
// upstream rate limits (429) have a chance to clear.
func retryFn[T any](count int, fn func() (T, error)) (T, error) {
	var lastErr error
	for i := 0; i <= count; i++ {
		v, err := fn()
		if err == nil {
			return v, nil
		}
		lastErr = err
		if i < count && IsNetworkError(err) {
			time.Sleep(2 * time.Second)
		}
	}
	var zero T
	return zero, lastErr
}

var zeroWidthRe = regexp.MustCompile("[\u200B-\u200D\uFEFF]")

// removeExtraSpaces mirrors removeExtraSpaces from @animegarden/shared.
func removeExtraSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// stripSuffix mirrors stripSuffix from @animegarden/shared.
func stripSuffix(s string, suffixes []string) string {
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return s[:len(s)-len(suf)]
		}
	}
	return s
}

// replaceSuffix mirrors replaceSuffix from @animegarden/shared.
func replaceSuffix(s string, suffixes map[string]string) string {
	for suf, replaced := range suffixes {
		if strings.HasSuffix(s, suf) {
			return s[:len(s)-len(suf)] + replaced
		}
	}
	return s
}

// splitOnce mirrors splitOnce from @animegarden/shared.
func splitOnce(text, separator string) (string, string) {
	idx := strings.Index(text, separator)
	if idx == -1 {
		return text, ""
	}
	return text[:idx], text[idx:]
}

// titleCleanup is the pre-cleanup pipeline shared across providers.
func titleCleanup(title, fansub string) string {
	title = zeroWidthRe.ReplaceAllString(title, "")
	title = strings.ReplaceAll(title, "[[", "[")
	title = strings.ReplaceAll(title, "]]", "]")

	if fansub == "ANi" || fansub == "云光字幕组" {
		title = removeExtraSpaces(title)
		title = stripSuffix(title, []string{".torrent", ".mp3", ".MP3", ".mp4", ".MP4", ".mkv", ".MKV"})
	} else {
		title = removeExtraSpaces(title)
	}

	title = stripSuffix(title, []string{"v2"})
	return title
}

// sizeRe patterns for parseSize.
var (
	kbRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[Kk]i?[Bb]$`)
	mbRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[Mm]i?[Bb]$`)
	gbRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[Gg]i?[Bb]$`)
	tbRe = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[Tt]i?[Bb]$`)
)

// parseSize ports parseSize in resources/transform.ts: returns bytes.
func parseSize(size string) int64 {
	if size == "" {
		return 0
	}
	parse := func(re *regexp.Regexp) (float64, bool) {
		m := re.FindStringSubmatch(size)
		if m == nil {
			return 0, false
		}
		var f float64
		fmt.Sscanf(m[1], "%f", &f)
		return f, true
	}
	if f, ok := parse(kbRe); ok {
		return int64(f)
	}
	if f, ok := parse(mbRe); ok {
		return int64(f * 1024)
	}
	if f, ok := parse(gbRe); ok {
		return int64(f * 1024 * 1024)
	}
	if f, ok := parse(tbRe); ok {
		return int64(f * 1024 * 1024 * 1024)
	}
	var n int64
	if _, err := fmt.Sscanf(size, "%d", &n); err == nil {
		return n
	}
	return 0
}

// formatSize ports the ANi parseSize formatting (<1KB -> 'n B', <1MB -> 'n.n KB', ...).
func formatSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
	}
	return fmt.Sprintf("%.1f GB", float64(n)/1024/1024/1024)
}

// shanghaiLocation is Asia/Shanghai.
var shanghaiLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("UTC+8", 8*60*60)
	}
	return loc
}()
