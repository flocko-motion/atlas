// package: jwt / authn
// type:    adapter (JWKS key source)
// job:     fetch and refresh a JSON Web Key Set — the key source for a rotating issuer
// limits:  the only network access in this backend; Authenticate only reads the cached set
package jwt

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

// defaultJWKSRefresh is how often a jwksSource re-fetches in the background, unless
// config names a different interval.
const defaultJWKSRefresh = 5 * time.Minute

// maxJWKSBodyBytes bounds a fetch against a misbehaving or compromised issuer — a
// key set has no legitimate reason to approach this.
const maxJWKSBodyBytes = 1 << 20 // 1MB

// jwksSource holds a periodically-refreshed key set fetched from a URL. A fetch
// failure after the first leaves the previous set in place — a transient outage
// should not lock every request out, only silence forever (every key eventually
// rotating out of an uncached set) should.
type jwksSource struct {
	url      string
	client   *http.Client
	current  atomic.Pointer[jose.JSONWebKeySet]
	stop     chan struct{}
	stopOnce sync.Once
}

// newJWKSSource fetches once, synchronously, so New fails fast on an unreachable URL
// or an invalid response — the same "resolved at launch" standard every other
// adapter's config is held to — then starts refreshing every interval in the background.
func newJWKSSource(url string, interval time.Duration) (*jwksSource, error) {
	s := &jwksSource{url: url, client: &http.Client{Timeout: 10 * time.Second}, stop: make(chan struct{})}
	if err := s.fetch(); err != nil {
		return nil, err
	}
	go s.refreshLoop(interval)
	return s, nil
}

func (s *jwksSource) fetch() error {
	resp, err := s.client.Get(s.url)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", s.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: status %d", s.url, resp.StatusCode)
	}
	var set jose.JSONWebKeySet
	body := io.LimitReader(resp.Body, maxJWKSBodyBytes)
	if err := json.NewDecoder(body).Decode(&set); err != nil {
		return fmt.Errorf("decoding %s: %w", s.url, err)
	}
	s.current.Store(&set)
	return nil
}

// refreshLoop re-fetches on interval until stop closes. A failed fetch is dropped —
// see the type doc for why the stale set stays in service instead.
func (s *jwksSource) refreshLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = s.fetch()
		case <-s.stop:
			return
		}
	}
}

// key looks kid up in the last successfully fetched set. A lookup miss — rotated
// out, or not yet in cache — is not an error here; the caller treats it the same as
// any other reason a token fails to verify.
func (s *jwksSource) key(kid string) (any, bool) {
	set := s.current.Load()
	if set == nil {
		return nil, false
	}
	matches := set.Key(kid)
	if len(matches) == 0 {
		return nil, false
	}
	return matches[0].Key, true
}

// close stops the background refresh loop. Safe to call more than once.
func (s *jwksSource) close() {
	s.stopOnce.Do(func() { close(s.stop) })
}
