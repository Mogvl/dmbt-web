package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/filter"
	"github.com/Mogvl/dmbt-web/server/internal/model"
	"github.com/Mogvl/dmbt-web/server/internal/rss"
)

// handleFeed ports GET /feed.xml.
func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(405)
		return
	}
	// parse query params (+ optional JSON body for POST parity; GET has none)
	result := filter.ParseURLSearch(r.URL.Query(), nil)
	pagination, f := result.Pagination, result.Filter

	if !assertPagination(w, pagination) {
		return
	}

	find, err := s.sys.Query.Find(f, pagination.Page, pagination.PageSize)
	if err != nil {
		s.writeFeedError(w, "Resources query failed")
		return
	}

	site := "https://" + s.sys.Site
	withTracker := rss.IsTrackerEnabled(r.URL.Query())

	items := make([]rss.Item, 0, len(find.Resources))
	for _, res := range find.Resources {
		items = append(items, rss.ToItem(res, site, withTracker))
	}

	feed := rss.Feed{
		Title:       rss.GenerateTitleFromFilter(f, s.subjectNames),
		Description: "Anime Garden 是動漫花園資源網的第三方镜像站",
		Site:        site + "/resources/1" + rawSearch(r.URL.RawQuery),
		Items:       items,
	}
	body := rss.Build(feed)

	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(body))
}

// handleCollectionFeed ports GET /collection/:hash/feed.xml.
func (s *Server) handleCollectionFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(405)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/collection/")
	hash := strings.TrimSuffix(path, "/feed.xml")
	if hash == "" {
		s.errJSON(w, 400, "Missing collection hash")
		return
	}
	row, err := s.sys.Collections.GetRow(hash)
	if err != nil || row == nil {
		s.errJSON(w, 400, "Collection \""+hash+"\" not found")
		return
	}

	site := "https://" + s.sys.Site
	withTracker := rss.IsTrackerEnabled(r.URL.Query())

	var items []rss.Item
	for _, f := range row.FiltersRaw {
		options := FilterFromStoredMap(f)
		find, err := s.sys.Query.Find(options, 1, 1000)
		if err != nil {
			continue
		}
		for _, res := range find.Resources {
			items = append(items, rss.ToItem(res, site, withTracker))
		}
	}

	feed := rss.Feed{
		Title:       row.Name,
		Description: "Anime Garden 是動漫花園資源網的第三方镜像站.",
		Site:        site + "/collection/" + hash,
		Items:       items,
	}
	if feed.Title == "" {
		feed.Title = "收藏夹 " + hash
	}
	body := rss.Build(feed)

	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(body))
}

// subjectNames resolves subject ids to display names for the feed title.
func (s *Server) subjectNames(ids []int64) []string {
	var out []string
	for _, id := range ids {
		for _, sub := range s.sys.Subjects.AllSubjects {
			if sub.ID == id {
				out = append(out, sub.Name)
				break
			}
		}
	}
	return out
}

// rawSearch returns "?..." or "" for the given raw query.
func rawSearch(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	return "?" + rawQuery
}

// assertPagination ports assertResourcesPagination: 400 when too deep.
func assertPagination(w http.ResponseWriter, pagination filter.Pagination) bool {
	offset := (pagination.Page - 1) * pagination.PageSize
	if offset+pagination.PageSize > 10000 {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ERROR",
			"message": "Resources pagination is too deep. Please keep offset + limit <= 10000.",
		})
		return false
	}
	return true
}

// writeFeedError ports the XML error body for /feed.xml failures.
func (s *Server) writeFeedError(w http.ResponseWriter, message string) {
	escaped := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	).Replace(message)
	body := `<?xml version="1.0" encoding="UTF-8"?><error><message>` + escaped + `</message></error>`
	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(400)
	w.Write([]byte(body))
}

// --- sitemaps ---

// handleSitemapSubjects ports GET /sitemaps/subjects.
func (s *Server) handleSitemapSubjects(w http.ResponseWriter, r *http.Request) {
	subs := make([]map[string]any, 0, len(s.sys.Subjects.AllSubjects))
	for _, sub := range s.sys.Subjects.AllSubjects {
		subs = append(subs, map[string]any{
			"id":         sub.ID,
			"activedAt":  sub.ActivedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
			"isArchived": sub.IsArchived,
		})
	}
	s.json(w, 200, map[string]any{"status": "OK", "subjects": subs})
}

// handleSitemapMonth ports GET /sitemaps/:year/:month.
func (s *Server) handleSitemapMonth(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sitemaps/"), "/")
	year, err1 := strconv.Atoi(parts[0])
	month := 0
	if len(parts) > 1 {
		month, _ = strconv.Atoi(parts[1])
	}
	now := time.Now()
	if err1 != nil || year < 2020 || year > now.Year() || month < 1 || month > 12 ||
		(year == now.Year() && month > int(now.Month())) {
		s.json(w, 200, map[string]any{"status": "ERROR", "resources": []any{}})
		return
	}
	rows, err := s.sys.DB.Query(
		`SELECT id, provider_name, provider_id, fetched_at FROM resources
		 WHERE is_deleted = 0 AND duplicated_id IS NULL AND created_at >= ? AND created_at < ?`,
		dbFormatMonthStart(year, month), dbFormatMonthStart(nextMonth(year, month)))
	if err != nil {
		s.json(w, 200, map[string]any{"status": "ERROR", "resources": []any{}})
		return
	}
	defer rows.Close()
	type row struct {
		ID         int64  `json:"id"`
		Provider   string `json:"provider"`
		ProviderID string `json:"providerId"`
		FetchedAt  string `json:"fetchedAt"`
	}
	var resources []row
	for rows.Next() {
		var r2 row
		var fetchedAt string
		if err := rows.Scan(&r2.ID, &r2.Provider, &r2.ProviderID, &fetchedAt); err != nil {
			continue
		}
		r2.FetchedAt = fetchedAt
		resources = append(resources, r2)
	}
	s.json(w, 200, map[string]any{"status": "OK", "count": len(resources), "resources": resources})
}

// dbFormatMonthStart returns the Shanghai-midnight UTC instant for the month.
func dbFormatMonthStart(year, month int) string {
	// getShanghai: Date.UTC(y, m-1, 1) - 8h
	utc := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return utc.Add(-8 * time.Hour).Format("2006-01-02T15:04:05.000Z07:00")
}

func nextMonth(year, month int) (int, int) {
	if month == 12 {
		return year + 1, 1
	}
	return year, month + 1
}

var _ = sql.ErrNoRows
var _ = model.Resource{}
