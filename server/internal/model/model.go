// Package model defines the JSON wire types, matching @animegarden/client
// Resource / Subject / Collection shapes exactly.
package model

import (
	"encoding/json"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/filter"
)

// Resource mirrors the API Resource<T> type. Optional fields are omitted
// from JSON when empty, matching the original serialization.
type Resource struct {
	ID         int64      `json:"id"`
	Provider   string     `json:"provider"`
	ProviderID string     `json:"providerId"`
	Title      string     `json:"title"`
	Href       string     `json:"href"`
	Type       string     `json:"type"`
	Magnet     string     `json:"magnet"`
	Tracker    *string    `json:"tracker,omitempty"`
	Size       int64      `json:"size"`
	Fansub     *UserBrief `json:"fansub,omitempty"`
	Publisher  UserBrief  `json:"publisher"`
	SubjectID  *int64     `json:"subjectId,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	FetchedAt  time.Time  `json:"fetchedAt"`
	Metadata   *any       `json:"metadata,omitempty"`
}

// UserBrief mirrors the publisher/fansub object.
type UserBrief struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

// ResourceDetail mirrors ResourceDetail from @animegarden/client.
type ResourceDetail struct {
	Description  string   `json:"description"`
	Files        []File   `json:"files"`
	Magnets      []Magnet `json:"magnets"`
	HasMoreFiles bool     `json:"hasMoreFiles"`
}

type File struct {
	Name string `json:"name"`
	Size string `json:"size"`
}

type Magnet struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Subject mirrors the subjects API shape.
type Subject struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Keywords   []string  `json:"keywords"`
	ActivedAt  time.Time `json:"activedAt"`
	IsArchived bool      `json:"isArchived"`
}

// Provider mirrors the providers entry.
type Provider struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	RefreshedAt time.Time `json:"refreshedAt"`
	IsActive    bool      `json:"isActive"`
}

// MarshalJSON renders times in the fixed PostgreSQL-style format.
func (p Provider) MarshalJSON() ([]byte, error) {
	return []byte(`{"id":` + jsonString(p.ID) + `,"name":` + jsonString(p.Name) +
		`,"refreshedAt":` + jsonString(p.RefreshedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")) +
		`,"isActive":` + boolStr(p.IsActive) + `}`), nil
}

// CollectionFilter is one filter entry of a collection.
type CollectionFilter struct {
	// ResolvedFilterOptions fields (only present when set)
	Preset     string    `json:"preset,omitempty"`
	Provider   string    `json:"provider,omitempty"`
	Duplicate  *bool     `json:"duplicate,omitempty"`
	Types      []string  `json:"types,omitempty"`
	Fansubs    []string  `json:"fansubs,omitempty"`
	Publishers []string  `json:"publishers,omitempty"`
	Subjects   []int64   `json:"subjects,omitempty"`
	Search     []string  `json:"search,omitempty"`
	Include    []string  `json:"include,omitempty"`
	Keywords   []string  `json:"keywords,omitempty"`
	Exclude    []string  `json:"exclude,omitempty"`
	Before     time.Time `json:"before,omitempty"`
	After      time.Time `json:"after,omitempty"`

	// Collection-specific
	Name         string     `json:"name"`
	SearchParams string     `json:"searchParams,omitempty"`
	Resources    []Resource `json:"resources,omitempty"`
	Complete     bool       `json:"complete,omitempty"`
}

// CollectionResult is the server response for a stored collection.
type CollectionResult struct {
	OK        bool               `json:"ok"`
	Hash      string             `json:"hash"`
	Name      string             `json:"name"`
	CreatedAt string             `json:"createdAt"`
	Filters   []CollectionFilter `json:"filters"`
	Timestamp time.Time          `json:"timestamp"`
}

// CollectionResourcesResult includes resolved resources for each filter.
type CollectionResourcesResult struct {
	OK        bool                   `json:"ok"`
	Hash      string                 `json:"hash"`
	Name      string                 `json:"name"`
	Filters   []CollectionFilter     `json:"filters"`
	CreatedAt string                 `json:"createdAt"`
	Results   []CollectionResultItem `json:"results"`
	Timestamp time.Time              `json:"timestamp"`
}

type CollectionResultItem struct {
	Resources []Resource            `json:"resources"`
	Complete  bool                  `json:"complete"`
	Filter    *filter.FilterOptions `json:"filter"`
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
