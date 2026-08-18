// Package subjects ports apps/server/src/subjects: bgmx calendar sync,
// subject storage, and active-subject matching for resources.
package subjects

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/db"
	"github.com/Mogvl/dmbt-web/server/internal/search"
)

// BGMBase is the AnimeGarden Bangumi mirror used by bgmx.
const BGMBase = "https://bgm.animes.garden"

// Subject mirrors the API subject row plus keywords.
type Subject struct {
	ID         int64
	Name       string
	Keywords   []string
	ActivedAt  time.Time
	IsArchived bool
}

// CalendarSubject mirrors the bgmx CalendarSubject.
type CalendarSubject struct {
	ID        int64             `json:"id"`
	Title     string            `json:"title"`
	Alias     map[string][]string `json:"alias"`
	Poster    string            `json:"poster"`
	OnairDate *string           `json:"onair_date"`
	Search    struct {
		Include []string `json:"include"`
	} `json:"search"`
	Bangumi *struct {
		Date     string `json:"date"`
		Platform string `json:"platform"`
		Images   struct {
			Large string `json:"large"`
		} `json:"images"`
		Summary   string   `json:"summary"`
		MetaTags  []string `json:"meta_tags"`
		Tags      []string `json:"tags"`
	} `json:"bangumi"`
}

type calendarResp struct {
	OK   bool `json:"ok"`
	Data struct {
		Seasons   []string          `json:"seasons"`
		UpdatedAt string            `json:"updated_at"`
		Calendar  [][]CalendarSubject `json:"calendar"`
		Web       []CalendarSubject `json:"web"`
	} `json:"data"`
}

// Module manages subjects.
type Module struct {
	DB *sql.DB
	// ActiveSubjects is the in-memory active subject list.
	ActiveSubjects []Subject
	// AllSubjects is the full subject list (for sitemaps).
	AllSubjects []Subject
}

// Load reloads subjects from the DB into memory.
func (m *Module) Load() error {
	rows, err := m.DB.Query("SELECT bangumi_id, name, keywords, actived_at, is_archived FROM subjects")
	if err != nil {
		return err
	}
	defer rows.Close()
	var all []Subject
	for rows.Next() {
		var s Subject
		var keywords, activedAt string
		var archived int
		if err := rows.Scan(&s.ID, &s.Name, &keywords, &activedAt, &archived); err != nil {
			return err
		}
		json.Unmarshal([]byte(keywords), &s.Keywords)
		s.ActivedAt, _ = db.ParseTime(activedAt)
		s.IsArchived = archived != 0
		all = append(all, s)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	m.AllSubjects = all
	m.ActiveSubjects = nil
	for _, s := range all {
		if !s.IsArchived {
			m.ActiveSubjects = append(m.ActiveSubjects, s)
		}
	}
	return nil
}

// FetchCalendar ports bgmx fetchCalendar: GET /calendar.
func FetchCalendar(timeout time.Duration) ([]CalendarSubject, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(BGMBase + "/calendar")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload calendarResp
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if !payload.OK {
		return nil, fmt.Errorf("bgm calendar not ok")
	}
	var onair []CalendarSubject
	for _, day := range payload.Data.Calendar {
		onair = append(onair, day...)
	}
	onair = append(onair, payload.Data.Web...)
	return onair, nil
}

// TransformSubject ports transformSubjects for one calendar subject.
func TransformSubject(bgm CalendarSubject) (*Subject, error) {
	onairDate := ""
	if bgm.OnairDate != nil {
		onairDate = *bgm.OnairDate
	} else if bgm.Bangumi != nil {
		onairDate = bgm.Bangumi.Date
	}
	if onairDate == "" {
		return nil, fmt.Errorf("missing onair date for subject %d", bgm.ID)
	}
	activedAt, err := toShanghaiDate(onairDate)
	if err != nil {
		return nil, err
	}

	title := bgm.Title
	if aliasZH := bgm.Alias["zh"]; len(aliasZH) > 0 && aliasZH[0] != "" {
		title = aliasZH[0]
	}

	// keywords = dedupe(normalizeTitle([title, bgm.title, ...alias values, ...search.include]))
	var rawKeywords []string
	rawKeywords = append(rawKeywords, title)
	rawKeywords = append(rawKeywords, bgm.Title)
	for k, values := range bgm.Alias {
		_ = k
		rawKeywords = append(rawKeywords, values...)
	}
	rawKeywords = append(rawKeywords, bgm.Search.Include...)
	seen := map[string]bool{}
	var keywords []string
	for _, k := range rawKeywords {
		n := search.NormalizeTitle(k)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		keywords = append(keywords, n)
	}

	return &Subject{
		ID:        bgm.ID,
		Name:      title,
		Keywords:  keywords,
		ActivedAt: activedAt,
	}, nil
}

// toShanghaiDate ports toShanghai from bgmd.ts: 'YYYY-MM-DD' -> midnight
// Asia/Shanghai as a UTC instant (Date.UTC(y, m-1, d) - 8h).
func toShanghaiDate(date string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, err
	}
	utc := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return utc.Add(-8 * time.Hour), nil
}

// UpdateCalendar ports SubjectsModule.updateCalendar: fetch, archive missing,
// upsert changed, index resources, reload.
func (m *Module) UpdateCalendar(indexSubject func(subject Subject, overwrite bool) ([]int64, error)) error {
	onair, err := FetchCalendar(30 * time.Second)
	if err != nil {
		return err
	}

	insertMap := map[int64]bool{}
	for _, bgm := range onair {
		insertMap[bgm.ID] = true
	}

	// archive local active subjects not in the new calendar
	for _, s := range m.ActiveSubjects {
		if !insertMap[s.ID] {
			m.DB.Exec("UPDATE subjects SET is_archived = 1 WHERE bangumi_id = ?", s.ID)
		}
	}

	for _, bgm := range onair {
		sub, err := TransformSubject(bgm)
		if err != nil {
			continue
		}
		existing := m.getByID(sub.ID)
		shouldIndex := existing == nil ||
			!equalKeywords(existing.Keywords, sub.Keywords) ||
			!existing.ActivedAt.Equal(sub.ActivedAt)

		stmt := `INSERT INTO subjects (bangumi_id, name, keywords, actived_at, is_archived)
			VALUES (?, ?, ?, ?, 0)
			ON CONFLICT(bangumi_id) DO UPDATE SET
				name = excluded.name, keywords = excluded.keywords,
				actived_at = excluded.actived_at, is_archived = 0`
		keywords, _ := json.Marshal(sub.Keywords)
		if _, err := m.DB.Exec(stmt, sub.ID, sub.Name, string(keywords), db.FormatTime(sub.ActivedAt)); err != nil {
			continue
		}

		if shouldIndex && sub.ActivedAt.Year() >= 2000 {
			if indexSubject != nil {
				if _, err := indexSubject(*sub, false); err != nil {
					return err
				}
			}
		}
	}

	return m.Load()
}

// getByID returns the in-memory subject with the id.
func (m *Module) getByID(id int64) *Subject {
	for i := range m.AllSubjects {
		if m.AllSubjects[i].ID == id {
			return &m.AllSubjects[i]
		}
	}
	return nil
}

func equalKeywords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MatchActiveSubject ports matchActiveSubjects: first active subject whose
// keyword is contained in titleAlt (lowercased) wins.
func (m *Module) MatchActiveSubject(titleAlt string) *int64 {
	title := strings.ToLower(titleAlt)
	for _, sub := range m.ActiveSubjects {
		for _, key := range sub.Keywords {
			if strings.Contains(title, strings.ToLower(key)) {
				id := sub.ID
				return &id
			}
		}
	}
	return nil
}

// IndexSubject ports SubjectsModule.indexSubject: backfill subject_id for
// resources matching any keyword, created after activedAt-offset.
func (m *Module) IndexSubject(sub Subject, overwrite bool, offsetDays int) ([]int64, error) {
	cols := strings.Builder{}
	args := []any{}
	var conds []string
	conds = append(conds, "is_deleted = 0")
	conds = append(conds, "created_at >= ?")
	args = append(args, db.FormatTime(sub.ActivedAt.AddDate(0, 0, -offsetDays)))
	if !overwrite {
		conds = append(conds, "subject_id IS NULL")
	}
	var ors []string
	for _, key := range sub.Keywords {
		ors = append(ors, "title_alt LIKE ? ESCAPE '\\'")
		args = append(args, "%"+escapeLike(strings.ToLower(key))+"%")
	}
	conds = append(conds, "("+strings.Join(ors, " OR ")+")")
	_ = cols

	rows, err := m.DB.Query(
		"SELECT id FROM resources WHERE "+strings.Join(conds, " AND "), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		if _, err := m.DB.Exec("UPDATE resources SET subject_id = ? WHERE id = ?", sub.ID, id); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

// IndexSubjectAll indexes all active subjects (used at boot after import).
func (m *Module) IndexSubjectAll(offsetDays int) ([]int64, error) {
	var all []int64
	for _, sub := range m.ActiveSubjects {
		ids, err := m.IndexSubject(sub, false, offsetDays)
		if err != nil {
			return all, err
		}
		all = append(all, ids...)
	}
	return all, nil
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// SortByActivedAtDesc sorts subjects by activedAt desc (then id desc).
func SortByActivedAtDesc(subs []Subject) {
	sort.Slice(subs, func(i, j int) bool {
		if !subs[i].ActivedAt.Equal(subs[j].ActivedAt) {
			return subs[i].ActivedAt.After(subs[j].ActivedAt)
		}
		return subs[i].ID > subs[j].ID
	})
}