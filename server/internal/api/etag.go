package api

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
)

// captureWriter buffers the whole response so an ETag can be computed and
// conditional requests (If-None-Match) answered with 304, mirroring the
// original safeEtag middleware. Nothing reaches the client until Finish.
type captureWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newCaptureWriter() *captureWriter {
	return &captureWriter{header: http.Header{}, status: http.StatusOK}
}

func (w *captureWriter) Header() http.Header { return w.header }

func (w *captureWriter) WriteHeader(status int) { w.status = status }

func (w *captureWriter) Write(b []byte) (int, error) { return w.body.Write(b) }

// Finish copies the buffered response to the real writer.
func (w *captureWriter) Finish(dst http.ResponseWriter) {
	for k, vs := range w.header {
		for _, v := range vs {
			dst.Header().Add(k, v)
		}
	}
	dst.WriteHeader(w.status)
	dst.Write(w.body.Bytes())
}

// withEtag wraps the handler: on 2xx GET responses it sets a weak ETag
// (W/"sha1") and short-circuits with 304 when the client already has it.
func withEtag(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			h(w, r)
			return
		}
		cw := newCaptureWriter()
		h(cw, r)

		etag := ""
		if cw.status >= 200 && cw.status < 300 && cw.body.Len() > 0 {
			sum := sha1.Sum(cw.body.Bytes())
			etag = `W/"` + hex.EncodeToString(sum[:]) + `"`
		}
		if etag != "" {
			cw.header.Set("ETag", etag)
			if noneMatch := r.Header.Get("If-None-Match"); noneMatch != "" && noneMatch == etag {
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		cw.Finish(w)
	}
}
