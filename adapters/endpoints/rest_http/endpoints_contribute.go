// package: rest_http / transport
// type:    logic
// job:     the write endpoint — POST /contribute
// limits:  translation only; the merge runs behind coreapi.API (-> coreapi)
package rest_http

import (
	"net/http"

	"github.com/flocko-motion/rankedb/api"
)

// Contribute serves POST /contribute?branch=.
func (s *Server) Contribute(w http.ResponseWriter, r *http.Request, params api.ContributeParams) {
	res, err := s.core.Contribute(r.Context(), subjectOf(r.Context()), params.Branch, r.Body)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, api.ContributionResult{
		Head: idString(res.Head),
		Ids:  idStrings(res.Ids),
	})
}
