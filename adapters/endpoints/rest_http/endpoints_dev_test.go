// package: rest_http / transport
// type:    test
// job:     POST /dev/clock — refused when unwired, working when it is
package rest_http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flocko-motion/rankedb/adapters/auth"
	"github.com/flocko-motion/rankedb/config/scope"
	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/internal/core/access"
)

// devServerFor builds the endpoint over a core with the given extra options — this
// route's own concern, WithDevClock, being the one thing the shared helpers don't
// thread through.
func devServerFor(t *testing.T, opts ...core.Option) http.Handler {
	t.Helper()
	ctx := context.Background()
	a, err := auth.New(ctx, scope.Literal(map[string]string{"type": "noauth", "subject": "ops"}))
	if err != nil {
		t.Fatalf("auth.New: %v", err)
	}
	set, err := auth.NewSet([]auth.Auth{a})
	if err != nil {
		t.Fatalf("auth.NewSet: %v", err)
	}
	chk, err := access.New(map[string][]string{"ops": nil})
	if err != nil {
		t.Fatalf("access.New: %v", err)
	}
	s, err := New(ctx, scope.Literal(map[string]string{"addr": ":0"}), core.New(set, chk, nil, nil, opts...))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s.srv.Handler
}

// postDevClock sends the request and returns the response.
func postDevClock(t *testing.T, h http.Handler, at time.Time) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"time": at.Format(time.RFC3339)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/dev/clock", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestDevClockRefusedWithoutDevMode pins that a stack never launched --dev answers 501
// — the route exists (mounted by every stack, per the generated router) but the
// capability behind it does not.
func TestDevClockRefusedWithoutDevMode(t *testing.T) {
	rec := postDevClock(t, devServerFor(t), time.Now())
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
}

// TestDevClockAdvancesWhenWired pins the working path end to end: the wire body reaches
// the wired clock func, and its answer comes back as the response.
func TestDevClockAdvancesWhenWired(t *testing.T) {
	advance := func(t time.Time) time.Time { return t }
	h := devServerFor(t, core.WithDevClock(advance))

	want := time.Now().UTC().Add(9 * time.Hour).Truncate(time.Second)
	rec := postDevClock(t, h, want)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var got struct {
		Time time.Time `json:"time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.Time.Equal(want) {
		t.Errorf("response time = %s, want %s", got.Time, want)
	}
}
