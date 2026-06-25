// package: api / archives
// type:    adapter
// job:     REST endpoints for archive resources — status now; reads/lifecycle/gql to follow
// limits:  thin adapters over core (-> core); no logic here; registered via schemaf codegen
package api

import (
	"context"

	schemafapi "github.com/flocko-motion/schemaf/api"
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
