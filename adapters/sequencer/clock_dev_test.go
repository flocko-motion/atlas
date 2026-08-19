package sequencer_test

import (
	"testing"
	"time"

	"github.com/flocko-motion/rankedb/adapters/sequencer"
)

// TestSteerableClockReadsEpochUntilAdvanced pins the default: a fresh clock reads the
// epoch, not real time — bt₀ mints from this clock at server boot, before any --dev
// client can possibly have steered it, and every later branch table chains back to it
// (R-C6MERGE), so it must precede whatever a fixture's own past-dated story asks for.
func TestSteerableClockReadsEpochUntilAdvanced(t *testing.T) {
	c := sequencer.NewSteerableClock()
	if got := c.Now(); !got.Equal(time.Unix(0, 0).UTC()) {
		t.Errorf("Now() = %s, want the epoch", got)
	}
}

// TestSteerableClockFirstAdvanceLandsInThePast is the toy example a fixture generator
// actually needs: a story dated years before "now" (2024, say, when today is 2026).
// The bug this pins: comparing the first Advance against real wall time as the floor
// rejects any date before today, silently stranding the clock at real time forever —
// exactly what a re-run of this test on ranke-db's release generator hit.
func TestSteerableClockFirstAdvanceLandsInThePast(t *testing.T) {
	c := sequencer.NewSteerableClock()

	past := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // long before whenever this test runs
	if got := c.Advance(past); !got.Equal(past) {
		t.Fatalf("first Advance(%s) = %s, want the requested instant exactly", past, got)
	}
	if got := c.Now(); !got.Equal(past) {
		t.Fatalf("Now() after Advance(%s) = %s, want %s", past, got, past)
	}

	// The story moves forward from there — each event later than the last, all still
	// long before real time — and the clock must track every step of it.
	later := past.Add(48 * time.Hour)
	if got := c.Advance(later); !got.Equal(later) {
		t.Fatalf("Advance(%s) = %s, want %s", later, got, later)
	}
}

// TestSteerableClockAdvanceNeverGoesBackward pins the property everything here rests
// on: a request older than the clock's position is a no-op, not a regression — a
// merge's witnessed time moving backward would break every guarantee built on it.
func TestSteerableClockAdvanceNeverGoesBackward(t *testing.T) {
	c := sequencer.NewSteerableClock()

	forward := time.Now().UTC().Add(48 * time.Hour)
	if got := c.Advance(forward); !got.Equal(forward) {
		t.Fatalf("Advance(forward) = %s, want %s", got, forward)
	}
	if got := c.Now(); !got.Equal(forward) {
		t.Fatalf("Now() after Advance = %s, want %s", got, forward)
	}

	backward := forward.Add(-time.Hour)
	if got := c.Advance(backward); !got.Equal(forward) {
		t.Errorf("Advance(backward) = %s, want the clock unchanged at %s", got, forward)
	}
	if got := c.Now(); !got.Equal(forward) {
		t.Errorf("Now() after a backward Advance = %s, want unchanged at %s", got, forward)
	}
}

// TestSteerableClockTicksAreMonotonic pins that a Sequencer built over this clock sees
// created_at track the requested schedule across several contributions in a row, not
// just the first one.
func TestSteerableClockTicksAreMonotonic(t *testing.T) {
	c := sequencer.NewSteerableClock()
	base := time.Now().UTC().Add(24 * time.Hour)
	for i := range 5 {
		want := base.Add(time.Duration(i) * time.Hour)
		if got := c.Advance(want); !got.Equal(want) {
			t.Fatalf("tick %d: Advance(%s) = %s", i, want, got)
		}
		if got := c.Now(); !got.Equal(want) {
			t.Fatalf("tick %d: Now() = %s, want %s", i, got, want)
		}
	}
}
