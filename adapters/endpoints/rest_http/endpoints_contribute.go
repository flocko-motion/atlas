// package: rest_http / transport
// type:    logic
// job:     the write endpoint — POST /contribute
// limits:  translation only; the merge runs behind core.Handle (-> internal/core)
package rest_http

import (
	"net/http"

	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/openapi"
)

// Contribute serves POST /contribute?branch=.
func (s *Server) Contribute(w http.ResponseWriter, r *http.Request, params openapi.ContributeParams) {
	req := &core.Request{
		Credential: credentialOf(r.Context()),
		Op:         core.OpClaimContribute,
		Branch:     params.Branch,
		Body:       r.Body,
	}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusCreated)
}
