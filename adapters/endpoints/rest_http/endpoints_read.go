// package: rest_http / transport
// type:    logic
// job:     the read endpoints — POST /query and the cacheable by-id GET reads
// limits:  transport only; the read language is ranke-go's and the reads core's (-> internal/core)
package rest_http

import (
	"io"
	"net/http"

	ranke "github.com/rankegraph/ranke-go"

	"github.com/flocko-motion/rankedb/internal/core"
)

// Query serves POST /query: it runs the declarative query and streams the results.
func (s *Server) Query(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, core.CatInvalid, "malformed query")
		return
	}
	// The library owns the read language, request and results alike, so the wire form
	// decodes there and this endpoint carries bytes.
	rq, err := ranke.DecodeQuery(body)
	if err != nil {
		writeError(w, core.CatInvalid, err.Error())
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

// GetBranchInfo serves GET /branches/{branch}/info.
func (s *Server) GetBranchInfo(w http.ResponseWriter, r *http.Request, branch string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpBranchInfo, Branch: branch}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respondRevalidated(w, r, stream)
}

// GetArchiveInfo serves GET /archive/info — the branch-table head and its shape, read in
// the $archive scope, which is the grant it needs.
func (s *Server) GetArchiveInfo(w http.ResponseWriter, r *http.Request) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpArchiveInfo, Branch: core.Archive}
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
