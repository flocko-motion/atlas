// package: rest_http / transport
// type:    logic
// job:     the read endpoints — POST /query and the cacheable by-id GET reads — with wire→RQL query mapping
// limits:  translation only; the language is ranke-go's, the reads core's (-> internal/core)
package rest_http

import (
	"encoding/json"
	"errors"
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
	rq, err := rankeQuery(q)
	if err != nil {
		// A malformed query is the caller's fault (400). One asking for a capability the
		// bound library does not offer is not, and reports as unconfigured (501).
		cat := core.CatInvalid
		if errors.Is(err, core.ErrNotImplemented) {
			cat = core.CatUnimplemented
		}
		writeError(w, cat, err.Error())
		return
	}
	// The scope is what the grant is held against: a branch name, $archive, or the
	// privileged $universe.
	req := &core.Request{
		Credential: credentialOf(r.Context()),
		Op:         core.OpClaimQuery,
		Branch:     rq.Select.Branch,
		Query:      &rq,
	}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// ListBranches serves GET /branches. The scope is what the grant is held against, so
// the request names the reserved branch table: R on $branches is what admits it.
func (s *Server) ListBranches(w http.ResponseWriter, r *http.Request) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpBranchList, Branch: core.Branches}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respondRevalidated(w, r, stream)
}

// GetBranchHead serves GET /branches/{branch}/head.
func (s *Server) GetBranchHead(w http.ResponseWriter, r *http.Request, branch string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpBranchHead, Branch: branch}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respondRevalidated(w, r, stream)
}

// GetBranchClaim serves GET /branches/{branch}/claims/{id}.
func (s *Server) GetBranchClaim(w http.ResponseWriter, r *http.Request, branch string, idParam string) {
	s.claim(w, r, core.OpClaimGet, branch, idParam)
}

// GetBranchClaimContent serves GET /branches/{branch}/claims/{id}/content — the content
// of claim {id} within that branch's closure (inline or blob, resolved by core).
func (s *Server) GetBranchClaimContent(w http.ResponseWriter, r *http.Request, branch string, idParam string) {
	s.claim(w, r, core.OpClaimContent, branch, idParam)
}

// GetArchiveClaim serves GET /archive/claims/{id} — the $archive scope, the closure of
// the current head across every branch.
func (s *Server) GetArchiveClaim(w http.ResponseWriter, r *http.Request, idParam string) {
	s.claim(w, r, core.OpClaimGet, core.Archive, idParam)
}

// GetArchiveClaimContent serves GET /archive/claims/{id}/content.
func (s *Server) GetArchiveClaimContent(w http.ResponseWriter, r *http.Request, idParam string) {
	s.claim(w, r, core.OpClaimContent, core.Archive, idParam)
}

// GetClaim serves GET /universe/claims/{id} — the $universe scope, reached under no
// closure. The path drops the `$`, the grant checked does not.
func (s *Server) GetClaim(w http.ResponseWriter, r *http.Request, idParam string) {
	s.claim(w, r, core.OpClaimGet, core.Universe, idParam)
}

// GetClaimContent serves GET /universe/claims/{id}/content.
func (s *Server) GetClaimContent(w http.ResponseWriter, r *http.Request, idParam string) {
	s.claim(w, r, core.OpClaimContent, core.Universe, idParam)
}

// claim serves one by-id read in the scope its route named. Every such route is the same
// request but for its scope and whether it wants the claim or its content, and each is
// immutably cacheable: the id content-addresses the bytes.
func (s *Server) claim(w http.ResponseWriter, r *http.Request, op core.Operation, scope, idParam string) {
	id, err := ranke.ParseId(idParam)
	if err != nil {
		writeError(w, core.CatNotFound, "not found")
		return
	}
	req := &core.Request{Credential: credentialOf(r.Context()), Op: op, Branch: scope, ClaimID: id}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respondImmutable(w, r, stream, id.String())
}

// rankeQuery maps the wire query onto ranke-go's RQL, which the engine executes
// as-is. The wire binds that type field-for-field and spells every value as the
// library does, so this is a transcription — nothing folded, nothing restated.
func rankeQuery(q openapi.Query) (ranke.Query, error) {
	var rq ranke.Query

	if err := selectInto(&rq.Select, q.Select); err != nil {
		return rq, err
	}

	if q.Output != nil {
		if err := outputInto(&rq.Output, *q.Output); err != nil {
			return rq, err
		}
	}

	for _, k := range deref(q.Order) {
		key := ranke.OrderKey{Field: k.Field}
		if k.Compare != nil {
			key.Compare = ranke.Collation(*k.Compare)
		}
		if k.Dir != nil {
			key.Dir = ranke.SortDir(*k.Dir)
		}
		rq.Order = append(rq.Order, key)
	}

	if q.Limit != nil {
		if q.Limit.Results != nil {
			rq.Limit.Results = *q.Limit.Results
		}
		if q.Limit.Time != nil {
			d, err := time.ParseDuration(*q.Limit.Time)
			if err != nil {
				return rq, fmt.Errorf("limit.time: invalid duration %q", *q.Limit.Time)
			}
			rq.Limit.Time = d
		}
	}

	if q.Execution != nil {
		if q.Execution.Layer != nil {
			rq.Execution.Layer = *q.Execution.Layer
		}
		if q.Execution.Report != nil {
			rq.Execution.Report = ranke.ReportLevel(*q.Execution.Report)
		}
	}

	if q.Where != nil {
		w, err := whereOf(*q.Where)
		if err != nil {
			return rq, err
		}
		rq.Where = w
	}

	return rq, nil
}

// selectInto resolves the generator — scope, closure, start, traversal — and
// enforces the two rules the JSON schema cannot state: a scope is mandatory, and
// $universe confines nothing and so has no head to fall back on.
func selectInto(sel *ranke.Select, s openapi.Select) error {
	sel.Branch = s.Branch
	if sel.Branch == "" {
		return fmt.Errorf("select.branch: a scope is required")
	}
	if s.Head != nil {
		id, err := ranke.ParseId(*s.Head)
		if err != nil {
			return fmt.Errorf("select.head: invalid id")
		}
		sel.Head = id
	} else if sel.Branch == ranke.BranchUniverse {
		return fmt.Errorf("select.head: required under %s", ranke.BranchUniverse)
	}
	if s.Claim != nil {
		id, err := ranke.ParseId(*s.Claim)
		if err != nil {
			return fmt.Errorf("select.claim: invalid id")
		}
		sel.Claim = id
	}
	for _, p := range deref(s.Path) {
		step := ranke.PathStep{Edges: deref(p.Edges), Nodes: deref(p.Nodes)}
		if p.Dir != nil {
			step.Dir = ranke.Direction(*p.Dir)
		}
		if p.Min != nil {
			step.Min = ranke.Hops(*p.Min)
		}
		if p.Max != nil {
			step.Max = *p.Max
		}
		sel.Path = append(sel.Path, step)
	}
	return nil
}

// outputInto resolves the output shaping: each wire axis is one library axis.
func outputInto(out *ranke.Output, o openapi.Output) error {
	if o.Shape != nil {
		out.Shape = ranke.Shape(*o.Shape)
	}
	if o.Detail != nil {
		out.Detail = ranke.Detail(*o.Detail)
	}
	if o.Form != nil {
		out.Form = ranke.Form(*o.Form)
	}
	if o.Encoding != nil {
		out.Encoding = ranke.ResultEncoding(*o.Encoding)
	}
	if o.Content != nil {
		return errContentUnsupported
	}
	return nil
}

// errContentUnsupported refuses a query the bound ranke.Output has no field to carry,
// so the caller learns their cap went unapplied.
var errContentUnsupported = fmt.Errorf("%w: output.content", core.ErrNotImplemented)

// whereOf maps the wire's boolean tree onto the library's. The wire models the
// tree as a oneOf — and/or/not subtrees, or a {field, test} leaf — so the variant
// is found by which key is present.
func whereOf(w openapi.Where) (*ranke.Where, error) {
	if and, err := w.AsWhere0(); err == nil && len(and.And) > 0 {
		subs, err := whereList(and.And)
		if err != nil {
			return nil, err
		}
		return &ranke.Where{And: subs}, nil
	}
	if or, err := w.AsWhere1(); err == nil && len(or.Or) > 0 {
		subs, err := whereList(or.Or)
		if err != nil {
			return nil, err
		}
		return &ranke.Where{Or: subs}, nil
	}
	if not, err := w.AsWhere2(); err == nil {
		if sub, err := whereOf(not.Not); err == nil && sub != nil {
			return &ranke.Where{Not: sub}, nil
		}
	}
	leaf, err := w.AsWhere3()
	if err != nil || leaf.Field == "" {
		return nil, fmt.Errorf("where: want and/or/not, or a {field, test} leaf")
	}
	return &ranke.Where{Field: leaf.Field, Test: comparisonOf(leaf.Test)}, nil
}

// whereList maps a subtree list, flattening the pointers the recursion returns.
func whereList(ws []openapi.Where) ([]ranke.Where, error) {
	out := make([]ranke.Where, 0, len(ws))
	for _, w := range ws {
		sub, err := whereOf(w)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, nil
}

// comparisonOf maps one field's comparison. The operators are passed through as
// the untyped values the wire carried; the engine decides how each compares.
func comparisonOf(c openapi.Comparison) *ranke.Comparison {
	out := &ranke.Comparison{Eq: c.Eq, Ne: c.Ne, Lt: c.Lt, Le: c.Le, Gt: c.Gt, Ge: c.Ge}
	if c.In != nil {
		out.In = *c.In
	}
	if c.Glob != nil {
		out.Glob = *c.Glob
	}
	return out
}

// deref reads an optional wire array as a plain slice.
func deref[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
}
