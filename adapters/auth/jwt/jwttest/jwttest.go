// package: jwttest / authn
// type:    test-support
// job:     the jwt backend's conformance setup hook and token fixtures, beside the backend
// limits:  a test helper; only the conformance driver imports it (-> adapters/auth)
//
// Every fixture here signs with EdDSA over a freshly generated ed25519 key: one concrete
// algorithm and key type exercises the shared verify/parse/claims path identically to any
// other the backend accepts, since jwt.go treats them uniformly once New has resolved one.
package jwttest

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/flocko-motion/rankedb/config/scope"
)

// Setup returns a jwt config verifying EdDSA tokens against a freshly generated key, a
// token minted for account with a one-hour expiry, and a no-op teardown — the backend
// has no external counterpart.
func Setup(t *testing.T, account string) (scope.Section, string, func()) {
	t.Helper()
	pub, priv := GenerateKey(t)
	token := Sign(t, priv, jwt.Claims{Subject: account, Expiry: jwt.NewNumericDate(time.Now().Add(time.Hour))})
	cfg := scope.Literal(map[string]string{
		"type":      "jwt",
		"algorithm": "EdDSA",
		"key":       PublicKeyPEM(t, pub),
	})
	return cfg, token, func() {}
}

// GenerateKey mints a fresh ed25519 keypair for a test to sign fixtures with.
func GenerateKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	return pub, priv
}

// PublicKeyPEM renders a public key as PKIX PEM, the form jwt.New's "key" config
// value reads for every algorithm but HMAC.
func PublicKeyPEM(t *testing.T, pub any) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("x509.MarshalPKIXPublicKey: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

// Sign serializes claims as an EdDSA-signed JWT under priv — the one fixture shape every
// backend-specific attack test in jwt_test.go starts from and then deviates.
func Sign(t *testing.T, priv ed25519.PrivateKey, claims jwt.Claims) string {
	t.Helper()
	return SignWithKid(t, priv, "", claims)
}

// SignWithKid is Sign, tagging the token's header with kid — what a JWKS-sourced
// verification looks the signing key up by; Sign passes "" since a static-key config
// never reads the header at all.
func SignWithKid(t *testing.T, priv ed25519.PrivateKey, kid string, claims jwt.Claims) string {
	t.Helper()
	opts := &jose.SignerOptions{}
	if kid != "" {
		opts = opts.WithHeader("kid", kid)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: priv}, opts)
	if err != nil {
		t.Fatalf("jose.NewSigner: %v", err)
	}
	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	return token
}

// JWKSServer starts a real HTTP server (per the architecture spec: a network-reaching
// adapter is tested against a real counterpart, not a mock) serving a JWKS holding pub
// under kid, and returns its URL and a func to stop it.
func JWKSServer(t *testing.T, kid string, pub ed25519.PublicKey) (string, func()) {
	t.Helper()
	body := jwksBody(t, kid, pub)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	return srv.URL, srv.Close
}

// jwksBody marshals a one-key JWKS document for kid/pub.
func jwksBody(t *testing.T, kid string, pub ed25519.PublicKey) []byte {
	t.Helper()
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
		{Key: pub, KeyID: kid, Algorithm: string(jose.EdDSA), Use: "sig"},
	}}
	body, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshalling JWKS: %v", err)
	}
	return body
}

// SetupJWKS is Setup's JWKS-sourced counterpart: a real JWKS server, a token minted
// for account and signed under the key it serves, and a teardown stopping the server.
func SetupJWKS(t *testing.T, account string) (scope.Section, string, func()) {
	t.Helper()
	const kid = "conformance-kid"
	pub, priv := GenerateKey(t)
	url, stop := JWKSServer(t, kid, pub)
	token := SignWithKid(t, priv, kid, jwt.Claims{Subject: account, Expiry: jwt.NewNumericDate(time.Now().Add(time.Hour))})
	cfg := scope.Literal(map[string]string{
		"type":      "jwt",
		"algorithm": "EdDSA",
		"jwks_url":  url,
	})
	return cfg, token, stop
}
