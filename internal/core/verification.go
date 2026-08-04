// package: core / orchestration
// type:    registry
// job:     give verification runs the identity, history and lifecycle the library's live handle has none of
// limits:  operational state only; the walk itself is ranke-go's (-> github.com/flocko-motion/ranke-go)
//
// The library hands back a live in-process handle — Verified, Failures, Done, Err, Wait —
// with no id, no persistence and no cancellation beyond the context it was started
// with. What an operator asks for is the other half: start a run and get an id back,
// list what is running, poll one by id after it has finished, stop one and keep what it
// found, and be refused when too many are already going. None of that is graph
// behaviour, so it lives here.
//
// A run's report is retained after the walk ends, because a report is a point-in-time
// record: a layer repaired externally shows clean in a later run, and the earlier
// finding is exactly what an operator needs to still be able to read.
package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	ranke "github.com/flocko-motion/ranke-go"
)

// defaultMaxActiveRuns bounds concurrent runs when the configuration names no limit.
// Verification is resource-heavy — a full-content pass re-reads every blob — so the
// default is one, and the server never stops a run to make room for another.
const defaultMaxActiveRuns = 1

// registry holds the verification runs this server knows about.
type registry struct {
	mu     sync.Mutex
	runs   map[string]*run
	order  []string // ids in start order, so listing is newest-first without sorting times
	max    int
	nextID int
	now    func() time.Time
}

// run is one verification run: the report an operator reads, plus what it takes to stop
// the walk producing it.
type run struct {
	report VerificationReport
	cancel context.CancelFunc
}

// newRegistry builds an empty registry bounded to max active runs.
func newRegistry(max int, now func() time.Time) *registry {
	if max <= 0 {
		max = defaultMaxActiveRuns
	}
	if now == nil {
		now = time.Now
	}
	return &registry{runs: make(map[string]*run), max: max, now: now}
}

// start launches a run over the archive and registers it. The walk runs detached from
// the request that asked for it — a pass over a large closure outlives any client — so
// it takes its own context, cancelled by stop.
func (r *registry) start(ctx context.Context, archive ranke.Archive, cfg VerificationConfig) (VerificationReport, error) {
	if archive == nil {
		return VerificationReport{}, fmt.Errorf("%w: no archive to verify", ErrNotImplemented)
	}

	r.mu.Lock()
	if r.active() >= r.max {
		r.mu.Unlock()
		return VerificationReport{}, fmt.Errorf("%w: %d run(s) already active", ErrBusy, r.max)
	}
	r.nextID++
	id := fmt.Sprintf("ver-%d", r.nextID)
	r.mu.Unlock()

	// Detached from the caller's context: cancellation is an operator action through
	// stop, not a side effect of the client that started it hanging up.
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	handle, err := archive.Verify(runCtx, verifyOptions(cfg)...)
	if err != nil {
		cancel()
		return VerificationReport{}, mapLibError(err)
	}

	report := VerificationReport{
		ID:        id,
		Config:    cfg,
		Head:      archive.Head().String(),
		Status:    RunRunning,
		StartedAt: r.now(),
	}
	r.mu.Lock()
	r.runs[id] = &run{report: report, cancel: cancel}
	r.order = append(r.order, id)
	r.mu.Unlock()

	go r.collect(id, handle)
	return report, nil
}

// collect waits for the walk and folds its outcome into the retained report.
func (r *registry) collect(id string, handle ranke.VerificationRun) {
	handle.Wait()

	r.mu.Lock()
	defer r.mu.Unlock()
	held, ok := r.runs[id]
	if !ok {
		return // deleted while running; nothing to record
	}
	r.absorb(&held.report, handle)
	held.report.CompletedAt = r.now()
	if held.report.Status == RunRunning {
		// A cancelled run was already marked stopped; anything still running here
		// finished on its own, cleanly or with a terminal error.
		held.report.Status = RunComplete
		if handle.Err() != nil {
			held.report.Status = RunError
		}
	}
}

// absorb copies the live handle's counters and findings onto the report, so what an
// operator polls advances while the walk runs and survives it afterwards.
func (r *registry) absorb(report *VerificationReport, handle ranke.VerificationRun) {
	report.ClaimsChecked = int64(handle.Verified())
	failures := handle.Failures()
	report.Failures = make([]VerificationFailure, 0, len(failures))
	for _, f := range failures {
		report.Failures = append(report.Failures, VerificationFailure{
			ID:     idString(f.ID),
			Mode:   FailureInvalidContent,
			Layer:  report.Config.Layer,
			Detail: failureDetail(f),
		})
	}
	report.OK = len(report.Failures) == 0
}

// get returns one run's report.
func (r *registry) get(id string) (VerificationReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	held, ok := r.runs[id]
	if !ok {
		return VerificationReport{}, fmt.Errorf("%w: verification run %q", ErrNotFound, id)
	}
	return held.report, nil
}

// list returns every retained report, newest first.
func (r *registry) list() []VerificationReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	reports := make([]VerificationReport, 0, len(r.runs))
	for i := len(r.order) - 1; i >= 0; i-- {
		if held, ok := r.runs[r.order[i]]; ok {
			reports = append(reports, held.report)
		}
	}
	return reports
}

// stop cancels a running walk and keeps its report, partial findings and all. It is
// idempotent: a run that already finished comes back unchanged.
func (r *registry) stop(id string) (VerificationReport, error) {
	r.mu.Lock()
	held, ok := r.runs[id]
	if !ok {
		r.mu.Unlock()
		return VerificationReport{}, fmt.Errorf("%w: verification run %q", ErrNotFound, id)
	}
	if held.report.Status != RunRunning {
		report := held.report
		r.mu.Unlock()
		return report, nil
	}
	held.report.Status = RunStopped
	cancel := held.cancel
	r.mu.Unlock()

	cancel()
	return r.get(id)
}

// remove stops a run if it is still going and drops its record entirely.
func (r *registry) remove(id string) error {
	r.mu.Lock()
	held, ok := r.runs[id]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("%w: verification run %q", ErrNotFound, id)
	}
	delete(r.runs, id)
	r.order = removeID(r.order, id)
	cancel := held.cancel
	r.mu.Unlock()

	cancel()
	return nil
}

// active counts the runs still walking. Called with the lock held.
func (r *registry) active() int {
	var n int
	for _, held := range r.runs {
		if held.report.Status == RunRunning {
			n++
		}
	}
	return n
}

// verifyOptions maps the run's configuration onto the library's verify options. Depth is
// the one axis that changes what the walk reads: full-content re-reads external blobs,
// where the shallower depths check what storage already hands over.
func verifyOptions(cfg VerificationConfig) []ranke.VerifyOption {
	var opts []ranke.VerifyOption
	if cfg.Depth == DepthFullContent {
		opts = append(opts, ranke.WithExternalContent())
	}
	return opts
}

// failureDetail renders one library failure for the report.
func failureDetail(f ranke.Failure) string {
	if f.Err == nil {
		return ""
	}
	return f.Err.Error()
}

// removeID drops one id from the order slice, preserving the rest.
func removeID(ids []string, id string) []string {
	out := ids[:0]
	for _, held := range ids {
		if held != id {
			out = append(out, held)
		}
	}
	return out
}

// --- the dispatch arms ----------------------------------------------------

// verification serves the run API: start, list, get, cancel, delete.
func (c *Core) verification(ctx context.Context, req *Request, archive ranke.Archive) (Stream, error) {
	switch req.Op {
	case OpVerificationStart:
		if req.VerConfig == nil {
			return nil, fmt.Errorf("%w: no verification config", ErrNotFound)
		}
		report, err := c.runs.start(ctx, archive, *req.VerConfig)
		if err != nil {
			return nil, err
		}
		req.Report.step("verification run %s started", report.ID)
		return &jsonStream{value: report}, nil

	case OpVerificationList:
		reports := c.runs.list()
		req.Report.step("%d verification run(s) listed", len(reports))
		return &jsonStream{value: verificationList{Reports: reports}}, nil

	case OpVerificationGet:
		report, err := c.runs.get(req.VerID)
		if err != nil {
			return nil, err
		}
		req.Report.step("verification run %s read", report.ID)
		return &jsonStream{value: report}, nil

	case OpVerificationCancel:
		report, err := c.runs.stop(req.VerID)
		if err != nil {
			return nil, err
		}
		req.Report.step("verification run %s stopped", report.ID)
		return &jsonStream{value: report}, nil

	case OpVerificationDelete:
		if err := c.runs.remove(req.VerID); err != nil {
			return nil, err
		}
		req.Report.step("verification run %s deleted", req.VerID)
		return &jsonStream{value: struct{}{}}, nil
	}
	return nil, ErrNotImplemented
}

// verificationList is the retained reports, newest first.
type verificationList struct {
	Reports []VerificationReport `json:"reports"`
}

// idString renders an id for a report, tolerating the nil a failure may carry.
func idString(id ranke.Id) string {
	if id == nil {
		return ""
	}
	return id.String()
}
