package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Mogvl/dmbt-web/server/internal/filter"
)

// handleServerCard ports the /.well-known/mcp/server-card.json 302 redirect.
func (s *Server) handleServerCard(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+s.sys.Site+"/.well-known/mcp/server-card.json", http.StatusFound)
}

// handleMCP ports the /mcp streamable-HTTP endpoint with a JSON-RPC
// implementation of the two registered capabilities (search_resources tool,
// resource_detail resource template).
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		s.errJSON(w, 400, "Please connect /mcp with MCP client")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// SSE stream — minimal: send an initialized endpoint event.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			s.errJSON(w, 500, "streaming unsupported")
			return
		}
		// The original opens a persistent SSE stream; we emit an empty
		// keepalive so clients can connect.
		w.Write([]byte("event: endpoint\ndata: /mcp\n\n"))
		flusher.Flush()
		select {} // block forever; the client disconnects eventually

	case http.MethodPost:
		var msg jsonrpcMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			s.errJSON(w, 400, "invalid JSON-RPC message")
			return
		}
		resp := s.handleMCPMessage(&msg)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(resp)

	case http.MethodDelete:
		w.WriteHeader(200)

	default:
		w.WriteHeader(405)
	}
}

type jsonrpcMessage struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

func (s *Server) handleMCPMessage(msg *jsonrpcMessage) any {
	switch msg.Method {
	case "initialize":
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      msg.ID,
			"result": map[string]any{
				"protocolVersion": "2025-03-26",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "animegarden",
					"version": "0.5.4",
				},
			},
		}
	case "notifications/initialized":
		return nil
	case "tools/list":
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      msg.ID,
			"result": map[string]any{
				"tools": []map[string]any{{
					"name":        "search_resources",
					"title":       "Search anime torrent resources aggregated from 動漫花園, 蜜柑计划, 萌番组, ANi with Anime Garden",
					"description": "Search anime resources with flexible filters. Supports AND-combination across different filter groups, and OR-combination within the same group (e.g. fansubs, publishers, types, subjects, include, exclude). The search option performs tokenized search and takes precedence over include; keywords requires the title to contain ALL values; exclude blocks resources containing ANY value.",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"fansubs":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Fansub group names. Match ANY value (OR)."},
							"publishers": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Publisher names. Match ANY value (OR). Combined with fansubs in OR logic within this group."},
							"types":      map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"动画", "合集", "音乐", "日剧", "RAW", "漫画", "游戏", "特摄", "其他"}}},
							"before":     map[string]any{"type": "string", "format": "date-time", "description": "Upper time bound (inclusive): createdAt <= before. Accepts date string or timestamp."},
							"after":      map[string]any{"type": "string", "format": "date-time", "description": "Lower time bound (inclusive): createdAt >= after. Accepts date string or timestamp."},
							"subjects":   map[string]any{"type": "array", "items": map[string]any{"type": "number"}, "description": "Bangumi subject IDs. Match ANY value (OR)."},
							"search":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Full-text query terms (tokenized search). Takes precedence over include."},
							"include":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Title-contains terms. Match ANY value (OR). Only effective when search is not provided."},
							"keywords":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Required title keywords. Title must contain ALL values (AND)."},
							"exclude":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Blocked title keywords. Exclude resources containing ANY value."},
						},
					},
				}},
			},
		}
	case "tools/call":
		params, _ := msg.Params.(map[string]any)
		name, _ := params["name"].(string)
		args, _ := params["arguments"].(map[string]any)
		if name == "search_resources" {
			return s.mcpSearchResources(msg.ID, args)
		}
		return mcpError(msg.ID, -32602, "Unknown tool: "+name)
	case "resources/read":
		params, _ := msg.Params.(map[string]any)
		uri, _ := params["uri"].(string)
		return s.mcpReadResource(msg.ID, uri)
	case "resources/list":
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      msg.ID,
			"result": map[string]any{
				"resources": []any{},
			},
		}
	case "prompts/list":
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      msg.ID,
			"result": map[string]any{
				"prompts": []any{},
			},
		}
	}
	return mcpError(msg.ID, -32601, "Method not found: "+msg.Method)
}

// mcpSearchResources ports the search_resources tool execution.
func (s *Server) mcpSearchResources(id any, args map[string]any) any {
	body := parseBodyOptions(args)
	result := filter.ParseURLSearch(nil, body)
	pagination, f := result.Pagination, result.Filter
	pagination.Page = 1
	pagination.PageSize = 30

	find, err := s.sys.Query.Find(f, 1, 30)
	if err != nil {
		return mcpError(id, -32603, "Internal error")
	}

	site := "https://" + s.sys.Site
	type item struct {
		ID        int64   `json:"id"`
		Provider  string  `json:"provider"`
		ProviderID string `json:"providerId"`
		Title     string  `json:"title"`
		URI       string  `json:"uri"`
		Href      string  `json:"href"`
		Type      string  `json:"type"`
		Magnet    string  `json:"magnet"`
		Size      int64   `json:"size"`
		CreatedAt string  `json:"createdAt"`
		Publisher string  `json:"publisher"`
		Fansub    *string `json:"fansub,omitempty"`
	}
	items := make([]item, 0, len(find.Resources))
	for _, res := range find.Resources {
		magnet := res.Magnet
		if res.Tracker != nil {
			magnet += *res.Tracker
		}
		it := item{
			ID:         res.ID,
			Provider:   res.Provider,
			ProviderID: res.ProviderID,
			Title:      res.Title,
			URI:        "animegarden://resources/" + res.Provider + "/" + res.ProviderID,
			Href:       site + "/detail/" + res.Provider + "/" + res.ProviderID,
			Type:       res.Type,
			Magnet:     magnet,
			Size:       res.Size,
			CreatedAt:  res.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
			Publisher:  res.Publisher.Name,
		}
		if res.Fansub != nil {
			name := res.Fansub.Name
			it.Fansub = &name
		}
		items = append(items, it)
	}

	text, _ := json.MarshalIndent(items, "", "  ")
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"content": []map[string]any{{
				"type": "text",
				"text": string(text),
			}},
			"structuredContent": map[string]any{
				"resources": items,
			},
		},
	}
}

// mcpReadResource ports the resource_detail template reader.
func (s *Server) mcpReadResource(id any, uri string) any {
	const prefix = "animegarden://resources/"
	if !strings.HasPrefix(uri, prefix) {
		body := map[string]any{
			"error":   "INVALID_RESOURCE_URI",
			"uri":     uri,
			"message": "Expected URI format: animegarden://resources/{provider}/{providerId}, provider in [dmhy, moe, mikan, ani].",
		}
		return mcpResourceContent(id, body)
	}
	rest := strings.TrimPrefix(uri, prefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		body := map[string]any{
			"error":   "INVALID_RESOURCE_URI",
			"uri":     uri,
			"message": "Expected URI format: animegarden://resources/{provider}/{providerId}, provider in [dmhy, moe, mikan, ani].",
		}
		return mcpResourceContent(id, body)
	}
	providerName := parts[0]
	providerID := parts[1]
	if _, ok := s.sys.Providers.Get(providerName); !ok || providerID == "" {
		body := map[string]any{
			"error":   "INVALID_RESOURCE_URI",
			"uri":     uri,
			"message": "Expected URI format: animegarden://resources/{provider}/{providerId}, provider in [dmhy, moe, mikan, ani].",
		}
		return mcpResourceContent(id, body)
	}

	row, err := s.sys.Store.GetByProviderID(providerName, providerID)
	if err != nil || row == nil {
		return mcpResourceContent(id, map[string]any{
			"error":      "RESOURCE_NOT_FOUND",
			"provider":   providerName,
			"providerId": providerID,
		})
	}

	site := "https://" + s.sys.Site
	out := map[string]any{
		"id":         row.ID,
		"provider":   row.Provider,
		"providerId": row.ProviderID,
		"title":      row.Title,
		"uri":        uri,
		"href":       site + "/detail/" + row.Provider + "/" + row.ProviderID,
		"type":       row.Type,
		"size":       row.Size,
		"createdAt":  row.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	}
	magnet := row.Magnet
	if row.Tracker != "" {
		magnet += row.Tracker
	}
	out["magnet"] = magnet
	if name, ok := s.sys.Store.UserIDToName[row.PublisherID]; ok {
		out["publisher"] = name
	}
	if row.FansubID.Valid {
		if name, ok := s.sys.Store.TeamIDToName[row.FansubID.Int64]; ok {
			out["fansub"] = name
		}
	}
	// detail
	if detailRow, _ := s.sys.Store.GetDetail(row.ID); detailRow != nil {
		var files []map[string]any
		json.Unmarshal([]byte(detailRow.Files), &files)
		out["description"] = detailRow.Description
		out["files"] = files
		out["hasMoreFiles"] = detailRow.HasMoreFiles
	}
	return mcpResourceContent(id, out)
}

func mcpResourceContent(id any, body map[string]any) any {
	text, _ := json.MarshalIndent(body, "", "  ")
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"contents": []map[string]any{{
				"uri":      "",
				"mimeType": "application/json",
				"text":     string(text),
			}},
		},
	}
}

func mcpError(id any, code int, message string) any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}