// package: rest_http / transport
// type:    logic
// job:     the write endpoint — POST /contribute
// limits:  translation only; the merge runs behind core.Handle (-> internal/core)
package rest_http

import (
	"net/http"

	"github.com/flocko-motion/rankedb/api"
	"github.com/flocko-motion/rankedb/internal/core"
)

// Contribute serves POST /contribute?branch=.
func (s *Server) Contribute(w http.ResponseWriter, r *http.Request, params api.ContributeParams) {
	req := &core.Request{
		Credential: credentialOf(r.Context()),
		Op:         core.OpContribute,
		Branch:     params.Branch,
		Body:       r.Body,
	}
	if err := s.core.Handle(r.Context(), req); err != nil {
		s.fail(w, err)
		return
	}
	res, ok := req.Response.(core.Contribution)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, api.ContributionResult{
		Head: idString(res.Head),
		Ids:  idStrings(res.Ids),
	})
}
