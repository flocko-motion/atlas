// package: rest_http / transport
// type:    logic
// job:     the read endpoints — POST /query and the cacheable by-id GET reads — with wire→RQL query mapping
// limits:  translation only; the query language is ranke-go's, the reads run behind core.Handle and render inside the Stream (-> internal/core)
package rest_http

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
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
		writeError(w, core.CatInvalid, err.Error())
		return
	}
	// A branch-rooted read authorizes R on that branch; a read with no branch is
	// the privileged $universe read.
	branch := rq.Select.Branch
	if branch == "" {
		branch = core.Universe
	}
	req := &core.Request{
		Credential: credentialOf(r.Context()),
		Op:         core.OpClaimQuery,
		Branch:     branch,
		Query:      &rq,
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

// rankeQuery maps the wire query onto ranke-go's RQL, which the engine executes
// as-is. Only the translation is ours: the wire's own concerns — its narrower
// `detail`, its `bool|int|string` content cap, its sequence framing — resolve to
// the library's vocabulary here, and nothing of the query language is restated.
func rankeQuery(q openapi.Query) (ranke.Query, error) {
	var rq ranke.Query

	if q.Select.Branch != nil {
		rq.Select.Branch = *q.Select.Branch
	}
	if q.Select.Claim != nil {
		id, err := ranke.ParseId(*q.Select.Claim)
		if err != nil {
			return rq, fmt.Errorf("select.claim: invalid id")
		}
		rq.Select.Claim = id
		// A claim with no branch is the privileged unconfined read; the library
		// spells that scope as a reserved branch name.
		if rq.Select.Branch == "" {
			rq.Select.Branch = ranke.BranchUniverse
		}
	}
	for _, p := range deref(q.Select.Path) {
		step := ranke.PathStep{Edges: p.Edges, Nodes: deref(p.Nodes)}
		if p.Dir != nil {
			step.Dir = ranke.Direction(*p.Dir)
		}
		if p.Depth != nil {
			step.Max = *p.Depth // the wire's depth is a max-hop bound
		}
		rq.Select.Path = append(rq.Select.Path, step)
	}

	if q.Output != nil {
		if err := outputInto(&rq.Output, *q.Output); err != nil {
			return rq, err
		}
	}

	if q.Order != nil {
		key := ranke.OrderKey{Field: q.Order.Field}
		if q.Order.Dir != nil {
			key.Dir = ranke.SortDir(*q.Order.Dir)
		}
		rq.Order = []ranke.OrderKey{key}
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
		// The wire asks for a report or not; the library grades verbosity.
		if q.Execution.Report != nil && *q.Execution.Report {
			rq.Execution.Report = ranke.ReportInfo
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

// outputInto resolves the wire's output shaping. The wire's `detail` folds two of
// the library's orthogonal axes into one enum — "path" asks for the route, which
// is a Shape, not a Detail — so it unfolds here.
func outputInto(out *ranke.Output, o openapi.Output) error {
	if o.Detail != nil {
		switch *o.Detail {
		case "id":
			out.Detail = ranke.DetailID
		case "claim":
			out.Detail = ranke.DetailClaims
		case "path":
			out.Shape, out.Detail = ranke.ShapePath, ranke.DetailClaims
		default:
			return fmt.Errorf("output.detail: unknown %q", *o.Detail)
		}
	}
	if o.Encoding != nil {
		// The wire names a sequence framing; the library names each claim's form.
		switch *o.Encoding {
		case "json-seq":
			out.Encoding = ranke.ResultJSON
		case "cbor-seq":
			out.Encoding = ranke.ResultCBOR
		default:
			return fmt.Errorf("output.encoding: unknown %q", *o.Encoding)
		}
	}
	cap, err := contentCap(o.Content)
	if err != nil {
		return err
	}
	if cap > 0 {
		out.Content = &ranke.Content{Max: cap}
		if o.Overflow != nil {
			out.Content.Overflow = ranke.Overflow(*o.Overflow)
		}
	}
	return nil
}

// contentCap resolves the content cap, a bool|int|string union: false (or absent)
// carries no content, true means uncapped-per-claim, a number is bytes, and a
// string is a human size ("4kb").
func contentCap(raw *openapi.Output_Content) (int64, error) {
	if raw == nil {
		return 0, nil
	}
	if b, err := raw.AsOutputContent0(); err == nil {
		if !b {
			return 0, nil
		}
		return math.MaxInt64, nil
	}
	if n, err := raw.AsOutputContent1(); err == nil {
		return int64(n), nil
	}
	s, err := raw.AsOutputContent2()
	if err != nil {
		return 0, fmt.Errorf("output.content: want a boolean, a byte count, or a size like \"4kb\"")
	}
	size, err := parseSize(s)
	if err != nil {
		return 0, fmt.Errorf("output.content: %w", err)
	}
	return size, nil
}

// whereOf maps the wire's boolean tree onto the library's. The wire models the
// tree as a oneOf — and/or/not subtrees, or a field→comparison map — so the
// variant is found by which key is present.
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
	fields, err := w.AsWhere3()
	if err != nil {
		return nil, fmt.Errorf("where: want and/or/not or a field→comparison map")
	}
	// A field map is a leaf per field; several fields conjoin, since each field
	// carries exactly one comparison. Sorted so the tree is deterministic.
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	leaves := make([]ranke.Where, 0, len(names))
	for _, name := range names {
		leaves = append(leaves, ranke.Where{Field: name, Test: comparisonOf(fields[name])})
	}
	switch len(leaves) {
	case 0:
		return nil, fmt.Errorf("where: empty")
	case 1:
		return &leaves[0], nil
	default:
		return &ranke.Where{And: leaves}, nil
	}
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

// parseSize parses a human-readable byte size: a bare number is bytes, or a
// number with a kb/mb/gb/tb suffix (binary multiples, case-insensitive, optional
// trailing "b").
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, nil
	}
	mult := int64(1)
	for _, u := range []struct {
		suffix string
		mult   int64
	}{{"tb", 1 << 40}, {"gb", 1 << 30}, {"mb", 1 << 20}, {"kb", 1 << 10}} {
		if strings.HasSuffix(s, u.suffix) {
			mult, s = u.mult, strings.TrimSpace(strings.TrimSuffix(s, u.suffix))
			break
		}
	}
	if mult == 1 {
		s = strings.TrimSpace(strings.TrimSuffix(s, "b")) // bare bytes, e.g. "4096b"
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return n * mult, nil
}

// deref reads an optional wire array as a plain slice.
func deref[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
}
