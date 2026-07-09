// package: rest_http / transport
// type:    logic
// job:     the read endpoints — POST /query and the cacheable by-id GET reads — with wire→core query mapping
// limits:  translation only; the reads run behind core.Handle and render inside the Stream (-> internal/core)
package rest_http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ranke "github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/openapi"
)

// Query serves POST /query: it runs the declarative query and streams the results.
func (s *Server) Query(w http.ResponseWriter, r *http.Request) {
	var q openapi.Query
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeError(w, core.CatInvalid, "malformed query")
		return
	}
	cq, err := coreQuery(q)
	if err != nil {
		writeError(w, core.CatInvalid, err.Error())
		return
	}
	// A branch-rooted read authorizes R on that branch; a claim-rooted read with no
	// branch is the privileged $universe read.
	branch := cq.Select.Branch
	if branch == "" {
		branch = core.Universe
	}
	req := &core.Request{
		Credential: credentialOf(r.Context()),
		Op:         core.OpClaimQuery,
		Branch:     branch,
		Query:      &cq,
	}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// GetBranchHead serves GET /{branch}/head.
func (s *Server) GetBranchHead(w http.ResponseWriter, r *http.Request, branch string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpBranchHead, Branch: branch}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// GetBranchClaim serves GET /{branch}/claim/{id}.
func (s *Server) GetBranchClaim(w http.ResponseWriter, r *http.Request, branch string, idParam string) {
	id, err := ranke.ParseId(idParam)
	if err != nil {
		writeError(w, core.CatNotFound, "not found")
		return
	}
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpClaimGet, Branch: branch, ClaimID: id}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// GetBranchClaimContent serves GET /{branch}/claim/{id}/content — the content of
// claim {id} within branch {name}'s closure (inline or blob, resolved by core).
func (s *Server) GetBranchClaimContent(w http.ResponseWriter, r *http.Request, branch string, idParam string) {
	id, err := ranke.ParseId(idParam)
	if err != nil {
		writeError(w, core.CatNotFound, "not found")
		return
	}
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpClaimContent, Branch: branch, ClaimID: id}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// GetUniverseClaim serves GET /$universe/claim/{id}.
func (s *Server) GetUniverseClaim(w http.ResponseWriter, r *http.Request, idParam string) {
	id, err := ranke.ParseId(idParam)
	if err != nil {
		writeError(w, core.CatNotFound, "not found")
		return
	}
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpClaimGet, Branch: core.Universe, ClaimID: id}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// GetUniverseClaimContent serves GET /$universe/claim/{id}/content — the content
// of claim {id}, privileged.
func (s *Server) GetUniverseClaimContent(w http.ResponseWriter, r *http.Request, idParam string) {
	id, err := ranke.ParseId(idParam)
	if err != nil {
		writeError(w, core.CatNotFound, "not found")
		return
	}
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpClaimContent, Branch: core.Universe, ClaimID: id}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// coreQuery maps the wire query to the core query. The wire encoding is not part
// of the core query — the Stream renders in the negotiated form.
func coreQuery(q openapi.Query) (core.Query, error) {
	var cq core.Query

	if q.Select.Branch != nil {
		cq.Select.Branch = *q.Select.Branch
	}
	if q.Select.Claim != nil {
		id, err := ranke.ParseId(*q.Select.Claim)
		if err != nil {
			return cq, fmt.Errorf("select.claim: invalid id")
		}
		cq.Select.Claim = id
	}
	if q.Select.Path != nil {
		for _, p := range *q.Select.Path {
			step := core.PathStep{Edges: p.Edges}
			if p.Dir != nil {
				step.Dir = core.Direction(string(*p.Dir))
			}
			if p.Depth != nil {
				step.Depth = *p.Depth
			}
			if p.Nodes != nil {
				step.Nodes = *p.Nodes
			}
			cq.Select.Path = append(cq.Select.Path, step)
		}
	}

	if q.Output != nil {
		if q.Output.Detail != nil {
			cq.Output.Detail = core.Detail(string(*q.Output.Detail))
		}
		if q.Output.Overflow != nil {
			cq.Output.Overflow = core.Overflow(string(*q.Output.Overflow))
		}
		// TODO: q.Output.Content is a bool|int|string union; resolve it (and a
		// human size like "4kb") to the core.Output.Content byte cap.
	}

	if q.Order != nil {
		cq.Order = &core.Order{
			Field: q.Order.Field,
			Desc:  q.Order.Dir != nil && string(*q.Order.Dir) == "desc",
		}
	}

	if q.Limit != nil {
		if q.Limit.Results != nil {
			cq.Limit.Results = *q.Limit.Results
		}
		if q.Limit.Time != nil {
			if d, err := time.ParseDuration(*q.Limit.Time); err == nil {
				cq.Limit.Time = d
			}
		}
	}

	if q.Execution != nil {
		if q.Execution.Layer != nil {
			cq.Execution.Layer = *q.Execution.Layer
		}
		if q.Execution.Report != nil {
			cq.Execution.Report = *q.Execution.Report
		}
	}

	// TODO: q.Where is a boolean-tree union (and/or/not/field-map); map it to
	// cq.Where. Until then a query filters nothing.

	return cq, nil
}
