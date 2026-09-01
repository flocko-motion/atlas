package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"

	"github.com/rankegraph/ranke-db/adapters/auth/autherr"
	"github.com/rankegraph/ranke-db/adapters/auth/jwt/jwttest"
	"github.com/rankegraph/ranke-db/config/scope"
)

// jsonWebKeySet is the one-key JWKS document mutableJWKSServer serves.
func jsonWebKeySet(kid string, pub any) jose.JSONWebKeySet {
	return jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: pub, KeyID: kid, Algorithm: string(jose.EdDSA), Use: "sig"}}}
}

// newJWKSAuth builds the backend against url, closing its background refresh loop
// when the test ends.
func newJWKSAuth(t *testing.T, url string, extra map[string]string) *Auth {
	t.Helper()
	values := map[string]string{"type": "jwt", "algorithm": "EdDSA", "jwks_url": url}
	for k, v := range extra {
		values[k] = v
	}
	a, err := New(context.Background(), scope.Literal(values))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(a.Close)
	return a
}

// TestJWKSUnreachableURLFailsFast: New does the first fetch synchronously — an
// unreachable JWKS URL fails the whole backend at launch, the same standard every
// other adapter's config is held to, rather than starting and rejecting every
// request until it happens to come back.
func TestJWKSUnreachableURLFailsFast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // closed before New ever asks it for anything

	if _, err := New(context.Background(), scope.Literal(map[string]string{
		"type": "jwt", "algorithm": "EdDSA", "jwks_url": url,
	})); err == nil {
		t.Fatal("unreachable jwks_url: want error, got nil")
	}
}

// TestJWKSInvalidBodyFailsFast: the endpoint answers, but not with a JWKS document.
func TestJWKSInvalidBodyFailsFast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	if _, err := New(context.Background(), scope.Literal(map[string]string{
		"type": "jwt", "algorithm": "EdDSA", "jwks_url": srv.URL,
	})); err == nil {
		t.Fatal("invalid JWKS body: want error, got nil")
	}
}

// TestJWKSUnknownKidIsRejected: a token whose "kid" is not in the fetched set is
// refused, whether it belongs to no one or to a key that has since rotated out.
func TestJWKSUnknownKidIsRejected(t *testing.T) {
	pub, _ := jwttest.GenerateKey(t)
	_, otherPriv := jwttest.GenerateKey(t)
	url, stop := jwttest.JWKSServer(t, "known-kid", pub)
	defer stop()
	a := newJWKSAuth(t, url, nil)

	token := jwttest.SignWithKid(t, otherPriv, "unknown-kid", josejwt.Claims{
		Subject: "conformance-account", Expiry: josejwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	if _, err := a.Authenticate(context.Background(), token); !errors.Is(err, autherr.ErrUnauthenticated) {
		t.Fatalf("unknown kid: err=%v, want ErrUnauthenticated", err)
	}
}

// TestJWKSRefreshPicksUpARotatedKey: a key minted after the backend started, once
// the background loop has fetched it, verifies — the whole point of JWKS over a
// static key. A short refresh interval stands in for a real rotation's timescale.
func TestJWKSRefreshPicksUpARotatedKey(t *testing.T) {
	server := newMutableJWKSServer(t)
	defer server.Close()
	oldPub, _ := jwttest.GenerateKey(t)
	server.setKey("v1", oldPub)

	a := newJWKSAuth(t, server.URL, map[string]string{"jwks_refresh": "20ms"})

	newPub, newPriv := jwttest.GenerateKey(t)
	server.setKey("v2", newPub) // the key this test actually signs with

	token := jwttest.SignWithKid(t, newPriv, "v2", josejwt.Claims{
		Subject: "conformance-account", Expiry: josejwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	deadline := time.Now().Add(2 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, lastErr = a.Authenticate(context.Background(), token); lastErr == nil {
			return // the background refresh picked up v2
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("rotated key never verified within the deadline, last error: %v", lastErr)
}

// TestJWKSStaleSetServesThroughAFailedRefresh: once fetched, a key stays verifiable
// even if the endpoint goes on to fail — a transient outage must not lock every
// request out just because the background loop's next tick could not reach it.
func TestJWKSStaleSetServesThroughAFailedRefresh(t *testing.T) {
	server := newMutableJWKSServer(t)
	defer server.Close()
	pub, priv := jwttest.GenerateKey(t)
	server.setKey("v1", pub)

	a := newJWKSAuth(t, server.URL, map[string]string{"jwks_refresh": "20ms"})
	server.fail(true) // every fetch after the first (already synchronous, in New) fails

	token := jwttest.SignWithKid(t, priv, "v1", josejwt.Claims{
		Subject: "conformance-account", Expiry: josejwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	time.Sleep(100 * time.Millisecond) // let several failed refresh ticks pass
	if _, err := a.Authenticate(context.Background(), token); err != nil {
		t.Fatalf("stale set should still verify v1: err=%v", err)
	}
}

// TestNewValidatesConfigBeforeFetchingJWKS pins the fix for a goroutine leak: New
// used to start the JWKS background refresh before account_claim/audience/issuer
// were parsed, so a bad value among those orphaned a running goroutine nothing
// could Close. account_claim is now checked first, so a bad one must fail without
// the server ever being asked — reverting the reorder makes this see one request.
func TestNewValidatesConfigBeforeFetchingJWKS(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer srv.Close()

	_, err := New(context.Background(), scope.Literal(map[string]string{
		"type": "jwt", "algorithm": "EdDSA", "jwks_url": srv.URL, "account_claim": "",
	}))
	if err == nil {
		t.Fatal("empty account_claim alongside jwks_url: want error, got nil")
	}
	if n := requests.Load(); n != 0 {
		t.Fatalf("jwks_url was fetched %d time(s) before the bad account_claim was caught, want 0", n)
	}
}

// mutableJWKSServer serves a JWKS that a test can swap or fail on demand — the
// rotation and stale-on-failure cases need the response to change mid-test, which
// jwttest.JWKSServer's fixed body cannot do.
type mutableJWKSServer struct {
	*httptest.Server
	body   atomic.Pointer[[]byte]
	failed atomic.Bool
}

func newMutableJWKSServer(t *testing.T) *mutableJWKSServer {
	t.Helper()
	s := &mutableJWKSServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if s.failed.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if body := s.body.Load(); body != nil {
			_, _ = w.Write(*body)
		}
	}))
	return s
}

func (s *mutableJWKSServer) setKey(kid string, pub any) {
	set := jsonWebKeySet(kid, pub)
	body, _ := json.Marshal(set)
	s.body.Store(&body)
}

func (s *mutableJWKSServer) fail(v bool) { s.failed.Store(v) }
