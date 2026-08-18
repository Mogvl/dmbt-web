// Package api implements the HTTP API server, mirroring the original Hono
// routes and middleware behavior.
package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/collections"
	"github.com/Mogvl/dmbt-web/server/internal/config"
	"github.com/Mogvl/dmbt-web/server/internal/filter"
	"github.com/Mogvl/dmbt-web/server/internal/model"
	"github.com/Mogvl/dmbt-web/server/internal/providers"
	"github.com/Mogvl/dmbt-web/server/internal/query"
	"github.com/Mogvl/dmbt-web/server/internal/resources"
	"github.com/Mogvl/dmbt-web/server/internal/subjects"
)

// System wires all modules, mirroring the original System.
type System struct {
	Cfg         *config.Config
	DB          *sql.DB
	Store       *resources.Store
	Query       *query.Query
	Subjects    *subjects.Module
	Collections *collections.Store
	Providers   *providers.Registry

	// Users/Teams lists for /users and /teams.
	Users  []model.UserBrief
	Teams  []model.UserBrief

	// Timestamp mirrors modules.providers.timestamp: latest provider refresh.
	Timestamp time.Time

	// AdminSecret mirrors sys.secret.
	AdminSecret string

	// Site host used in feed/detail URLs.
	Site string

	// Jobs executor (admin routes).
	Executor Executor
}

// GetResourceHref implements providers.System.
func (sys *System) GetResourceHref(provider, providerID string) (string, error) {
	return sys.Store.GetResourceHref(provider, providerID)
}

// ResourceIDsExist implements providers.System.
func (sys *System) ResourceIDsExist(provider string, ids []string) (map[string]bool, error) {
	return sys.Store.ResourceIDsExist(provider, ids)
}

// Executor executes admin-triggered jobs.
type Executor interface {
	FetchProvider(provider string) (any, int)
	SyncProvider(provider string, start, end int) (any, int)
	FetchAllProviders() (map[string]*model.Provider, error)
}

// Server is the HTTP server.
type Server struct {
	sys *System
}

// NewServer builds the HTTP handler.
func NewServer(sys *System) http.Handler {
	s := &Server{sys: sys}
	mux := http.NewServeMux()

	// middleware-wrapped routes
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = randomID()
		}
		w.Header().Set("X-Request-Id", requestID)

		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, PUT, POST, DELETE, PATCH, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}

		timestamp := s.sys.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now().UTC()
		}
		w.Header().Set("X-Response-Timestamp", timestamp.UTC().Format("2006-01-02T15:04:05.000Z07:00"))

		m := s.route(r)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		m(w, r)
	})

	mux.Handle("/", handler)
	return mux
}

// json writes a JSON response body (status defaults 200).
func (s *Server) json(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(body)
}

// errJSON writes the standard error envelope.
func (s *Server) errJSON(w http.ResponseWriter, status int, message string) {
	s.json(w, status, map[string]string{"status": "ERROR", "message": message})
}

func randomID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), randInt())
}

func randInt() int64 {
	return time.Now().UnixNano() % 1000000
}

// route dispatches to the per-path handlers.
func (s *Server) route(r *http.Request) func(http.ResponseWriter, *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/" || path == "/health":
		return s.handleRoot
	case path == "/users":
		return s.requireGET(s.handleUsers)
	case path == "/teams":
		return s.requireGET(s.handleTeams)
	case path == "/subjects":
		return s.requireGET(s.handleSubjects)
	case path == "/resources" || path == "/resources/":
		return s.handleResources
	case strings.HasPrefix(path, "/resources/"):
		return s.handleResources
	case strings.HasPrefix(path, "/resource/"):
		return s.handleProviderResource
	case strings.HasPrefix(path, "/detail/infohash/"):
		return s.handleInfoHash
	case strings.HasPrefix(path, "/detail/"):
		return s.handleProviderResource
	case path == "/collection":
		return s.handleCreateCollection
	case strings.HasPrefix(path, "/collection/"):
		rest := strings.TrimPrefix(path, "/collection/")
		if strings.HasSuffix(rest, "/feed.xml") {
			return s.handleCollectionFeed
		}
		return s.handleGetCollection
	case path == "/feed.xml":
		return s.handleFeed
	case path == "/sitemaps/subjects":
		return s.requireGET(s.handleSitemapSubjects)
	case strings.HasPrefix(path, "/sitemaps/"):
		return s.handleSitemapMonth
	case path == "/admin/providers" || strings.HasPrefix(path, "/admin/resources"):
		return s.handleAdmin
	case path == "/mcp":
		return s.handleMCP
	case path == "/.well-known/mcp/server-card.json":
		return s.handleServerCard
	}
	return nil
}

func (s *Server) requireGET(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(405)
			return
		}
		h(w, r)
	}
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	timestamp := s.sys.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	providersMap := map[string]*model.Provider{}
	for _, p := range s.sys.Providers.All() {
		providersMap[p.ID] = &model.Provider{
			ID:          p.ID,
			Name:        p.Name,
			RefreshedAt: timestamp,
			IsActive:    true,
		}
	}
	s.json(w, 200, map[string]any{
		"status":    "OK",
		"timestamp": timestamp.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		"providers": providersMap,
	})
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=86400")
	s.json(w, 200, map[string]any{"status": "OK", "users": s.sys.Users})
}

func (s *Server) handleTeams(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=86400")
	s.json(w, 200, map[string]any{"status": "OK", "teams": s.sys.Teams})
}

func (s *Server) handleSubjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=86400")
	subs := make([]map[string]any, 0, len(s.sys.Subjects.ActiveSubjects))
	for _, sub := range s.sys.Subjects.ActiveSubjects {
		subs = append(subs, map[string]any{
			"id":         sub.ID,
			"name":       sub.Name,
			"keywords":   sub.Keywords,
			"activedAt":  sub.ActivedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
			"isArchived": sub.IsArchived,
		})
	}
	s.json(w, 200, map[string]any{"status": "OK", "subjects": subs})
}

// handleResources ports the /resources routes (any method).
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	pagination, f := s.parseResourcesRequest(r)

	// deep pagination guard
	offset := (pagination.Page - 1) * pagination.PageSize
	if offset+pagination.PageSize > 10000 {
		s.errJSON(w, 400, "Resources pagination is too deep. Please keep offset + limit <= 10000.")
		return
	}

	resp, err := s.sys.Query.Find(f, pagination.Page, pagination.PageSize)
	if err != nil {
		log.Printf("resources query error: %v", err)
		s.errJSON(w, 500, "")
		return
	}

	// tracker / metadata flags
	enableTracker := isEnable(r, "tracker")
	enableMetadata := isEnable(r, "metadata")

	resourcesOut := make([]model.Resource, 0, len(resp.Resources))
	for _, res := range resp.Resources {
		if !enableTracker {
			res.Tracker = nil
		}
		if !enableMetadata {
			res.Metadata = nil
		}
		resourcesOut = append(resourcesOut, res)
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	s.json(w, 200, map[string]any{
		"status":     "OK",
		"complete":   resp.Pagination.Complete,
		"resources":  resourcesOut,
		"pagination": resp.Pagination,
		"filter":     resp.Filter,
	})
}

// parseResourcesRequest ports parseURLSearch with optional JSON body.
func (s *Server) parseResourcesRequest(r *http.Request) (filter.Pagination, filter.FilterOptions) {
	var body *filter.BodyOptions
	if (r.Method == http.MethodPost || r.Method == http.MethodPut) && r.Body != nil {
		dec := json.NewDecoder(r.Body)
		var raw map[string]any
		if err := dec.Decode(&raw); err == nil {
			body = parseBodyOptions(raw)
		}
	}
	result := filter.ParseURLSearch(r.URL.Query(), body)
	return result.Pagination, result.Filter
}

func isEnable(r *http.Request, key string) bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "true", "yes", "on":
		return true
	}
	return false
}

var _ = filter.DefaultPageSize