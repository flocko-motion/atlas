// package: rest_http / transport
// type:    adapter
// job:     resolve a request credential to a Subject and carry it in the request context
// limits:  authentication only; the access decision on the subject is the core's (-> coreapi)
package rest_http

import (
	"context"
	"net/http"
	"strings"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/adapters/endpoints/coreapi"
)

type ctxKey int

const subjectKey ctxKey = iota

// withAuth resolves the request's Subject once, before routing, and stashes it in
// the context for the handlers. A credential that no authenticator accepts is 401.
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subj, err := s.authenticate(r.Context(), extractCredential(r))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), subjectKey, subj)))
	})
}

// authenticate resolves a credential to a Subject.
//
// TODO: route by the presented scheme to the matching authenticator (Bearer→JWT,
// X-API-Key→apikey, Macaroon→macaroon, none→noauth) once the auth port exposes
// its scheme. For now it tries each configured authenticator in turn.
func (s *Server) authenticate(ctx context.Context, cred string) (coreapi.Subject, error) {
	for _, a := range s.auths {
		if subj, err := a.Authenticate(ctx, cred); err == nil {
			return coreapi.Subject(subj), nil
		}
	}
	return "", auth.ErrUnauthenticated
}

// extractCredential pulls the raw credential the authenticators expect: the token
// after the scheme in an Authorization header, else the X-API-Key value, else "".
func extractCredential(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if i := strings.IndexByte(h, ' '); i > 0 {
			return h[i+1:]
		}
		return h
	}
	return r.Header.Get("X-API-Key")
}

func subjectOf(ctx context.Context) coreapi.Subject {
	s, _ := ctx.Value(subjectKey).(coreapi.Subject)
	return s
}
