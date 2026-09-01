// package: rest_http / transport
// type:    adapter
// job:     the HTTP envelope — content-type + status + body copy, and error → status mapping
// limits:  framing/rendering live in core's Stream; this sets headers/status and copies (-> internal/core)
package rest_http

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rankegraph/ranke-db/internal/core"
	"github.com/rankegraph/ranke-db/openapi"
)

// respond writes the stream's content-type, the route's status, then the body. Framing
// happens inside WriteTo, so this stays wire-oblivious. Always closes the stream.
func (s *Server) respond(w http.ResponseWriter, stream core.Stream, status int) {
	defer stream.Close()
	w.Header().Set("Content-Type", stream.ContentType())
	w.WriteHeader(status)
	_, _ = stream.WriteTo(w) // status already sent; a mid-stream error can only be logged
}

// fail answers an error through its transport-neutral category, which fixes the status
// and ships as the machine-readable code.
func (s *Server) fail(w http.ResponseWriter, err error) {
	writeError(w, core.Categorize(err), err.Error())
}

// httpStatus is the one place HTTP status numbers live.
func httpStatus(cat core.Category) int {
	switch cat {
	case core.CatUnauthenticated:
		return http.StatusUnauthorized
	case core.CatForbidden:
		return http.StatusForbidden
	case core.CatNotFound:
		return http.StatusNotFound
	case core.CatConflict:
		return http.StatusConflict
	case core.CatBusy:
		return http.StatusTooManyRequests
	case core.CatInvalid:
		return http.StatusBadRequest
	case core.CatUnimplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes an error body at its category's status, the category doubling as the
// stable code. Every error path goes through one, so status and code always agree.
func writeError(w http.ResponseWriter, cat core.Category, msg string) {
	if cat == core.CatBusy {
		// A refusal for want of a free slot is worth retrying, so say when.
		w.Header().Set("Retry-After", retryAfterSeconds)
	}
	writeJSON(w, httpStatus(cat), openapi.Error{Code: string(cat), Error: msg})
}

// mediaJSON is the content type a JSON response carries.
const mediaJSON = "application/json"

// The cache posture the contract declares: a by-id read is immutable, its id
// content-addressing the bytes; a head or listing moves.
const (
	cacheImmutable  = "public, max-age=31536000, immutable"
	cacheRevalidate = "no-cache"
)

// respondImmutable answers a by-id read: the id is a strong validator, being the hash of
// what comes back. The 304 comes after the read, so caching changes no answer.
func (s *Server) respondImmutable(w http.ResponseWriter, r *http.Request, stream core.Stream, id string) {
	etag := `"` + id + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", cacheImmutable)
	if matches(r.Header.Get("If-None-Match"), etag) {
		_ = stream.Close()
		w.WriteHeader(http.StatusNotModified)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// respondRevalidated answers a read whose value moves. The body is read here to derive a
// weak validator, which is what makes the promised conditional request cheap.
func (s *Server) respondRevalidated(w http.ResponseWriter, r *http.Request, stream core.Stream) {
	defer stream.Close()
	var body bytes.Buffer
	if _, err := stream.WriteTo(&body); err != nil {
		s.fail(w, err)
		return
	}
	sum := sha256.Sum256(body.Bytes())
	etag := `W/"` + base64.RawURLEncoding.EncodeToString(sum[:16]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", cacheRevalidate)
	if matches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	s.respondBytes(w, stream.ContentType(), http.StatusOK, body.Bytes())
}

// matches reads an If-None-Match list, where `*` matches anything held (RFC 9110).
func matches(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

// respondBytes writes a body already in hand, for a route that read it to derive a header.
func (s *Server) respondBytes(w http.ResponseWriter, contentType string, status int, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
