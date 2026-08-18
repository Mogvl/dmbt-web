package api

import (
	"strconv"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/filter"
)

// parseBodyOptions converts a raw JSON body into BodyOptions, mirroring the
// zod coercions of the original BodySchema.
func parseBodyOptions(raw map[string]any) *filter.BodyOptions {
	opt := &filter.BodyOptions{}
	if v, ok := raw["provider"].(string); ok {
		opt.Provider = v
	}
	if v, ok := raw["duplicate"]; ok {
		b := coerceBool(v)
		opt.Duplicate = &b
	}
	if v, ok := raw["page"]; ok {
		f := coerceFloat(v)
		opt.Page = &f
	}
	if v, ok := raw["pageSize"]; ok {
		f := coerceFloat(v)
		opt.PageSize = &f
	}
	if v, ok := raw["fansub"].(string); ok {
		opt.Fansub = &v
	}
	if v, ok := raw["fansubs"].([]any); ok {
		opt.Fansubs = coerceStrings(v)
	}
	if v, ok := raw["publisher"].(string); ok {
		opt.Publisher = &v
	}
	if v, ok := raw["publishers"].([]any); ok {
		opt.Publishers = coerceStrings(v)
	}
	if v, ok := raw["type"].(string); ok {
		opt.Type = &v
	}
	if v, ok := raw["types"].([]any); ok {
		opt.Types = coerceStrings(v)
	}
	if v, ok := raw["before"]; ok {
		if t := coerceDate(v); t != nil {
			opt.Before = t
		}
	}
	if v, ok := raw["after"]; ok {
		if t := coerceDate(v); t != nil {
			opt.After = t
		}
	}
	if v, ok := raw["subject"]; ok {
		if f := coerceFloat(v); !isNaN(f) {
			n := int64(f)
			opt.Subject = &n
		}
	}
	if v, ok := raw["subjects"].([]any); ok {
		for _, item := range v {
			if f := coerceFloat(item); !isNaN(f) {
				opt.Subjects = append(opt.Subjects, int64(f))
			}
		}
	}
	if v, ok := raw["search"]; ok {
		opt.Search = coerceStringArray(v)
	}
	if v, ok := raw["include"]; ok {
		opt.Include = coerceStringArray(v)
	}
	if v, ok := raw["keywords"]; ok {
		opt.Keywords = coerceStringArray(v)
	}
	if v, ok := raw["exclude"]; ok {
		opt.Exclude = coerceStringArray(v)
	}
	if v, ok := raw["preset"].(string); ok {
		opt.Preset = v
	}
	return opt
}

// coerceBool ports z.coerce.boolean(): JS Boolean() semantics — any
// non-empty string coerces to true ("false", "0" included).
func coerceBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case nil:
		return false
	}
	return true
}

// coerceFloat ports Number(value).
func coerceFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return 0
		}
		return f
	case bool:
		if t {
			return 1
		}
		return 0
	case nil:
		return 0
	}
	return 0
}

func isNaN(f float64) bool { return f != f }

// coerceDate ports z.coerce.date().
func coerceDate(v any) *time.Time {
	switch t := v.(type) {
	case float64:
		tt := time.UnixMilli(int64(t)).UTC()
		return &tt
	case string:
		if n, err := strconv.ParseFloat(t, 64); err == nil {
			tt := time.UnixMilli(int64(n)).UTC()
			return &tt
		}
		if tt, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return &tt
		}
	}
	return nil
}

func coerceStrings(v []any) []string {
	out := make([]string, 0, len(v))
	for _, item := range v {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// coerceStringArray ports stringArray: string -> [string], array -> array.
func coerceStringArray(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		return coerceStrings(t)
	}
	return nil
}
