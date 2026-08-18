package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Mogvl/dmbt-web/server/internal/model"
)

// handleAdmin ports /admin routes. The original registers bearerAuth on the
// literal path '/admin/' only (which the endpoints never match in practice),
// so admin operations are effectively unprotected — replicated here.
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/")
	parts := strings.SplitN(path, "/", 2)
	resource := parts[0]

	switch resource {
	case "providers":
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		providers, err := s.sys.Executor.FetchAllProviders()
		if err != nil {
			s.errJSON(w, 500, "")
			return
		}
		s.json(w, 200, map[string]any{"status": "OK", "providers": providers})

	case "resources":
		if len(parts) != 2 {
			w.WriteHeader(404)
			return
		}
		sub := strings.SplitN(parts[1], "/", 2)
		providerName := sub[0]
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if len(sub) == 1 {
			// POST /admin/resources/:provider
			body, status := s.sys.Executor.FetchProvider(providerName)
			w.WriteHeader(status)
			s.json(w, status, body)
			return
		}
		if sub[1] == "sync" {
			// POST /admin/resources/:provider/sync?start=&end=
			start, _ := strconv.Atoi(r.URL.Query().Get("start"))
			if start == 0 {
				start = 1
			}
			end, _ := strconv.Atoi(r.URL.Query().Get("end"))
			if end == 0 {
				end = 10
			}
			body, status := s.sys.Executor.SyncProvider(providerName, start, end)
			w.WriteHeader(status)
			s.json(w, status, body)
			return
		}
		w.WriteHeader(404)
	default:
		w.WriteHeader(404)
	}
}

var _ = model.Provider{}