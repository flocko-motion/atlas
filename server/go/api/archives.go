// package: api / archives
// type:    adapter
// job:     REST endpoints for archive resources — status now; reads/lifecycle/gql to follow
// limits:  thin adapters over core (-> core); no logic here; registered via schemaf codegen
package api

import (
	"context"
	"fmt"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"rankedb/assembler"
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

// StackStorage is the storage (𝒰) backend choice for a new archive.
type StackStorage struct {
	Backend string `json:"backend"` // mem | fs | sqlite (s3 later)
	Dir     string `json:"dir,omitempty"`
	DSN     string `json:"dsn,omitempty"`
}

// StackSequencer is the sequencer (B_h) backend choice for a new archive.
type StackSequencer struct {
	Backend string `json:"backend"` // mem | file | postgres | internal
	Path    string `json:"path,omitempty"`
	DSN     string `json:"dsn,omitempty"`
	Key     string `json:"key,omitempty"`
}

// CreateArchiveEndpoint defines a new archive and its persistence stack, then
// brings it up. The stack (storage + sequencer backends) is chosen at runtime by
// the caller — this is what lets a test suite drive any backend. Gated by tenant-admin.
type CreateArchiveEndpoint struct{}

// Method is POST.
func (CreateArchiveEndpoint) Method() string { return "POST" }

// Path is the tenant's archives collection.
func (CreateArchiveEndpoint) Path() string { return "/api/tenants/{tenant}/archives" }

// Auth requires a valid JWT.
func (CreateArchiveEndpoint) Auth() bool { return true }

// Handle creates the archive from its requested stack and returns its status.
func (CreateArchiveEndpoint) Handle(ctx context.Context, req CreateArchiveReq) (GetArchiveResp, error) {
	actor, _ := schemafapi.Subject(ctx)
	spec := assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: req.Storage.Backend, Dir: req.Storage.Dir, DSN: req.Storage.DSN},
		Sequencer: assembler.SequencerSpec{Backend: req.Sequencer.Backend, Path: req.Sequencer.Path, DSN: req.Sequencer.DSN, Key: req.Sequencer.Key},
	}
	st, err := svc.CreateArchive(ctx, actor, req.Tenant, req.RA, req.Title, spec)
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

// CreateArchiveReq names the tenant (path) and defines the archive (body).
type CreateArchiveReq struct {
	Tenant    string         `path:"tenant"`
	RA        string         `json:"ra"`
	Title     string         `json:"title,omitempty"`
	Storage   StackStorage   `json:"storage"`
	Sequencer StackSequencer `json:"sequencer"`
}

var _ schemafapi.Endpoint[CreateArchiveReq, GetArchiveResp] = CreateArchiveEndpoint{}

// DeleteArchiveEndpoint stops an archive and removes its definition. Gated by tenant-admin.
type DeleteArchiveEndpoint struct{}

// Method is DELETE.
func (DeleteArchiveEndpoint) Method() string { return "DELETE" }

// Path addresses an archive in the tenant's archives collection.
func (DeleteArchiveEndpoint) Path() string { return "/api/tenants/{tenant}/archives/{ra}" }

// Auth requires a valid JWT.
func (DeleteArchiveEndpoint) Auth() bool { return true }

// Handle deletes the archive.
func (DeleteArchiveEndpoint) Handle(ctx context.Context, req DeleteArchiveReq) (DeleteArchiveResp, error) {
	actor, _ := schemafapi.Subject(ctx)
	if err := svc.DeleteArchive(ctx, actor, req.Tenant, req.RA); err != nil {
		return DeleteArchiveResp{}, mapErr(err)
	}
	return DeleteArchiveResp{Deleted: true}, nil
}

// DeleteArchiveReq addresses an archive by tenant and ra.
type DeleteArchiveReq struct {
	Tenant string `path:"tenant"`
	RA     string `path:"ra"`
}

// DeleteArchiveResp confirms deletion.
type DeleteArchiveResp struct {
	Deleted bool `json:"deleted"`
}

var _ schemafapi.Endpoint[DeleteArchiveReq, DeleteArchiveResp] = DeleteArchiveEndpoint{}
