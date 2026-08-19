// package: sequencer / coordination
// type:    adapter
// job:     a steerable clock for --dev — real time until told otherwise
// limits:  time source only; who may steer it is core's access decision (-> core)
package sequencer

import (
	"sync"
	"time"
)

// SteerableClock is a time source a caller can push forward — epoch-pinned until the
// first Advance, then pinned there until told to move again. Epoch, not real time,
// because bt₀ mints from this clock at boot, before any --dev caller can steer it, and
// every later branch table chains back to bt₀ (R-C6MERGE) — same reasoning as the
// sequencer's own identity in sequencer.go's contributor.
type SteerableClock struct {
	mu  sync.Mutex
	set time.Time // zero until the first Advance; Now reads the epoch until then
}

// NewSteerableClock returns a clock reading the epoch until Advance is first called.
func NewSteerableClock() *SteerableClock { return &SteerableClock{} }

// Now is this clock's now func() time.Time source for sequencer.New.
func (c *SteerableClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.set.IsZero() {
		return time.Unix(0, 0).UTC()
	}
	return c.set
}

// Advance moves the clock to at least t, returning its position afterward. t compares
// against the clock's own last position — zero before the first call, which is why that
// first call always lands exactly on t, whatever real time says. A t at or behind the
// current position is a no-op that just reports where the clock already is.
func (c *SteerableClock) Advance(t time.Time) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t.After(c.set) {
		c.set = t
	}
	return c.set
}
