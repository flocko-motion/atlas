// package: api / archives
// type:    adapter
// job:     REST endpoints for archive resources — status now; reads/lifecycle/gql to follow
// limits:  thin adapters over core (-> core); no logic here; registered via schemaf codegen
package api

import (
	"context"
	"fmt"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"rankedb/core"
)

// GetArchiveEndpoint returns an archive's status: its title and current + target
// lifecycle state. Works in any state (a stopped/failed archive still reports);
// hidden (404) from a subject with no visibility into the tenant.
type GetArchiveEndpoint struct{}

// Method is GET.
func (GetArchiveEndpoint) Method() string { return "GET" }

// Path addresses the archive by tenant and ra.
func (GetArchiveEndpoint) Path() string { return "/api/archives/{tenant}/{ra}" }

// Auth requires a valid JWT.
func (GetArchiveEndpoint) Auth() bool { return true }

// Handle returns the archive's status, or 404 if the caller can't see the tenant.
func (GetArchiveEndpoint) Handle(ctx context.Context, req GetArchiveReq) (GetArchiveResp, error) {
	subject, _ := schemafapi.Subject(ctx)
	st, err := svc.Status(ctx, subject, req.Tenant, req.RA)
	if err != nil {
		return GetArchiveResp{}, mapErr(err)
	}
	return GetArchiveResp{
		Tenant:  st.Tenant,
		RA:      st.RA,
		Title:   st.Title,
		Current: string(st.Current),
		Target:  string(st.Target),
	}, nil
}

// GetArchiveReq addresses an archive by its tenant and ra path segments.
type GetArchiveReq struct {
	Tenant string `path:"tenant"`
	RA     string `path:"ra"`
}

// GetArchiveResp is an archive's reported status.
type GetArchiveResp struct {
	Tenant  string `json:"tenant"`
	RA      string `json:"ra"`
	Title   string `json:"title"`
	Current string `json:"current"` // runtime: stopped/starting/running/running-readonly/failed
	Target  string `json:"target"`  // desired: running/running-readonly/stopped
}

var _ schemafapi.Endpoint[GetArchiveReq, GetArchiveResp] = GetArchiveEndpoint{}

// PatchArchiveEndpoint sets an archive's target lifecycle state (start / stop /
// set read-only) — declarative: it writes the target and core reconciles toward
// it. Gated by ra.control. Returns the archive's updated status.
type PatchArchiveEndpoint struct{}

// Method is PATCH.
func (PatchArchiveEndpoint) Method() string { return "PATCH" }

// Path addresses the archive by tenant and ra.
func (PatchArchiveEndpoint) Path() string { return "/api/archives/{tenant}/{ra}" }

// Auth requires a valid JWT.
func (PatchArchiveEndpoint) Auth() bool { return true }

// Handle sets the target lifecycle state and returns the archive's updated status.
func (PatchArchiveEndpoint) Handle(ctx context.Context, req PatchArchiveReq) (GetArchiveResp, error) {
	subject, _ := schemafapi.Subject(ctx)
	target := core.State(req.Target)
	switch target {
	case core.StateRunning, core.StateReadonly, core.StateStopped:
	default:
		return GetArchiveResp{}, fmt.Errorf("target %q must be running|running-readonly|stopped: %w", req.Target, schemafapi.ErrBadRequest)
	}
	if err := svc.Control(ctx, subject, req.Tenant, req.RA, target); err != nil {
		return GetArchiveResp{}, mapErr(err)
	}
	st, err := svc.Status(ctx, subject, req.Tenant, req.RA)
	if err != nil {
		return GetArchiveResp{}, mapErr(err)
	}
	return GetArchiveResp{
		Tenant:  st.Tenant,
		RA:      st.RA,
		Title:   st.Title,
		Current: string(st.Current),
		Target:  string(st.Target),
	}, nil
}

// PatchArchiveReq addresses an archive (path) and carries the desired target state (body).
type PatchArchiveReq struct {
	Tenant string `path:"tenant"`
	RA     string `path:"ra"`
	Target string `json:"target"` // running | running-readonly | stopped
}

var _ schemafapi.Endpoint[PatchArchiveReq, GetArchiveResp] = PatchArchiveEndpoint{}
