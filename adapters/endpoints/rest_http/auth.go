// package: rest_http / transport
// type:    logic
// job:     extract the request's auth credential from the wire and carry it to the handlers
// limits:  extraction only; core resolves it and applies grants (-> internal/core)
package rest_http

import (
	"context"
	"net/http"
	"strings"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/internal/core"
)

type ctxKey int

const credentialKey ctxKey = iota

// withCredential extracts the request's auth credential once, before routing, and
// stashes it for the handlers to hand core. Presenting more than one scheme is a
// 400 (ambiguous); presenting none yields the zero credential, which core resolves
// as NoAuth.
func (s *Server) withCredential(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cred, err := extractCredential(r)
		if err != nil {
			writeError(w, core.CatInvalid, "ambiguous credentials")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), credentialKey, cred)))
	})
}

// extractCredential reads the one credential the request presents, tagged with its
// scheme: an Authorization Bearer/Macaroon token, or an X-API-Key value. More than
// one is auth.ErrAmbiguousCredentials; none is the zero value. Routing by scheme
// (not sniffing the token) is what lets the auth Set dispatch to the right backend.
func extractCredential(r *http.Request) (auth.Credential, error) {
	var creds []auth.Credential
	if h := r.Header.Get("Authorization"); h != "" {
		scheme, token := auth.SchemeBearer, h
		if i := strings.IndexByte(h, ' '); i > 0 {
			switch strings.ToLower(h[:i]) {
			case "macaroon":
				scheme = auth.SchemeMacaroon
			default:
				scheme = auth.SchemeBearer
			}
			token = h[i+1:]
		}
		creds = append(creds, auth.Credential{Scheme: scheme, Token: token})
	}
	if k := r.Header.Get("X-API-Key"); k != "" {
		creds = append(creds, auth.Credential{Scheme: auth.SchemeAPIKey, Token: k})
	}
	switch len(creds) {
	case 0:
		return auth.Credential{}, nil
	case 1:
		return creds[0], nil
	default:
		return auth.Credential{}, auth.ErrAmbiguousCredentials
	}
}

// credentialOf returns the credential withCredential stashed on the request context.
func credentialOf(ctx context.Context) auth.Credential {
	c, _ := ctx.Value(credentialKey).(auth.Credential)
	return c
}
