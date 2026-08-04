package core

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	ranke "github.com/flocko-motion/ranke-go"
)

// TestVerificationRunLifecycle drives a run through the registry against the real
// library: it is startable, gets an id, and its report is still retrievable by that id
// once the walk has finished — which is the whole reason the registry exists, the
// library's handle having no identity and no life after the walk.
func TestVerificationRunLifecycle(t *testing.T) {
	c := newStack(t)

	body, mediaType := serve(t, c, &Request{
		Op:        OpVerificationStart,
		VerConfig: &VerificationConfig{Closure: "$archive", Depth: DepthCompleteness},
	})
	if mediaType != mediaJSON {
		t.Fatalf("content type = %q, want %q", mediaType, mediaJSON)
	}
	var started VerificationReport
	if err := json.Unmarshal(body, &started); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, body)
	}
	if started.ID == "" {
		t.Fatal("run has no id")
	}
	if started.Head == "" {
		t.Fatal("run pinned no head")
	}

	// The walk is detached, so give the registry a moment to record its outcome.
	settled := waitForRun(t, c, started.ID)

	report, err := c.runs.get(started.ID)
	if err != nil {
		t.Fatalf("the report did not survive the run: %v", err)
	}
	if !settled {
		t.Fatal("the walk did not finish; the registry never recorded an outcome")
	}
	if report.Status != RunComplete {
		t.Fatalf("status = %q, want %q (err %v)", report.Status, RunComplete, report.Failures)
	}
	if report.CompletedAt.IsZero() {
		t.Fatal("a finished run has no completion time")
	}

	// And it lists.
	listed := c.runs.list()
	if len(listed) != 1 || listed[0].ID != started.ID {
		t.Fatalf("list = %+v, want the one run", listed)
	}
}

// TestVerificationCancelKeepsFindings pins that stopping a run keeps its record: the
// report stays, marked stopped, with whatever it had gathered.
func TestVerificationCancelKeepsFindings(t *testing.T) {
	c := newStack(t)

	started, err := c.runs.start(context.Background(), mustArchive(t, c), VerificationConfig{Closure: "$archive"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	stopped, err := c.runs.stop(started.ID)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if stopped.Status != RunStopped && stopped.Status != RunComplete {
		t.Fatalf("status = %q, want stopped (or complete, if it had already finished)", stopped.Status)
	}

	// Retained either way: a report is a point-in-time record.
	if _, err := c.runs.get(started.ID); err != nil {
		t.Fatalf("the report did not survive cancellation: %v", err)
	}

	// Stopping twice is idempotent rather than an error.
	if _, err := c.runs.stop(started.ID); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

// TestVerificationRunLimit pins that the active-run cap refuses rather than queues, and
// reports as busy so an endpoint answers 429.
func TestVerificationRunLimit(t *testing.T) {
	c := newStack(t)
	WithMaxVerificationRuns(1)(c)
	archive := mustArchive(t, c)

	first, err := c.runs.start(context.Background(), archive, VerificationConfig{Closure: "$archive"})
	if err != nil {
		t.Fatalf("first start: %v", err)
	}

	// Only meaningful while the first is still walking; a settled run frees its slot.
	if report, err := c.runs.get(first.ID); err == nil && report.Status == RunRunning {
		_, err := c.runs.start(context.Background(), archive, VerificationConfig{Closure: "$archive"})
		if !errors.Is(err, ErrBusy) {
			t.Fatalf("second start err = %v, want ErrBusy", err)
		}
		if got := Categorize(err); got != CatBusy {
			t.Fatalf("category = %q, want %q", got, CatBusy)
		}
	}
}

// TestVerificationDeleteRemovesTheRecord pins that delete really removes, unlike cancel.
func TestVerificationDeleteRemovesTheRecord(t *testing.T) {
	c := newStack(t)

	started, err := c.runs.start(context.Background(), mustArchive(t, c), VerificationConfig{Closure: "$archive"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := c.runs.remove(started.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := c.runs.get(started.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete = %v, want ErrNotFound", err)
	}
	if err := c.runs.remove(started.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second remove = %v, want ErrNotFound", err)
	}
}

// TestVerificationUnknownRunIsNotFound pins the by-id arms on an id nobody issued.
func TestVerificationUnknownRunIsNotFound(t *testing.T) {
	c := newStack(t)
	for _, op := range []Operation{OpVerificationGet, OpVerificationCancel, OpVerificationDelete} {
		_, err := c.Handle(context.Background(), &Request{Op: op, VerID: "ver-nope"})
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("%v err = %v, want ErrNotFound", op, err)
		}
	}
}

// mustArchive resolves the core's snapshot for a test that needs it directly.
func mustArchive(t *testing.T, c *Core) ranke.Archive {
	t.Helper()
	archive, err := c.archive(context.Background())
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	return archive
}

// waitForRun blocks until a run leaves the running state, reporting whether it did.
func waitForRun(t *testing.T, c *Core, id string) bool {
	t.Helper()
	for range 200 {
		report, err := c.runs.get(id)
		if err != nil {
			t.Fatalf("get %s: %v", id, err)
		}
		if report.Status != RunRunning {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
