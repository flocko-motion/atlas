// package: rest_http / transport
// type:    logic
// job:     answer a browser's cross-origin checks for the origins the config admits
// limits:  transport only; which origins are admitted is policy, and policy is config's
//
// A browser refuses a cross-origin read unless the server says the origin may have it, so
// without this an explorer served from anywhere but the API's own origin cannot call it at
// all. Which origins may is an operational choice about this deployment, so it is declared
// in the endpoint's config rather than decided here — and declaring none keeps a server
// that no browser is meant to reach exactly as closed as it is today.
package rest_http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/flocko-motion/rankedb/config/scope"
)

// The cross-origin answers. Requests carry a credential in a header rather than a cookie,
// so no origin is ever trusted with credentialed access — an allowed origin may read the
// API with the token it was given, and nothing rides along on the browser's behalf.
const (
	corsHeaders = "Content-Type, Authorization, X-API-Key, If-None-Match"
	corsMethods = "GET, POST, DELETE, OPTIONS"
	// ETag drives the conditional reads the contract promises, and a script cannot see a
	// response header unless it is exposed.
	corsExpose = "ETag"
	// How long a browser may cache the preflight answer.
	corsMaxAge = "600"
)

// corsPolicy is the set of origins this endpoint admits, as the config declared them.
type corsPolicy struct {
	origins map[string]bool
	// any is the wildcard: every origin, which suits a dev instance and no other.
	any bool
}

// corsFrom reads the transport section's "allowedOrigins": a comma-separated list, so it is
// one leaf value and an `env()` reference can supply it per deployment. Absent or empty
// means no cross-origin access, which is what a server nobody browses should say. `*`
// admits every origin, which suits a dev instance and nothing else.
func corsFrom(ctx context.Context, cfg scope.Section) (corsPolicy, error) {
	p := corsPolicy{origins: map[string]bool{}}
	if !cfg.HasValue("allowedOrigins") {
		return p, nil
	}
	declared, err := cfg.Get(ctx, "allowedOrigins")
	if err != nil {
		return p, fmt.Errorf("rest_http: allowedOrigins: %w", err)
	}
	for _, origin := range strings.Split(declared, ",") {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		switch {
		case origin == "":
			continue
		case origin == "*":
			p.any = true
		default:
			p.origins[origin] = true
		}
	}
	return p, nil
}

// enabled reports whether any origin is admitted.
func (p corsPolicy) enabled() bool { return p.any || len(p.origins) > 0 }

// allows reports whether this origin may read the API.
func (p corsPolicy) allows(origin string) bool {
	return p.any || p.origins[strings.TrimRight(origin, "/")]
}

// withCORS answers the browser's checks ahead of routing: a preflight is answered here and
// goes no further, and an allowed cross-origin read carries the headers that let the script
// see the answer. A request with no Origin is untouched — curl and the server's own clients
// never meet this.
func (p corsPolicy) withCORS(next http.Handler) http.Handler {
	if !p.enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !p.allows(origin) {
			// Say nothing rather than deny: an answer without the header is what a browser
			// reads as refused, and the request itself is none of CORS's business.
			next.ServeHTTP(w, r)
			return
		}

		h := w.Header()
		// Echo the origin even for the wildcard, so a cache keyed on Vary stays correct.
		h.Set("Access-Control-Allow-Origin", origin)
		h.Add("Vary", "Origin")
		h.Set("Access-Control-Expose-Headers", corsExpose)

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			h.Set("Access-Control-Allow-Methods", corsMethods)
			h.Set("Access-Control-Allow-Headers", corsHeaders)
			h.Set("Access-Control-Max-Age", corsMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
