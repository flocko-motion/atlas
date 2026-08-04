package rest_http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newCORSServer builds an endpoint whose transport declares these origins.
func newCORSServer(t *testing.T, origins string) http.Handler {
	t.Helper()
	return newServerWith(t, everyReadRight,
		map[string]string{"addr": ":0", "allowedOrigins": origins}, nil, nil)
}

// TestCORSOffByDefault pins that a config declaring no origin stays shut: a browser reads a
// missing Allow-Origin as refused, which is right for a server nobody is meant to browse.
func TestCORSOffByDefault(t *testing.T) {
	h := newTestServer(t)
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	h.ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q with no origins configured, want none", got)
	}
}

// TestCORSAdmitsDeclaredOrigin drives the two things a browser does: the preflight, and the
// read that follows it.
func TestCORSAdmitsDeclaredOrigin(t *testing.T) {
	const origin = "http://localhost:5173"
	h := newCORSServer(t, origin+", http://127.0.0.1:5173")

	t.Run("preflight is answered without routing", func(t *testing.T) {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodOptions, "/query", nil)
		r.Header.Set("Origin", origin)
		r.Header.Set("Access-Control-Request-Method", "POST")
		r.Header.Set("Access-Control-Request-Headers", "content-type")
		h.ServeHTTP(rec, r)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204: %s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("Allow-Origin = %q, want %q", got, origin)
		}
		for _, want := range []string{"POST", "GET"} {
			if got := rec.Header().Get("Access-Control-Allow-Methods"); !contains(got, want) {
				t.Errorf("Allow-Methods = %q, missing %s", got, want)
			}
		}
		// The credential headers the contract enumerates have to be allowed, or a browser
		// strips them and every authenticated read is anonymous.
		for _, want := range []string{"Authorization", "X-API-Key", "Content-Type"} {
			if got := rec.Header().Get("Access-Control-Allow-Headers"); !contains(got, want) {
				t.Errorf("Allow-Headers = %q, missing %s", got, want)
			}
		}
	})

	t.Run("a read carries the answer and exposes ETag", func(t *testing.T) {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/health", nil)
		r.Header.Set("Origin", origin)
		h.ServeHTTP(rec, r)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("Allow-Origin = %q, want %q", got, origin)
		}
		// Without this a script cannot read ETag, so the conditional reads the contract
		// promises are unusable from a browser.
		if got := rec.Header().Get("Access-Control-Expose-Headers"); !contains(got, "ETag") {
			t.Errorf("Expose-Headers = %q, want ETag", got)
		}
		if got := rec.Header().Get("Vary"); !contains(got, "Origin") {
			t.Errorf("Vary = %q, want Origin — the answer differs per origin", got)
		}
	})

	t.Run("an undeclared origin gets no answer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/health", nil)
		r.Header.Set("Origin", "https://not-declared.example")
		h.ServeHTTP(rec, r)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Allow-Origin = %q for an undeclared origin, want none", got)
		}
	})

	t.Run("a request with no origin is untouched", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 — curl sends no Origin", rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Allow-Origin = %q with no Origin sent", got)
		}
	})
}

// TestCORSWildcard pins the dev posture: every origin, still with no credentialed access,
// since the API takes its credential from a header rather than a cookie.
func TestCORSWildcard(t *testing.T) {
	h := newCORSServer(t, "*")
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.Header.Set("Origin", "https://anywhere.example")
	h.ServeHTTP(rec, r)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://anywhere.example" {
		t.Fatalf("Allow-Origin = %q, want the origin echoed", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Allow-Credentials = %q — no origin is trusted with the browser's credentials", got)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
