// Package query implements the resources QueryManager: filter → SQL,
// mirroring apps/server/src/resources/query.ts of the original.
package query

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/db"
	"github.com/Mogvl/dmbt-web/server/internal/filter"
	"github.com/Mogvl/dmbt-web/server/internal/model"
	"github.com/Mogvl/dmbt-web/server/internal/search"
)

// Store resolves user/team names to ids and holds DB access.
type Store struct {
	DB *sql.DB

	// NameToUserID / NameToTeamID are refreshed by the users/teams modules.
	NameToUserID map[string]int64
	NameToTeamID map[string]int64
	UserIDToName map[int64]string
	TeamIDToName map[int64]string
}

// FindResult mirrors the find() response payload.
type FindResult struct {
	Resources  []model.Resource
	Pagination PaginationResult
	Filter     map[string]any
}

// PaginationResult mirrors the API pagination object.
type PaginationResult struct {
	Page     int  `json:"page"`
	PageSize int  `json:"pageSize"`
	Complete bool `json:"complete"`
}

// Banned names for the bangumi preset, mirroring filter.ts.
var BANGUMI_BANNED_FANSUBS = []string{"Kirara Fantasia", "沸班亚马制作组", "GMTeam"}
var BANGUMI_BANNED_PUBLISHERS = []string{"Resona", "百度云盘", "Lanborey"}

// NormalizeDatabaseFilterOptions ports normalizeDatabaseFilterOptions:
// names -> ids, trims, removePunctuations, normalizeTitle + lowercase.
type DBFilter struct {
	Preset     string
	Provider   string
	Duplicate  *bool
	Publishers []int64 // resolved ids
	Fansubs    []int64 // resolved ids
	Types      []string
	Before     *time.Time
	After      *time.Time
	Subjects   []int64
	Search     []string // normalized
	Include    []string // normalized
	Keywords   []string // normalized
	Exclude    []string // normalized
}

func (q *Query) normalizeDBFilter(f filter.FilterOptions) DBFilter {
	out := DBFilter{
		Preset:    f.Preset,
		Provider:  f.Provider,
		Duplicate: f.Duplicate,
		Types:     f.Types,
		Before:    f.Before,
		After:     f.After,
		Subjects:  f.Subjects,
	}
	for _, name := range f.Publishers {
		if id, ok := q.store.NameToUserID[name]; ok {
			out.Publishers = append(out.Publishers, id)
		}
	}
	for _, name := range f.Fansubs {
		if id, ok := q.store.NameToTeamID[name]; ok {
			out.Fansubs = append(out.Fansubs, id)
		}
	}
	normalize := func(terms []string, punctuation bool) []string {
		var out []string
		for _, t := range terms {
			t = strings.TrimSpace(t)
			if punctuation {
				t = search.RemovePunctuations(t)
				t = strings.TrimSpace(t)
			}
			if t == "" {
				continue
			}
			out = append(out, strings.ToLower(search.NormalizeTitle(t)))
		}
		return out
	}
	if len(f.Search) > 0 {
		out.Search = normalize(f.Search, true)
	}
	if len(f.Include) > 0 {
		out.Include = normalize(f.Include, false)
	}
	if len(f.Keywords) > 0 {
		out.Keywords = normalize(f.Keywords, false)
	}
	if len(f.Exclude) > 0 {
		out.Exclude = normalize(f.Exclude, false)
	}
	return out
}

// Query runs resource queries against SQLite.
type Query struct {
	store *Store
	tok   *search.Tokenizer
}

func New(store *Store, tok *search.Tokenizer) *Query {
	return &Query{store: store, tok: tok}
}

// Find ports QueryManager.find: executes the filter with pagination and
// returns the API payload pieces.
func (q *Query) Find(f filter.FilterOptions, page, pageSize int) (*FindResult, error) {
	dbOptions := q.normalizeDBFilter(f)

	resources, hasMore, err := q.findFromDatabase(dbOptions, (page-1)*pageSize, pageSize+1)
	if err != nil {
		return nil, err
	}
	if len(resources) > pageSize {
		resources = resources[:pageSize]
	}

	// filter echo: names for fansubs/publishers, ISO strings for dates
	echo := map[string]any{}
	if dbOptions.Preset != "" {
		echo["preset"] = dbOptions.Preset
	}
	if dbOptions.Provider != "" {
		echo["provider"] = dbOptions.Provider
	}
	if dbOptions.Duplicate != nil {
		echo["duplicate"] = *dbOptions.Duplicate
	}
	if len(dbOptions.Publishers) > 0 {
		names := make([]string, 0, len(dbOptions.Publishers))
		for _, id := range dbOptions.Publishers {
			names = append(names, q.store.UserIDToName[id])
		}
		echo["publishers"] = names
	}
	if len(dbOptions.Fansubs) > 0 {
		names := make([]string, 0, len(dbOptions.Fansubs))
		for _, id := range dbOptions.Fansubs {
			names = append(names, q.store.TeamIDToName[id])
		}
		echo["fansubs"] = names
	}
	if len(dbOptions.Types) > 0 {
		echo["types"] = dbOptions.Types
	}
	if dbOptions.Before != nil {
		echo["before"] = dbOptions.Before.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	if dbOptions.After != nil {
		echo["after"] = dbOptions.After.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	if len(dbOptions.Subjects) > 0 {
		echo["subjects"] = dbOptions.Subjects
	}
	if len(dbOptions.Search) > 0 {
		echo["search"] = dbOptions.Search
	}
	if len(dbOptions.Include) > 0 {
		echo["include"] = dbOptions.Include
	}
	if len(dbOptions.Keywords) > 0 {
		echo["keywords"] = dbOptions.Keywords
	}
	if len(dbOptions.Exclude) > 0 {
		echo["exclude"] = dbOptions.Exclude
	}

	result := &FindResult{
		Resources: resources,
		Filter:    echo,
	}
	result.Pagination.Page = page
	result.Pagination.PageSize = pageSize
	result.Pagination.Complete = !hasMore
	return result, nil
}

// findFromDatabase ports findFromDatabase: builds SQL conditions.
func (q *Query) findFromDatabase(f DBFilter, offset, limit int) ([]model.Resource, bool, error) {
	var conds []string
	var args []any

	conds = append(conds, "is_deleted = 0")

	if f.Provider != "" {
		conds = append(conds, "provider_name = ?")
		args = append(args, f.Provider)
	}

	if f.Duplicate != nil && !*f.Duplicate {
		conds = append(conds, "duplicated_id IS NULL")
	}

	if len(f.Fansubs) > 0 || len(f.Publishers) > 0 {
		var sub []string
		if len(f.Fansubs) > 0 {
			ph := placeholders(len(f.Fansubs))
			sub = append(sub, "fansub_id IN ("+ph+")")
			for _, id := range f.Fansubs {
				args = append(args, id)
			}
		}
		if len(f.Publishers) > 0 {
			ph := placeholders(len(f.Publishers))
			sub = append(sub, "publisher_id IN ("+ph+")")
			for _, id := range f.Publishers {
				args = append(args, id)
			}
		}
		conds = append(conds, "("+strings.Join(sub, " OR ")+")")
	}

	if len(f.Types) > 0 {
		ph := placeholders(len(f.Types))
		conds = append(conds, "type IN ("+ph+")")
		for _, t := range f.Types {
			args = append(args, t)
		}
	}

	if len(f.Subjects) > 0 {
		ph := placeholders(len(f.Subjects))
		conds = append(conds, "subject_id IN ("+ph+")")
		for _, s := range f.Subjects {
			args = append(args, s)
		}
	}

	if f.Before != nil {
		conds = append(conds, "created_at <= ?")
		args = append(args, db.FormatTime(*f.Before))
	}
	if f.After != nil {
		conds = append(conds, "created_at >= ?")
		args = append(args, db.FormatTime(*f.After))
	}

	if len(f.Search) > 0 {
		// jieba cut each term; all tokens must be present as whole lexemes
		var tokens []string
		for _, term := range f.Search {
			for _, tok := range q.tok.Cut(term) {
				tokens = append(tokens, strings.ToLower(tok))
			}
		}
		if len(tokens) > 0 {
			// token match: whole-token equality in the stored search text
			for _, tok := range tokens {
				conds = append(conds, `(' ' || title_search || ' ') LIKE '% ' || ? || ' %' ESCAPE '\'`)
				args = append(args, escapeLike(tok))
			}
		}
	}

	if len(f.Include) > 0 {
		var sub []string
		for _, i := range f.Include {
			sub = append(sub, "title_alt LIKE ? ESCAPE '\\'")
			args = append(args, "%"+escapeLike(i)+"%")
		}
		conds = append(conds, "("+strings.Join(sub, " OR ")+")")
	}

	if len(f.Keywords) > 0 {
		for _, k := range f.Keywords {
			conds = append(conds, "title_alt LIKE ? ESCAPE '\\'")
			args = append(args, "%"+escapeLike(k)+"%")
		}
	}

	if len(f.Exclude) > 0 {
		for _, e := range f.Exclude {
			conds = append(conds, "title_alt NOT LIKE ? ESCAPE '\\'")
			args = append(args, "%"+escapeLike(e)+"%")
		}
	}

	// preset bangumi: ban publishers and fansubs
	switch f.Preset {
	case "bangumi":
		var bannedUsers, bannedTeams []int64
		for _, name := range BANGUMI_BANNED_PUBLISHERS {
			if id, ok := q.store.NameToUserID[name]; ok {
				bannedUsers = append(bannedUsers, id)
			}
		}
		for _, name := range BANGUMI_BANNED_FANSUBS {
			if id, ok := q.store.NameToTeamID[name]; ok {
				bannedTeams = append(bannedTeams, id)
			}
		}
		if len(bannedUsers) > 0 {
			ph := placeholders(len(bannedUsers))
			conds = append(conds, "publisher_id NOT IN ("+ph+")")
			for _, id := range bannedUsers {
				args = append(args, id)
			}
		}
		if len(bannedTeams) > 0 {
			ph := placeholders(len(bannedTeams))
			conds = append(conds, "fansub_id NOT IN ("+ph+")")
			for _, id := range bannedTeams {
				args = append(args, id)
			}
		}
	}

	sql := "SELECT id, provider_name, provider_id, title, title_alt, href, type, magnet, tracker, size, created_at, fetched_at, publisher_id, fansub_id, subject_id, metadata FROM resources WHERE " +
		strings.Join(conds, " AND ") +
		" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := q.store.DB.Query(sql, args...)
	if err != nil {
		return nil, false, fmt.Errorf("query resources: %w", err)
	}
	defer rows.Close()

	var out []model.Resource
	for rows.Next() {
		var r resourceRow
		if err := rows.Scan(&r.id, &r.provider, &r.providerId, &r.title, &r.titleAlt, &r.href, &r.typ, &r.magnet, &r.tracker, &r.size, &r.createdAt, &r.fetchedAt, &r.publisherID, &r.fansubID, &r.subjectID, &r.metadata); err != nil {
			return nil, false, err
		}
		out = append(out, q.transform(r))
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	// hasMore: the caller passes limit=pageSize+1; hitting the full limit
	// means more rows exist.
	return out, len(out) == limit && limit > 0, nil
}

type resourceRow struct {
	id          int64
	provider    string
	providerId  string
	title       string
	titleAlt    string
	href        string
	typ         string
	magnet      string
	tracker     sql.NullString
	size        int64
	createdAt   string
	fetchedAt   string
	publisherID int64
	fansubID    sql.NullInt64
	subjectID   sql.NullInt64
	metadata    sql.NullString
}

// transform ports QueryManager.transform + transformResourceHref.
func (q *Query) transform(r resourceRow) model.Resource {
	res := model.Resource{
		ID:         r.id,
		Provider:   r.provider,
		ProviderID: r.providerId,
		Title:      r.title,
		Href:       transformResourceHref(r.provider, r.href),
		Type:       r.typ,
		Magnet:     r.magnet,
		Size:       r.size,
		CreatedAt:  parseTime(r.createdAt),
		FetchedAt:  parseTime(r.fetchedAt),
		Publisher: model.UserBrief{
			ID:   r.publisherID,
			Name: q.store.UserIDToName[r.publisherID],
		},
	}
	if r.tracker.Valid {
		t := r.tracker.String
		res.Tracker = &t
	}
	if r.fansubID.Valid {
		res.Fansub = &model.UserBrief{
			ID:   r.fansubID.Int64,
			Name: q.store.TeamIDToName[r.fansubID.Int64],
		}
	}
	if r.subjectID.Valid {
		id := r.subjectID.Int64
		res.SubjectID = &id
	}
	// metadata: always empty object per original (metadata: {})
	if r.metadata.Valid && r.metadata.String != "" && r.metadata.String != "{}" {
		var meta any
		if err := json.Unmarshal([]byte(r.metadata.String), &meta); err == nil {
			res.Metadata = &meta
		}
	}
	return res
}

// transformResourceHref ports transformResourceHref from @animegarden/client.
func transformResourceHref(provider, href string) string {
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

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func parseTime(s string) time.Time {
	t, err := db.ParseTime(s)
	if err != nil {
		return time.Time{}
	}
	return t
}
