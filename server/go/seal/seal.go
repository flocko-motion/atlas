// package: seal / gate
// type:    logic
// job:     gate a service that boots LOCKED until secret-zero arrives (env or rate-limited /unlock)
// limits:  no vault/crypto specifics — the vault-opening is injected as an Open callback (-> vault.Opener)
//
// Package seal gates a service that boots LOCKED: it must receive
// "secret-zero" (e.g. the vault master key) before its vault can be opened.
//
// Secret-zero arrives by either of two paths, by design:
//   - FromEnv — the environment at boot (simple; for examples/dev), or
//   - Unlock — pushed to the /unlock endpoint by an external unlocker
//     (production: no secret-zero at rest on the host; a restart leaves the
//     service sealed until the unlocker re-delivers).
//
// Unlock is rate-limited (one real attempt per minGap) to throttle brute
// force against the endpoint. The actual vault-opening is injected as an
// Open callback, so this package carries no vault/crypto specifics.
package seal

import (
	"errors"
	"os"
	"sync"
	"time"
)

// ErrTooSoon is returned by Unlock when called within minGap of the previous
// real attempt — the brute-force throttle.
var ErrTooSoon = errors.New("seal: unlock attempted too soon, try again later")

// Open opens the vault from secret-zero. A non-nil error (e.g. a wrong key →
// decryption failed) leaves the gate sealed.
type Open func(secretZero []byte) error

// Gate is the sealed/unsealed state guard. Build one with New.
type Gate struct {
	mu      sync.Mutex
	open    bool
	lastTry time.Time
	minGap  time.Duration
	onOpen  Open
	now     func() time.Time
}

// New returns a sealed Gate. minGap is the minimum spacing between real
// Unlock attempts; onOpen opens the vault from secret-zero.
func New(minGap time.Duration, onOpen Open) *Gate {
	return &Gate{minGap: minGap, onOpen: onOpen, now: time.Now}
}

// Sealed reports whether the gate is still locked. The real API should 503
// (and health report "locked") while this is true.
func (g *Gate) Sealed() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return !g.open
}

// FromEnv opens the gate from the named environment variable if it is set
// and non-empty — the boot/examples path. Not rate-limited (a single local
// attempt at startup). Returns whether secret-zero was present.
func (g *Gate) FromEnv(name string) (present bool, err error) {
	s, ok := os.LookupEnv(name)
	if !ok || s == "" {
		return false, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return true, g.openLocked([]byte(s))
}

// Unlock attempts to open the gate with secret-zero — the /unlock path.
// Rate-limited: ErrTooSoon if called within minGap of the previous real
// attempt (a too-soon call does not consume an attempt). A no-op if already
// open. A wrong secret returns the Open callback's error and leaves the gate
// sealed, so the next attempt must again wait minGap.
func (g *Gate) Unlock(secretZero []byte) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open {
		return nil
	}
	now := g.now()
	if !g.lastTry.IsZero() && now.Sub(g.lastTry) < g.minGap {
		return ErrTooSoon
	}
	g.lastTry = now
	return g.openLocked(secretZero)
}

// openLocked runs the Open callback; caller must hold g.mu.
func (g *Gate) openLocked(secretZero []byte) error {
	if g.open {
		return nil
	}
	if err := g.onOpen(secretZero); err != nil {
		return err
	}
	g.open = true
	return nil
}
