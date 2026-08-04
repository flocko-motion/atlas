// package: rest_http / transport
// type:    adapter
// job:     the HTTP envelope — content-type + status + body copy, and error → status mapping
// limits:  framing/rendering live in core's Stream; this sets headers/status and copies (-> internal/core)
package rest_http

import (
	"encoding/json"
	"net/http"

	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/openapi"
)

// respond writes a successful response: the stream's content-type, the route's
// status, then the body — framing and rendering happen inside the stream's WriteTo,
// so this stays wire-oblivious. It always closes the stream.
func (s *Server) respond(w http.ResponseWriter, stream core.Stream, status int) {
	defer stream.Close()
	w.Header().Set("Content-Type", stream.ContentType())
	w.WriteHeader(status)
	_, _ = stream.WriteTo(w) // status already sent; a mid-stream error can only be logged
}

// fail maps a core (or auth) error to its response via the transport-neutral
// category — the category fixes the HTTP status and ships as the machine-readable
// code, the error's text as the human message.
func (s *Server) fail(w http.ResponseWriter, err error) {
	writeError(w, core.Categorize(err), err.Error())
}

// httpStatus maps a core.Category to its HTTP status — the one place status
// numbers live.
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

// writeError writes an error body at the category's HTTP status, carrying the
// category as the stable machine-readable code and msg as the human message. Every
// error path — transport-level (a malformed body) and core (via fail) — goes
// through a category, so status and code always agree.
func writeError(w http.ResponseWriter, cat core.Category, msg string) {
	if cat == core.CatBusy {
		// A refusal for want of a free slot is worth retrying, so say when.
		w.Header().Set("Retry-After", retryAfterSeconds)
	}
	writeJSON(w, httpStatus(cat), openapi.Error{Code: string(cat), Error: msg})
}

// mediaJSON is the content type a JSON response carries.
const mediaJSON = "application/json"

// respondBytes writes a body this package already holds, for the routes that must read it
// to derive a header before sending it.
func (s *Server) respondBytes(w http.ResponseWriter, contentType string, status int, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
