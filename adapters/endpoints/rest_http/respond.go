// package: rest_http / transport
// type:    adapter
// job:     write responses, map core sentinel errors to HTTP status, carry claim/content bodies
// limits:  serialization only; the domain values come from core (-> internal/core)
package rest_http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	ranke "github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/api"
	"github.com/flocko-motion/rankedb/internal/core"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, api.Error{Error: msg})
}

// fail maps a core (or auth) sentinel error to its HTTP status.
func (s *Server) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, core.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, core.ErrForbidden):
		writeError(w, http.StatusForbidden, "access denied")
	case errors.Is(err, core.ErrConflict):
		writeError(w, http.StatusConflict, "conflict with the current head")
	case errors.Is(err, core.ErrBusy):
		w.Header().Set("Retry-After", "30")
		writeError(w, http.StatusTooManyRequests, "verification run limit reached")
	case errors.Is(err, core.ErrNotImplemented):
		writeError(w, http.StatusNotImplemented, "capability not configured")
	case errors.Is(err, auth.ErrUnauthenticated), errors.Is(err, auth.ErrAmbiguousCredentials):
		writeError(w, http.StatusUnauthorized, "unauthenticated")
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// writeClaim streams a claim's canonical CBOR, cacheable immutably by its id.
func writeClaim(w http.ResponseWriter, claim ranke.Claim) {
	b, err := claim.Encode()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encode claim")
		return
	}
	w.Header().Set("Content-Type", "application/cbor")
	w.Header().Set("ETag", `"`+claim.ID().String()+`"`)
	w.Header().Set("Cache-Control", "public, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

// streamContent copies a content blob to w as raw bytes, cacheable by hash.
func streamContent(w http.ResponseWriter, hash string, rc io.ReadCloser) {
	defer rc.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", `"`+hash+`"`)
	w.Header().Set("Cache-Control", "public, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

func idString(id ranke.Id) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func idStrings(ids []ranke.Id) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, idString(id))
	}
	return out
}
