// Package filter ports parseURLSearch / stringifyURLSearch from
// @animegarden/client (packages/client/src/resolver.ts).
package filter

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPageSize   = 100
	MaxRequestPageSize = 1000
)

var SupportProviders = []string{"dmhy", "moe", "mikan", "ani"}
var SupportPresets = []string{"bangumi"}

// FilterOptions mirrors ResolvedFilterOptions.
type FilterOptions struct {
	Preset     string
	Provider   string
	Duplicate  *bool
	Types      []string
	Fansubs    []string
	Publishers []string
	Subjects   []int64
	Search     []string
	Include    []string
	Keywords   []string
	Exclude    []string
	Before     *time.Time
	After      *time.Time
}

// IsSet reports whether any filter option is set.
func (f *FilterOptions) IsSet() bool {
	return f.Preset != "" ||
		f.Provider != "" ||
		f.Duplicate != nil ||
		len(f.Types) > 0 ||
		len(f.Fansubs) > 0 ||
		len(f.Publishers) > 0 ||
		len(f.Subjects) > 0 ||
		len(f.Search) > 0 ||
		len(f.Include) > 0 ||
		len(f.Keywords) > 0 ||
		len(f.Exclude) > 0 ||
		f.Before != nil ||
		f.After != nil
}

// Pagination mirrors ResolvedPaginationOptions.
type Pagination struct {
	Page     int
	PageSize int
}

// ParseResult is the output of ParseURLSearch.
type ParseResult struct {
	Pagination Pagination
	Filter     FilterOptions
}

// parseDateLike ports zod dateLike: number -> Date(n), or date parse.
func parseDateLike(v string) *time.Time {
	if v == "" {
		return nil
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil {
		t := time.UnixMilli(int64(n)).UTC()
		return &t
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return &t
	}
	return nil
}

// parseBool ports zod z.coerce.boolean() — JavaScript Boolean(value).
func parseBool(v string) *bool {
	b := v != "" && v != "0" && !strings.EqualFold(v, "false") && !strings.EqualFold(v, "null") && !strings.EqualFold(v, "undefined")
	return &b
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func uniqueInt64(in []int64) []int64 {
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, n := range in {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func isNaNish(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case int:
		return t == 0
	case float64:
		return t != t // NaN
	}
	return false
}

// jsRound ports JavaScript Math.round: round half up (toward +infinity for .5).
func jsRound(f float64) int {
	if f >= 0 {
		return int(f + 0.5)
	}
	return int(f - 0.5)
}

// parseNum ports z.coerce.number() on a raw string (like Number(value)).
func parseNum(v string) *float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return &f
}

// parsePagination ports the pagination validation logic.
func parsePagination(page, pageSize *float64) Pagination {
	p := 1
	ps := DefaultPageSize
	if page != nil && !isNaNish(*page) && *page >= 1 {
		p = jsRound(*page)
	}
	if pageSize != nil && !isNaNish(*pageSize) && *pageSize >= 1 && *pageSize <= MaxRequestPageSize {
		ps = jsRound(*pageSize)
	}
	return Pagination{Page: p, PageSize: ps}
}

// BodyOptions represents an optional JSON request body (used by POST /resources).
type BodyOptions struct {
	Provider  string
	Duplicate *bool
	Page      *float64
	PageSize  *float64
	Fansub    *string
	Fansubs   []string
	Publisher *string
	Publishers []string
	Type      *string
	Types     []string
	Before    *time.Time
	After     *time.Time
	Subject   *int64
	Subjects  []int64
	Search    []string
	Include   []string
	Keywords  []string
	Exclude   []string
	Preset    string
}

// ParseURLSearch ports parseURLSearch(params, body) from the original.
// params and body may both be nil.
func ParseURLSearch(params url.Values, body *BodyOptions) ParseResult {
	var res1 = struct {
		provider  *string
		duplicate *bool
		page      *float64
		pageSize  *float64
		fansub    []string
		publisher []string
		typ       []string
		before    *time.Time
		after     *time.Time
		subject   []int64
		search    []string
		include   []string
		keyword   []string
		exclude   []string
		preset    string
	}{}

	if params != nil {
		if v := params.Get("provider"); v != "" {
			res1.provider = &v
		}
		if v, ok := params["duplicate"]; ok && len(v) > 0 {
			res1.duplicate = parseBool(v[0])
		}
		res1.page = parseNum(params.Get("page"))
		res1.pageSize = parseNum(params.Get("pageSize"))
		res1.fansub = params["fansub"]
		res1.publisher = params["publisher"]
		res1.typ = params["type"]
		res1.before = parseDateLike(params.Get("before"))
		res1.after = parseDateLike(params.Get("after"))
		if vs, ok := params["subject"]; ok {
			for _, v := range vs {
				if n, err := strconv.ParseInt(v, 10, 64); err == nil {
					res1.subject = append(res1.subject, n)
				}
			}
		}
		res1.search = params["search"]
		res1.include = params["include"]
		res1.keyword = params["keyword"]
		res1.exclude = params["exclude"]
		if v := params.Get("preset"); v != "" {
			res1.preset = v
		}
	}

	// filter assembly, following the original resolution order
	var filter FilterOptions

	if body != nil && body.Preset != "" {
		filter.Preset = body.Preset
	} else if res1.preset != "" {
		filter.Preset = res1.preset
	}

	if body != nil && body.Provider != "" {
		if body.Duplicate != nil {
			filter.Duplicate = body.Duplicate
		} else if res1.duplicate != nil {
			filter.Duplicate = res1.duplicate
		} else {
			t := true
			filter.Duplicate = &t
		}
		filter.Provider = body.Provider
	} else if res1.provider != nil {
		filter.Provider = *res1.provider
		if body != nil && body.Duplicate != nil {
			filter.Duplicate = body.Duplicate
		} else if res1.duplicate != nil {
			filter.Duplicate = res1.duplicate
		} else {
			t := true
			filter.Duplicate = &t
		}
	}

	if body != nil && body.Fansub != nil {
		filter.Fansubs = []string{*body.Fansub}
	} else if body != nil && len(body.Fansubs) > 0 {
		filter.Fansubs = body.Fansubs
	} else if len(res1.fansub) > 0 {
		filter.Fansubs = res1.fansub
	}
	if filter.Fansubs != nil {
		filter.Fansubs = uniqueStrings(filter.Fansubs)
	}

	if body != nil && body.Publisher != nil {
		filter.Publishers = []string{*body.Publisher}
	} else if body != nil && len(body.Publishers) > 0 {
		filter.Publishers = body.Publishers
	} else if len(res1.publisher) > 0 {
		filter.Publishers = res1.publisher
	}
	if filter.Publishers != nil {
		filter.Publishers = uniqueStrings(filter.Publishers)
	}

	if body != nil && body.Type != nil {
		filter.Types = []string{*body.Type}
	} else if body != nil && len(body.Types) > 0 {
		filter.Types = body.Types
	} else if len(res1.typ) > 0 {
		filter.Types = res1.typ
	}
	if filter.Types != nil {
		filter.Types = uniqueStrings(filter.Types)
	}

	if body != nil && body.Before != nil {
		filter.Before = body.Before
	} else if res1.before != nil {
		filter.Before = res1.before
	}
	if body != nil && body.After != nil {
		filter.After = body.After
	} else if res1.after != nil {
		filter.After = res1.after
	}

	if body != nil && body.Subject != nil {
		filter.Subjects = []int64{*body.Subject}
	} else if body != nil && len(body.Subjects) > 0 {
		filter.Subjects = body.Subjects
	} else if len(res1.subject) > 0 {
		filter.Subjects = res1.subject
	}
	if filter.Subjects != nil {
		filter.Subjects = uniqueInt64(filter.Subjects)
	}

	if body != nil && len(body.Search) > 0 {
		filter.Search = body.Search
	} else if len(res1.search) > 0 {
		filter.Search = res1.search
	}
	if filter.Search != nil {
		filter.Search = uniqueStrings(filter.Search)
	}

	if body != nil && len(body.Include) > 0 {
		filter.Include = body.Include
	} else if len(res1.include) > 0 {
		filter.Include = res1.include
	}
	if filter.Include != nil {
		filter.Include = uniqueStrings(filter.Include)
	}

	if body != nil && len(body.Keywords) > 0 {
		filter.Keywords = body.Keywords
	} else if len(res1.keyword) > 0 {
		filter.Keywords = res1.keyword
	}
	if filter.Keywords != nil {
		filter.Keywords = uniqueStrings(filter.Keywords)
	}

	if body != nil && len(body.Exclude) > 0 {
		filter.Exclude = body.Exclude
	} else if len(res1.exclude) > 0 {
		filter.Exclude = res1.exclude
	}
	if filter.Exclude != nil {
		filter.Exclude = uniqueStrings(filter.Exclude)
	}

	// search mode discards include
	if len(filter.Search) > 0 {
		filter.Include = nil
	}

	// pagination
	var page, pageSize *float64
	if body != nil {
		page = body.Page
		pageSize = body.PageSize
	}
	if res1.page != nil {
		page = res1.page
	}
	if res1.pageSize != nil {
		pageSize = res1.pageSize
	}
	pagination := parsePagination(page, pageSize)

	return ParseResult{Pagination: pagination, Filter: filter}
}

// StringifyURLSearch ports stringifyURLSearch: page/pageSize/preset/
// duplicate/after/before are single params; arrays are repeated params;
// params are sorted by key.
func StringifyURLSearch(f FilterOptions, page, pageSize int, duplicate bool) url.Values {
	params := url.Values{}
	if f.Preset != "" {
		params.Set("preset", f.Preset)
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.Itoa(pageSize))
	}
	if duplicate {
		params.Set("duplicate", "true")
	}
	if f.After != nil {
		params.Set("after", strconv.FormatInt(f.After.UnixMilli(), 10))
	}
	if f.Before != nil {
		params.Set("before", strconv.FormatInt(f.Before.UnixMilli(), 10))
	}
	if f.Provider != "" {
		params.Set("provider", f.Provider)
	}
	if len(f.Search) > 0 {
		for _, w := range uniqueStrings(f.Search) {
			params.Add("search", w)
		}
		for _, w := range uniqueStrings(f.Keywords) {
			params.Add("keyword", w)
		}
		for _, w := range uniqueStrings(f.Exclude) {
			params.Add("exclude", w)
		}
	} else if len(f.Include) > 0 {
		for _, w := range uniqueStrings(f.Include) {
			params.Add("include", w)
		}
		for _, w := range uniqueStrings(f.Keywords) {
			params.Add("keyword", w)
		}
		for _, w := range uniqueStrings(f.Exclude) {
			params.Add("exclude", w)
		}
	} else {
		for _, w := range uniqueStrings(f.Keywords) {
			params.Add("keyword", w)
		}
		for _, w := range uniqueStrings(f.Exclude) {
			params.Add("exclude", w)
		}
	}
	if len(f.Subjects) > 0 {
		for _, s := range uniqueInt64(f.Subjects) {
			params.Add("subject", strconv.FormatInt(s, 10))
		}
	}
	for _, t := range uniqueStrings(f.Types) {
		params.Add("type", t)
	}
	for _, f2 := range uniqueStrings(f.Fansubs) {
		params.Add("fansub", f2)
	}
	for _, p := range uniqueStrings(f.Publishers) {
		params.Add("publisher", p)
	}
	return params
}
