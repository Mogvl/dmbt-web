// Package collections ports apps/server/src/collections + the ohash
// collection hashing from @animegarden/client.
package collections

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/db"
)

// Filter mirrors CollectionFilter as stored (JSON passthrough style).
type Filter map[string]any

// Collection mirrors the client Collection type.
type Collection struct {
	Hash          string   `json:"hash,omitempty"`
	Name          string   `json:"name"`
	Authorization string   `json:"authorization"`
	Filters       []Filter `json:"filters"`
}

// ParseCollection ports parseCollection validation:
// name coerce string default '', authorization required string,
// filters array 1..50, each filter name default '' + searchParams string.
func ParseCollection(payload any) *Collection {
	raw, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	name := ""
	if v, ok := raw["name"]; ok {
		name = coerceString(v)
	}
	authorization, ok := raw["authorization"].(string)
	if !ok || authorization == "" {
		return nil
	}
	filtersRaw, ok := raw["filters"].([]any)
	if !ok || len(filtersRaw) < 1 || len(filtersRaw) > 50 {
		return nil
	}
	filters := make([]Filter, 0, len(filtersRaw))
	for _, f := range filtersRaw {
		fm, ok := f.(map[string]any)
		if !ok {
			return nil
		}
		sp, ok := fm["searchParams"].(string)
		if !ok {
			return nil
		}
		cf := Filter{}
		for k, v := range fm {
			cf[k] = v
		}
		if _, ok := cf["name"]; !ok {
			cf["name"] = ""
		}
		cf["searchParams"] = sp
		filters = append(filters, cf)
	}
	return &Collection{
		Name:          name,
		Authorization: authorization,
		Filters:       filters,
	}
}

func coerceString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return fmt.Sprintf("%v", t)
	case nil:
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// HashCollection ports hashCollection: sort filters by searchParams
// (localeCompare = code-unit order), strip name/searchParams/resources/
// complete, ohash-serialize, SHA-1 hex.
func HashCollection(c *Collection) string {
	sorted := append([]Filter{}, c.Filters...)
	sort.Slice(sorted, func(i, j int) bool {
		return localeCompare(sp(sorted[i]), sp(sorted[j]))
	})
	var cleaned []map[string]any
	for _, f := range sorted {
		m := map[string]any{}
		for k, v := range f {
			switch k {
			case "name", "searchParams", "resources", "complete":
				continue
			}
			// z.coerce.date() turns before/after into Date objects; the
			// ohash serializer then emits Date(ISO). Convert here.
			if k == "before" || k == "after" {
				if s, ok := v.(string); ok {
					if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
						v = t
					}
				} else if n, ok := v.(float64); ok {
					v = time.UnixMilli(int64(n)).UTC()
				}
			}
			m[k] = v
		}
		cleaned = append(cleaned, m)
	}
	body := OhashSerialize(cleaned)
	sum := sha1.Sum([]byte(body))
	return hex.EncodeToString(sum[:])
}

func sp(f Filter) string {
	s, _ := f["searchParams"].(string)
	return s
}

// localeCompare mimics String.prototype.localeCompare with default locale:
// falls back to code-unit comparison.
func localeCompare(a, b string) bool {
	if a == b {
		return false
	}
	return a < b
}

// Store persists collections in SQLite.
type Store struct {
	DB *sql.DB
}

// Generate ports generateCollection: insert on conflict do nothing, return
// id/hash/createdAt.
func (s *Store) Generate(c *Collection) (id int64, hash string, createdAt string, err error) {
	hsh := HashCollection(c)
	filters, _ := json.Marshal(c.Filters)
	now := db.Now()
	resp, err := s.DB.Exec(`INSERT INTO collections (hash, name, user, filters, fetched_at)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT(hash) DO NOTHING`,
		hsh, c.Name, c.Authorization, string(filters), now)
	if err != nil {
		return 0, "", "", err
	}
	if n, _ := resp.RowsAffected(); n > 0 {
		id, _ = resp.LastInsertId()
		return id, hsh, now, nil
	}
	// exists: re-select
	var createdAt2 string
	if err := s.DB.QueryRow("SELECT id, hash, fetched_at FROM collections WHERE hash = ?", hsh).Scan(&id, &hash, &createdAt2); err != nil {
		return 0, "", "", err
	}
	return id, hsh, createdAt2, nil
}

// GetRow returns a stored collection row by hash.
func (s *Store) GetRow(hash string) (*Row, error) {
	var r Row
	var filters string
	var fetchedAt string
	err := s.DB.QueryRow("SELECT id, hash, name, user, filters, fetched_at FROM collections WHERE hash = ?", hash).
		Scan(&r.ID, &r.Hash, &r.Name, &r.User, &filters, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.FetchedAt, _ = db.ParseTime(fetchedAt)
	if err := json.Unmarshal([]byte(filters), &r.FiltersRaw); err != nil {
		return nil, err
	}
	return &r, nil
}

// Row is a stored collection row.
type Row struct {
	ID         int64
	Hash       string
	Name       string
	User       string
	FiltersRaw []Filter
	FetchedAt  time.Time
}

var _ = strings.TrimSpace
