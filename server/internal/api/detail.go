package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/collections"
	"github.com/Mogvl/dmbt-web/server/internal/filter"
	"github.com/Mogvl/dmbt-web/server/internal/model"
	"github.com/Mogvl/dmbt-web/server/internal/providers"
	"github.com/Mogvl/dmbt-web/server/internal/resources"
	"github.com/Mogvl/dmbt-web/server/internal/scraper"
)

const detailExpire = 7 * 24 * 60 * 60 // seconds (7 days)

// detailCache mirrors the findProviderDetail memo (1h TTL, in-memory).
type detailCacheEntry struct {
	expires time.Time
	body    map[string]any
}

var detailCache = map[string]detailCacheEntry{}

func (s *Server) getCachedDetail(key string) (map[string]any, bool) {
	e, ok := detailCache[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expires) {
		delete(detailCache, key)
		return nil, false
	}
	return e.body, true
}

func (s *Server) setCachedDetail(key string, body map[string]any) {
	detailCache[key] = detailCacheEntry{expires: time.Now().Add(time.Hour), body: body}
	if len(detailCache) > 10000 {
		for k := range detailCache {
			delete(detailCache, k)
			if len(detailCache) <= 9000 {
				break
			}
		}
	}
}

// handleProviderResource ports /resource/:provider/:id and /detail/:provider/:id.
func (s *Server) handleProviderResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(405)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) != 3 {
		s.errJSON(w, 400, "Unknown detail id")
		return
	}
	providerName := parts[1]
	detailID := parts[2]

	provider, ok := s.sys.Providers.Get(providerName)
	if !ok {
		s.errJSON(w, 200, fmt.Sprintf("Unknown detail id: %s %s", providerName, detailID))
		return
	}

	key := providerName + ":" + detailID
	if cached, ok := s.getCachedDetail(key); ok {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		s.json(w, 200, cached)
		return
	}

	detailURL, err := provider.GetDetailURL(s.sys, detailID)
	if err != nil || detailURL == nil {
		s.errJSON(w, 200, fmt.Sprintf("Unknown detail id: %s %s", providerName, detailID))
		return
	}

	respBody := s.fetchProviderDetail(providerName, provider, detailURL)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	s.setCachedDetail(key, respBody)
	s.json(w, 200, respBody)
}

// fetchProviderDetail ports findProviderDetail + DetailsManager.getByProviderId.
func (s *Server) fetchProviderDetail(providerName string, provider *providers.Provider, detailURL *providers.DetailURL) map[string]any {
	row, err := s.sys.Store.GetByProviderID(providerName, detailURL.ProviderID)
	if err != nil {
		return map[string]any{"status": "ERROR", "message": fmt.Sprintf("Unknown detail id: %s %s", providerName, detailURL.ProviderID)}
	}
	if row == nil {
		return map[string]any{
			"status":       "OK",
			"resource":     nil,
			"detail":       nil,
			"isDeleted":    false,
			"duplicatedId": nil,
		}
	}

	resourceBody := s.resourceToJSON(row)

	// detail freshness
	var detailBody *model.ResourceDetail
	detailRow, _ := s.sys.Store.GetDetail(row.ID)
	stale := false
	if detailRow == nil {
		stale = true
	} else {
		stale = time.Now().UTC().Sub(detailRow.FetchedAt) >= detailExpire*time.Second
	}

	if stale {
		fetched, err := provider.FetchDetail(detailURL.Href, 5)
		if err == nil && fetched != nil {
			d := model.ResourceDetail{
				Description:  fetched.Description,
				HasMoreFiles: fetched.HasMoreFiles,
			}
			for _, f := range fetched.Files {
				d.Files = append(d.Files, model.File{Name: f.Name, Size: f.Size})
			}
			for _, m := range fetched.Magnets {
				d.Magnets = append(d.Magnets, model.Magnet{Name: m.Name, URL: m.URL})
			}
			detailBody = &d
			s.sys.Store.InsertDetail(row.ID, d)
			s.fixResourceWithDetail(row, fetched)
		}
	} else {
		var d model.ResourceDetail
		json.Unmarshal([]byte(detailRow.Magnets), &d.Magnets)
		json.Unmarshal([]byte(detailRow.Files), &d.Files)
		d.Description = detailRow.Description
		d.HasMoreFiles = detailRow.HasMoreFiles
		detailBody = &d
	}

	return map[string]any{
		"status":       "OK",
		"resource":     resourceBody,
		"detail":       detailBody,
		"isDeleted":    row.IsDeleted,
		"duplicatedId": nullInt64(row.DuplicatedID),
	}
}

// fixResourceWithDetail ports fixResourceWithDetail: backfill missing
// publisher/fansub avatars from the scraped detail.
func (s *Server) fixResourceWithDetail(row *resources.DBResourceRow, detail *scraper.ScrapedResourceDetail) {
	if detail == nil {
		return
	}
	if detail.Publisher != nil && detail.Publisher.Avatar != "" {
		var avatar string
		if err := s.sys.DB.QueryRow("SELECT avatar FROM users WHERE id = ?", row.PublisherID).Scan(&avatar); err == nil && avatar == "" {
			s.sys.DB.Exec("UPDATE users SET avatar = ? WHERE id = ?", detail.Publisher.Avatar, row.PublisherID)
		}
	}
	if row.FansubID.Valid && detail.Fansub != nil && detail.Fansub.Avatar != "" {
		var avatar string
		if err := s.sys.DB.QueryRow("SELECT avatar FROM teams WHERE id = ?", row.FansubID.Int64).Scan(&avatar); err == nil && avatar == "" {
			s.sys.DB.Exec("UPDATE teams SET avatar = ? WHERE id = ?", detail.Fansub.Avatar, row.FansubID.Int64)
		}
	}
}

func transformHref(provider, href string) string {
	switch provider {
	case "dmhy":
		return "https://share.dmhy.org/topics/view/" + href
	case "mikan":
		return "https://mikanani.me/Home/Episode/" + href
	case "moe":
		return "https://bangumi.moe/torrent/" + href
	case "ani":
		return href
	}
	return ""
}

// resourceToJSON renders a DB resource row with the API shape.
func (s *Server) resourceToJSON(row *resources.DBResourceRow) map[string]any {
	out := map[string]any{
		"id":         row.ID,
		"provider":   row.Provider,
		"providerId": row.ProviderID,
		"title":      row.Title,
		"href":       transformHref(row.Provider, row.Href),
		"type":       row.Type,
		"magnet":     row.Magnet,
		"size":       row.Size,
		"createdAt":  row.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		"fetchedAt":  row.FetchedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	}
	if row.Tracker != "" {
		out["tracker"] = row.Tracker
	}
	publisher := map[string]any{"id": row.PublisherID}
	if name, ok := s.sys.Store.UserIDToName[row.PublisherID]; ok {
		publisher["name"] = name
	}
	if avatar, ok := s.userAvatar(row.PublisherID); ok && avatar != "" {
		publisher["avatar"] = avatar
	}
	out["publisher"] = publisher
	if row.FansubID.Valid {
		fansub := map[string]any{"id": row.FansubID.Int64}
		if name, ok := s.sys.Store.TeamIDToName[row.FansubID.Int64]; ok {
			fansub["name"] = name
		}
		if avatar, ok := s.teamAvatar(row.FansubID.Int64); ok && avatar != "" {
			fansub["avatar"] = avatar
		}
		out["fansub"] = fansub
	}
	if row.SubjectID.Valid {
		out["subjectId"] = row.SubjectID.Int64
	}
	if row.Metadata != "" && row.Metadata != "{}" {
		var meta any
		if json.Unmarshal([]byte(row.Metadata), &meta) == nil {
			out["metadata"] = meta
		}
	}
	return out
}

func (s *Server) userAvatar(id int64) (string, bool) {
	var avatar string
	if err := s.sys.DB.QueryRow("SELECT avatar FROM users WHERE id = ?", id).Scan(&avatar); err != nil {
		return "", false
	}
	return avatar, true
}

func (s *Server) teamAvatar(id int64) (string, bool) {
	var avatar string
	if err := s.sys.DB.QueryRow("SELECT avatar FROM teams WHERE id = ?", id).Scan(&avatar); err != nil {
		return "", false
	}
	return avatar, true
}

var hexRe = regexp.MustCompile(`^[0-9A-F]{40}$`)
var b32CoreRe = regexp.MustCompile(`^[A-Z2-7]{32}$`)

func nullInt64(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

// handleInfoHash ports GET /detail/infohash/:hash.
func (s *Server) handleInfoHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(405)
		return
	}
	raw := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/detail/infohash/"), "/")
	infoHash := strings.TrimSpace(raw)
	upper := strings.ToUpper(infoHash)
	isHex := hexRe.MatchString(upper)
	isBase32 := b32CoreRe.MatchString(upper)

	notFound := func(message string, noStore bool) {
		if noStore {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		s.json(w, 200, map[string]any{
			"status":       "ERROR",
			"message":      message,
			"resource":     nil,
			"detail":       nil,
			"isDeleted":    false,
			"duplicatedId": nil,
		})
	}

	if infoHash == "" || (!isHex && !isBase32) {
		notFound(fmt.Sprintf("Invalid info hash: %s", raw), true)
		return
	}

	row, err := s.sys.Store.GetByInfoHash(infoHash)
	if err != nil || row == nil {
		notFound(fmt.Sprintf("Unknown detail info hash: %s", infoHash), false)
		return
	}

	var respBody map[string]any
	if provider, ok := s.sys.Providers.Get(row.Provider); ok {
		respBody = s.fetchProviderDetail(row.Provider, provider, &providers.DetailURL{ProviderID: row.ProviderID, Href: row.Href})
	} else {
		respBody = map[string]any{
			"status":       "OK",
			"resource":     s.resourceToJSON(row),
			"detail":       nil,
			"isDeleted":    row.IsDeleted,
			"duplicatedId": nullInt64(row.DuplicatedID),
		}
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	s.json(w, 200, respBody)
}

// --- collections routes ---

// handleCreateCollection ports POST|PUT /collection.
func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		w.WriteHeader(405)
		return
	}
	var payload any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.errJSON(w, 400, "Incorrect collection format")
		return
	}
	collection := collections.ParseCollection(payload)
	if collection == nil {
		s.errJSON(w, 400, "Incorrect collection format")
		return
	}
	id, hash, createdAt, err := s.sys.Collections.Generate(collection)
	if err != nil {
		s.errJSON(w, 400, "Failed generating collection")
		return
	}
	s.json(w, 200, map[string]any{
		"status":    "OK",
		"id":        id,
		"hash":      hash,
		"createdAt": createdAt,
	})
}

// handleGetCollection ports GET /collection/:hash.
func (s *Server) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimPrefix(r.URL.Path, "/collection/")
	hash = strings.TrimSuffix(hash, "/")
	if r.Method != http.MethodGet || hash == "" {
		s.errJSON(w, 400, "Missing collection hash")
		return
	}
	row, err := s.sys.Collections.GetRow(hash)
	if err != nil || row == nil {
		s.errJSON(w, 400, "Failed querying collection result")
		return
	}

	var results []map[string]any
	for _, f := range row.FiltersRaw {
		options := FilterFromStoredMap(f)
		find, err := s.sys.Query.Find(options, 1, 1000)
		if err != nil {
			results = append(results, map[string]any{"resources": []any{}, "complete": true, "filter": nil})
			continue
		}
		resources := make([]model.Resource, 0, len(find.Resources))
		resources = append(resources, find.Resources...)
		var filterJSON any
		if len(find.Filter) > 0 {
			filterJSON = find.Filter
		}
		results = append(results, map[string]any{
			"resources": resources,
			"complete":  find.Pagination.Complete,
			"filter":    filterJSON,
		})
	}

	s.json(w, 200, map[string]any{
		"status":    "OK",
		"hash":      row.Hash,
		"name":      row.Name,
		"createdAt": row.FetchedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		"updatedAt": time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		"filters":   row.FiltersRaw,
		"results":   results,
	})
}

// FilterFromStoredMap converts a stored filter map to FilterOptions,
// mirroring collections/index.ts: empty arrays dropped, dates parsed.
func FilterFromStoredMap(f collections.Filter) filter.FilterOptions {
	options := filter.FilterOptions{}
	dropEmpty := func(key string) []string {
		v, ok := f[key].([]any)
		if !ok {
			return nil
		}
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	if v, ok := f["preset"].(string); ok {
		options.Preset = v
	}
	if v, ok := f["provider"].(string); ok {
		options.Provider = v
	}
	if v, ok := f["duplicate"].(bool); ok {
		options.Duplicate = &v
	}
	options.Types = dropEmpty("types")
	options.Fansubs = dropEmpty("fansubs")
	options.Publishers = dropEmpty("publishers")
	options.Search = dropEmpty("search")
	options.Include = dropEmpty("include")
	options.Keywords = dropEmpty("keywords")
	options.Exclude = dropEmpty("exclude")
	if v, ok := f["subjects"].([]any); ok {
		for _, item := range v {
			if n, ok := item.(float64); ok {
				options.Subjects = append(options.Subjects, int64(n))
			} else if s, ok := item.(string); ok {
				if n, err := strconv.ParseInt(s, 10, 64); err == nil {
					options.Subjects = append(options.Subjects, n)
				}
			}
		}
	}
	for _, key := range []string{"before", "after"} {
		if v, ok := f[key]; ok {
			if t := coerceDateValue(v); t != nil {
				switch key {
				case "before":
					options.Before = t
				case "after":
					options.After = t
				}
			}
		}
	}
	return options
}

func coerceDateValue(v any) *time.Time {
	switch t := v.(type) {
	case string:
		if n, err := strconv.ParseFloat(t, 64); err == nil {
			tt := time.UnixMilli(int64(n)).UTC()
			return &tt
		}
		if tt, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return &tt
		}
	case float64:
		tt := time.UnixMilli(int64(t)).UTC()
		return &tt
	}
	return nil
}