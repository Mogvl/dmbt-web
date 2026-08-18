package api

import (
	"io"
	"net/http"
	"strings"
	"time"
)

// BGMProxy forwards the web frontend's bangumi-mirror requests to
// bgm.animes.garden (the original web SSR proxies bgmx clients).
var bgmClient = &http.Client{Timeout: 60 * time.Second}

const bgmUpstream = "https://bgm.animes.garden"

// proxyToBGM mirrors a request path+query upstream.
func (s *Server) proxyToBGM(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/bgmx") {
		path = strings.TrimPrefix(path, "/bgmx")
	}
	url := bgmUpstream + path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		s.errJSON(w, 500, "")
		return
	}
	req.Header.Set("User-Agent", "dmbt-web/0.1")
	req.Header.Set("Accept", "application/json")
	resp, err := bgmClient.Do(req)
	if err != nil {
		s.errJSON(w, 502, "")
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.errJSON(w, 502, "")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
