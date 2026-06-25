// package: core / logic
// type:    logic
// job:     the server's logic layer — assemble configured archives into a registry with lifecycle state, reconcile toward each archive's target state, and enforce the two gates (authz + lifecycle) for every access
// limits:  composition root (imports config/access/grants/assembler/ranke); no HTTP (-> REST layer maps errors to 401/403/404/503); no mint-signing yet (the vault→Contributor wiring is a later slice)
//
// Think of it as systemd for archives: archives are the services (each built
// by the assembler from its config), core brings them to their target state at
// boot (Reconcile) and starts/stops them on demand (Control); a service that
// fails to build is Failed, not fatal to the boot.
//
// Lifecycle is a control loop: the TARGET state (running/running-readonly/
// stopped) is persisted as a config field; the CURRENT state lives only here,
// in the in-memory registry, recomputed by reconciliation. Every data/control
// access passes two gates: gate 1 is authorization (access.Require → 403), gate
// 2 is the archive's current lifecycle state (→ operational error). Both must
// pass.
package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	ranke "github.com/flocko-motion/ranke-go"

	"rankedb/access"
	"rankedb/adapter/config"
	"rankedb/adapter/grants"
	"rankedb/assembler"
)

// State is an archive's lifecycle state. The target subset (settable, persisted)
// is Running/Readonly/Stopped; Starting/Failed are transient runtime-only.
type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateReadonly State = "running-readonly"
	StateFailed   State = "failed"
)

// Operational (gate-2) errors, distinct from an *access.Denied (gate-1 / 403).
var (
	ErrNotFound    = errors.New("core: archive not found")
	ErrReadOnly    = errors.New("core: archive is read-only")
	ErrUnavailable = errors.New("core: archive is not running")
	ErrInvalid     = errors.New("core: invalid request") // bad name / missing backend → 400
)

// entry is the in-memory record for one archive: its definition, current state,
// the live handle (nil unless running/readonly), and the last assembly error.
type entry struct {
	def     archiveDef
	current State
	handle  *assembler.Handle
	err     error
}

// Core is the server logic layer. Build with New, then Reconcile to bring
// configured archives up to their target state.
type Core struct {
	authz *access.Authz
	cfg   config.Store
	deps  assembler.Deps

	mu  sync.Mutex
	reg map[string]*entry // keyed by "tenant/ra"
}

// New builds a Core over its injected dependencies (all interfaces, so it is
// unit-testable with mem backends): the authz engine, the config store, and the
// assembler deps (e.g. the internal DB for opt-in "internal" backends).
func New(authz *access.Authz, cfg config.Store, deps assembler.Deps) *Core {
	return &Core{authz: authz, cfg: cfg, deps: deps, reg: map[string]*entry{}}
}

// Authz exposes the authorization engine for the API's capability-management
// endpoints (admit/grant/revoke/disable). Grant administration is an authz
// concern, distinct from the archive lifecycle this core supervises.
func (c *Core) Authz() *access.Authz { return c.authz }

func key(tenant, ra string) string { return tenant + "/" + ra }

// Reconcile (re)loads the config and drives every configured archive toward its
// target state. A per-archive assembly failure marks that archive Failed and is
// recorded — it never aborts the pass. A malformed config (bad names/state) is
// a hard error.
func (c *Core) Reconcile(ctx context.Context) error {
	entries, err := c.cfg.Load(ctx)
	if err != nil {
		return fmt.Errorf("core: load config: %w", err)
	}
	defs, err := loadArchives(entries)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range defs {
		k := key(d.Tenant, d.RA)
		e := c.reg[k]
		if e == nil {
			e = &entry{current: StateStopped}
			c.reg[k] = e
		}
		e.def = d
		c.reconcileEntry(ctx, e)
	}
	return nil
}

// reconcileEntry drives one entry toward e.def.Target. Caller holds c.mu.
func (c *Core) reconcileEntry(ctx context.Context, e *entry) {
	switch e.def.Target {
	case StateStopped:
		if e.handle != nil {
			_ = e.handle.Close()
			e.handle = nil
		}
		e.current, e.err = StateStopped, nil
	case StateRunning, StateReadonly:
		if e.handle == nil {
			e.current = StateStarting
			h, err := assembler.Assemble(ctx, e.def.Tenant, e.def.RA, e.def.Spec, c.deps)
			if err != nil {
				e.current, e.err = StateFailed, err
				return
			}
			e.handle = h
		}
		e.err = nil
		if e.def.Target == StateReadonly {
			e.current = StateReadonly
		} else {
			e.current = StateRunning
		}
	}
}

// Reader enforces both gates for a read/query/verify and returns the live
// archive: gate 1 = ReadRA authorization, gate 2 = serving (running or
// running-readonly).
func (c *Core) Reader(ctx context.Context, subject, tenant, ra string) (ranke.Archive, error) {
	return c.archiveFor(ctx, subject, tenant, ra, access.ReadRA, false)
}

// archiveFor runs gate 1 (authz for action) then gate 2 (lifecycle) and returns
// the live archive. needRunning rejects a read-only archive (writes need it).
func (c *Core) archiveFor(ctx context.Context, subject, tenant, ra string, action access.Action, needRunning bool) (ranke.Archive, error) {
	if err := c.authz.Require(ctx, access.Request{Subject: subject, Action: action, Scope: grants.Archive(tenant, ra)}); err != nil {
		return nil, err // *access.Denied (403) or a store error
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.reg[key(tenant, ra)]
	if !ok {
		return nil, ErrNotFound
	}
	switch e.current {
	case StateRunning:
		// serves reads and writes
	case StateReadonly:
		if needRunning {
			return nil, ErrReadOnly
		}
	default:
		return nil, ErrUnavailable // stopped / starting / failed
	}
	return e.handle.Archive, nil
}

// Control changes an archive's target lifecycle state (gated by ControlRA),
// persists the new target to config, and reconciles the live archive toward it.
// target must be a settable state (running, running-readonly, or stopped).
func (c *Core) Control(ctx context.Context, subject, tenant, ra string, target State) error {
	if err := c.authz.Require(ctx, access.Request{Subject: subject, Action: access.ControlRA, Scope: grants.Archive(tenant, ra)}); err != nil {
		return err
	}
	if target != StateRunning && target != StateReadonly && target != StateStopped {
		return fmt.Errorf("core: %q is not a settable target state", target)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.reg[key(tenant, ra)]
	if !ok {
		return ErrNotFound
	}
	if err := c.persistTarget(ctx, tenant, ra, target); err != nil {
		return err
	}
	e.def.Target = target
	c.reconcileEntry(ctx, e)
	return e.err // surfaces a failed (re)assembly, if any
}

// persistTarget writes the archive's target state to its config field. Caller holds c.mu.
func (c *Core) persistTarget(ctx context.Context, tenant, ra string, target State) error {
	entries, err := c.cfg.Load(ctx)
	if err != nil {
		return fmt.Errorf("core: load config: %w", err)
	}
	entries["tenants."+tenant+".archives."+ra+".state"] = string(target)
	if err := c.cfg.Save(ctx, entries); err != nil {
		return fmt.Errorf("core: save config: %w", err)
	}
	return nil
}

// StateOf returns an archive's current (runtime) state — for health/introspection.
func (c *Core) StateOf(tenant, ra string) (State, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.reg[key(tenant, ra)]
	if !ok {
		return "", false
	}
	return e.current, true
}

// Status is an archive's metadata plus its current and target lifecycle state.
type Status struct {
	Tenant  string
	RA      string
	Title   string
	Current State
	Target  State
}

// Status returns an archive's status if the subject can see its tenant (else
// ErrNotFound — hiding existence). Unlike Reader it works in ANY lifecycle
// state: you can see that an archive is stopped or failed. Gate is visibility,
// not ReadRA — an operator who manages lifecycle but can't read data still sees status.
func (c *Core) Status(ctx context.Context, subject, tenant, ra string) (Status, error) {
	visible, err := c.authz.Visible(ctx, subject, tenant)
	if err != nil {
		return Status{}, err
	}
	if !visible {
		return Status{}, ErrNotFound // hide existence from a subject with no foothold in the tenant
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.reg[key(tenant, ra)]
	if !ok {
		return Status{}, ErrNotFound
	}
	return Status{Tenant: tenant, RA: ra, Title: e.def.Title, Current: e.current, Target: e.def.Target}, nil
}

// CreateArchive defines a new archive and its persistence stack, then brings it
// up — the runtime equivalent of a config entry, so a client (e.g. the test
// suite) chooses the storage/sequencer backend at runtime. Gated by tenant-admin.
// Names must be valid slugs and a sequencer backend is required (B_h is the
// archive's key). Returns the new archive's status — Failed if its backends
// don't assemble.
func (c *Core) CreateArchive(ctx context.Context, actor, tenant, ra, title string, spec assembler.Spec) (Status, error) {
	if err := c.authz.Require(ctx, access.Request{Subject: actor, Action: access.AdminTenant, Scope: grants.Tenant(tenant)}); err != nil {
		return Status{}, err
	}
	if !validName(tenant) || !validName(ra) {
		return Status{}, fmt.Errorf("%w: tenant/archive names must be valid slugs", ErrInvalid)
	}
	if spec.Storage.Backend == "" || spec.Sequencer.Backend == "" {
		return Status{}, fmt.Errorf("%w: storage.backend and sequencer.backend are required", ErrInvalid)
	}

	entries, err := c.cfg.Load(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("core: load config: %w", err)
	}
	p := "tenants." + tenant + ".archives." + ra + "."
	set := func(k, v string) {
		if v != "" {
			entries[p+k] = v
		}
	}
	set("title", title)
	entries[p+"state"] = string(StateRunning)
	set("storage.backend", spec.Storage.Backend)
	set("sequencer.backend", spec.Sequencer.Backend)
	set("sequencer.dsn", spec.Sequencer.DSN)
	set("sequencer.key", spec.Sequencer.Key)
	if err := c.cfg.Save(ctx, entries); err != nil {
		return Status{}, fmt.Errorf("core: save config: %w", err)
	}
	if err := c.Reconcile(ctx); err != nil { // picks up + assembles the new archive
		return Status{}, err
	}
	return c.Status(ctx, actor, tenant, ra)
}

// DeleteArchive stops an archive and removes its definition. Gated by tenant-admin.
func (c *Core) DeleteArchive(ctx context.Context, actor, tenant, ra string) error {
	if err := c.authz.Require(ctx, access.Request{Subject: actor, Action: access.AdminTenant, Scope: grants.Tenant(tenant)}); err != nil {
		return err
	}
	k := key(tenant, ra)
	c.mu.Lock()
	e, ok := c.reg[k]
	if ok {
		if e.handle != nil {
			_ = e.handle.Close()
		}
		delete(c.reg, k)
	}
	c.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	entries, err := c.cfg.Load(ctx)
	if err != nil {
		return fmt.Errorf("core: load config: %w", err)
	}
	prefix := "tenants." + tenant + ".archives." + ra + "."
	for ck := range entries {
		if strings.HasPrefix(ck, prefix) {
			delete(entries, ck)
		}
	}
	return c.cfg.Save(ctx, entries)
}
