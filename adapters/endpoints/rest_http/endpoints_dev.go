// package: rest_http / transport
// type:    logic
// job:     the dev-only endpoint — POST /dev/clock
// limits:  translation only; whether the capability exists is core's (-> internal/core)
package rest_http

import (
	"encoding/json"
	"net/http"

	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/openapi"
)

// AdvanceDevClock serves POST /dev/clock.
func (s *Server) AdvanceDevClock(w http.ResponseWriter, r *http.Request) {
	var body openapi.DevClockAdvance
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, core.CatInvalid, "malformed dev clock advance")
		return
	}
	req := &core.Request{
		Credential: credentialOf(r.Context()),
		Op:         core.OpDevClockAdvance,
		DevClockAt: body.Time,
	}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}
