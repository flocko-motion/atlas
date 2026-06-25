package seal

import (
	"errors"
	"testing"
	"time"
)

func TestFromEnvOpens(t *testing.T) {
	t.Setenv("RANKE_MASTER", "master")
	var got string
	g := New(5*time.Second, func(s []byte) error { got = string(s); return nil })

	present, err := g.FromEnv("RANKE_MASTER")
	if !present || err != nil {
		t.Fatalf("FromEnv = (%v, %v), want (true, nil)", present, err)
	}
	if got != "master" {
		t.Fatalf("Open received %q, want master", got)
	}
	if g.Sealed() {
		t.Fatal("gate still sealed after FromEnv")
	}
}

func TestFromEnvAbsentStaysSealed(t *testing.T) {
	g := New(5*time.Second, func([]byte) error { return nil })

	present, err := g.FromEnv("RANKE_MASTER_DEFINITELY_UNSET")
	if present || err != nil {
		t.Fatalf("FromEnv = (%v, %v), want (false, nil)", present, err)
	}
	if !g.Sealed() {
		t.Fatal("gate should stay sealed when env unset")
	}
}

func TestUnlockRateLimited(t *testing.T) {
	cur := time.Unix(1000, 0)
	attempts := 0
	g := New(5*time.Second, func(s []byte) error {
		attempts++
		if string(s) != "right" {
			return errors.New("wrong key")
		}
		return nil
	})
	g.now = func() time.Time { return cur }

	// 1st attempt, wrong key → error, still sealed, attempt consumed.
	if err := g.Unlock([]byte("wrong")); err == nil {
		t.Fatal("expected error on wrong key")
	}
	if !g.Sealed() {
		t.Fatal("should be sealed after wrong key")
	}

	// Immediate retry → ErrTooSoon, attempt NOT consumed.
	if err := g.Unlock([]byte("right")); err != ErrTooSoon {
		t.Fatalf("err = %v, want ErrTooSoon", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1 (rate-limited)", attempts)
	}

	// Advance past minGap → allowed; right key → opens.
	cur = cur.Add(6 * time.Second)
	if err := g.Unlock([]byte("right")); err != nil {
		t.Fatalf("Unlock after gap: %v", err)
	}
	if g.Sealed() {
		t.Fatal("should be open")
	}

	// Already open → no-op even with garbage.
	if err := g.Unlock([]byte("garbage")); err != nil {
		t.Fatalf("Unlock when open: %v", err)
	}
}
