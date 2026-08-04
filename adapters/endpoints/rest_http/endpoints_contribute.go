// package: rest_http / transport
// type:    logic
// job:     the write endpoint — POST /contribute
// limits:  translation only; the merge runs behind core.Handle (-> internal/core)
package rest_http

import (
	"net/http"

	"github.com/flocko-motion/rankedb/internal/core"
)

// Contribute serves POST /contribute. The body names the branch each claim joins, so
// core reads them from it and checks the C right on each.
func (s *Server) Contribute(w http.ResponseWriter, r *http.Request) {
	req := &core.Request{
		Credential: credentialOf(r.Context()),
		Op:         core.OpClaimContribute,
		Body:       r.Body,
	}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusCreated)
}
