// package: api / reads
// type:    adapter
// job:     read endpoints over a running archive — list/get branches, verify a branch, get a claim (JSON projection + canonical bytes)
// limits:  thin over core.Reader (ReadRA + serving gates); content blobs (/contents) deferred — they live on the Universe, not the Archive
package api

import (
	"context"
	"encoding/base64"

	ranke "github.com/flocko-motion/ranke-go"
	schemafapi "github.com/flocko-motion/schemaf/api"
)

// idStr renders an Id, tolerating nil (an unset head/ref → "").
func idStr(id ranke.Id) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// dirStr renders a relation edge's direction (from/to), "" for non-relations.
func dirStr(d ranke.RelationDirection) string {
	switch d {
	case ranke.RelationFrom:
		return "from"
	case ranke.RelationTo:
		return "to"
	default:
		return ""
	}
}

// BranchSummary is a branch and its current head id.
type BranchSummary struct {
	Name string `json:"name"`
	Head string `json:"head"`
}

// ListBranchesEndpoint lists an archive's branches with their head ids.
type ListBranchesEndpoint struct{}

// Method is GET.
func (ListBranchesEndpoint) Method() string { return "GET" }

// Path is the archive's branches collection.
func (ListBranchesEndpoint) Path() string { return "/api/archives/{tenant}/{ra}/branches" }

// Auth requires a valid JWT.
func (ListBranchesEndpoint) Auth() bool { return true }

// Handle returns the archive's branches (read-gated; 503 if not serving).
func (ListBranchesEndpoint) Handle(ctx context.Context, req ArchiveReadReq) (ListBranchesResp, error) {
	subject, _ := schemafapi.Subject(ctx)
	arc, err := svc.Reader(ctx, subject, req.Tenant, req.RA)
	if err != nil {
		return ListBranchesResp{}, mapErr(err)
	}
	out := []BranchSummary{}
	for _, b := range arc.Branches(ctx) {
		out = append(out, BranchSummary{Name: b.Name(), Head: idStr(b.Latest().Head())})
	}
	return ListBranchesResp{Branches: out}, nil
}

// ArchiveReadReq addresses an archive by tenant and ra.
type ArchiveReadReq struct {
	Tenant string `path:"tenant"`
	RA     string `path:"ra"`
}

// ListBranchesResp is an archive's branches.
type ListBranchesResp struct {
	Branches []BranchSummary `json:"branches"`
}

var _ schemafapi.Endpoint[ArchiveReadReq, ListBranchesResp] = ListBranchesEndpoint{}

// GetBranchEndpoint returns one branch: its head, binding time, contributor, and history depth.
type GetBranchEndpoint struct{}

// Method is GET.
func (GetBranchEndpoint) Method() string { return "GET" }

// Path addresses one branch by name.
func (GetBranchEndpoint) Path() string { return "/api/archives/{tenant}/{ra}/branches/{name}" }

// Auth requires a valid JWT.
func (GetBranchEndpoint) Auth() bool { return true }

// Handle returns the named branch, or 404 if it doesn't exist.
func (GetBranchEndpoint) Handle(ctx context.Context, req BranchReq) (BranchResp, error) {
	subject, _ := schemafapi.Subject(ctx)
	arc, err := svc.Reader(ctx, subject, req.Tenant, req.RA)
	if err != nil {
		return BranchResp{}, mapErr(err)
	}
	if !arc.HasBranch(ctx, req.Name) {
		return BranchResp{}, schemafapi.ErrNotFound
	}
	b, err := arc.GetBranch(ctx, req.Name)
	if err != nil {
		return BranchResp{}, mapErr(err)
	}
	e := b.Latest()
	return BranchResp{
		Name:        b.Name(),
		Head:        idStr(e.Head()),
		Time:        e.Time().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Contributor: idStr(e.Contributor().ID()),
		History:     len(b.Provenance()),
	}, nil
}

// BranchReq addresses a branch by tenant, ra, and name.
type BranchReq struct {
	Tenant string `path:"tenant"`
	RA     string `path:"ra"`
	Name   string `path:"name"`
}

// BranchResp is one branch's current binding.
type BranchResp struct {
	Name        string `json:"name"`
	Head        string `json:"head"`
	Time        string `json:"time"`
	Contributor string `json:"contributor"`
	History     int    `json:"history"` // number of prior bindings
}

var _ schemafapi.Endpoint[BranchReq, BranchResp] = GetBranchEndpoint{}

// VerifyBranchEndpoint runs the §5.10 verification across a branch's provenance.
// Verification failure is a result (200, valid=false), not an HTTP error; a
// missing branch is 404.
type VerifyBranchEndpoint struct{}

// Method is GET.
func (VerifyBranchEndpoint) Method() string { return "GET" }

// Path is a branch's verification sub-resource.
func (VerifyBranchEndpoint) Path() string {
	return "/api/archives/{tenant}/{ra}/branches/{name}/verification"
}

// Auth requires a valid JWT.
func (VerifyBranchEndpoint) Auth() bool { return true }

// Handle verifies the branch and reports the result.
func (VerifyBranchEndpoint) Handle(ctx context.Context, req BranchReq) (VerificationResp, error) {
	subject, _ := schemafapi.Subject(ctx)
	arc, err := svc.Reader(ctx, subject, req.Tenant, req.RA)
	if err != nil {
		return VerificationResp{}, mapErr(err)
	}
	if !arc.HasBranch(ctx, req.Name) {
		return VerificationResp{}, schemafapi.ErrNotFound
	}
	if err := arc.VerifyBranch(ctx, req.Name); err != nil {
		return VerificationResp{Valid: false, Error: err.Error()}, nil
	}
	return VerificationResp{Valid: true}, nil
}

// VerificationResp is a branch verification result.
type VerificationResp struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

var _ schemafapi.Endpoint[BranchReq, VerificationResp] = VerifyBranchEndpoint{}

// EdgeView is one edge in a claim's projection.
type EdgeView struct {
	Reference string `json:"reference"`
	Type      string `json:"type"`
	Direction string `json:"direction,omitempty"` // from/to on relation edges
}

// ClaimView is a claim's readable projection plus its canonical signed bytes.
type ClaimView struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Contributor string     `json:"contributor"`
	Encoding    string     `json:"encoding,omitempty"`
	CreatedAt   string     `json:"created_at"`
	ContentHash string     `json:"content_hash,omitempty"`
	Size        uint64     `json:"size,omitempty"`
	Edges       []EdgeView `json:"edges"`
	Canonical   string     `json:"canonical"` // base64 of the canonical CBOR — independently verifiable
}

// GetClaimEndpoint returns a claim: a readable projection plus its canonical bytes.
type GetClaimEndpoint struct{}

// Method is GET.
func (GetClaimEndpoint) Method() string { return "GET" }

// Path addresses one claim by id.
func (GetClaimEndpoint) Path() string { return "/api/archives/{tenant}/{ra}/claims/{id}" }

// Auth requires a valid JWT.
func (GetClaimEndpoint) Auth() bool { return true }

// Handle returns the claim at id (404 if absent, 400 if the id is malformed).
func (GetClaimEndpoint) Handle(ctx context.Context, req ClaimReq) (ClaimView, error) {
	subject, _ := schemafapi.Subject(ctx)
	arc, err := svc.Reader(ctx, subject, req.Tenant, req.RA)
	if err != nil {
		return ClaimView{}, mapErr(err)
	}
	id, err := ranke.ParseId(req.ID)
	if err != nil {
		return ClaimView{}, schemafapi.ErrBadRequest
	}
	if !arc.HasClaim(ctx, id) {
		return ClaimView{}, schemafapi.ErrNotFound
	}
	c, err := arc.GetClaim(ctx, id)
	if err != nil {
		return ClaimView{}, mapErr(err)
	}
	return projectClaim(c), nil
}

// projectClaim builds the JSON view + canonical bytes for a claim.
func projectClaim(c ranke.Claim) ClaimView {
	n := c.Node()
	edges := []EdgeView{}
	for _, e := range c.Edges() {
		edges = append(edges, EdgeView{Reference: idStr(e.Reference()), Type: e.Type(), Direction: dirStr(e.RelationDirection())})
	}
	canonical := ""
	if b, err := c.Encode(); err == nil {
		canonical = base64.StdEncoding.EncodeToString(b)
	}
	return ClaimView{
		ID:          idStr(c.ID()),
		Type:        n.Type(),
		Contributor: idStr(c.Contributor().ID()),
		Encoding:    n.Encoding(),
		CreatedAt:   n.CreatedAt().UTC().Format("2006-01-02T15:04:05Z07:00"),
		ContentHash: idStr(n.ContentHash()),
		Size:        n.Size(),
		Edges:       edges,
		Canonical:   canonical,
	}
}

// ClaimReq addresses a claim by tenant, ra, and id.
type ClaimReq struct {
	Tenant string `path:"tenant"`
	RA     string `path:"ra"`
	ID     string `path:"id"`
}

var _ schemafapi.Endpoint[ClaimReq, ClaimView] = GetClaimEndpoint{}
