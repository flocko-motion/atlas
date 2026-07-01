// package: signer / crypto
// type:    interface
// job:     the Signer port — the server's own signing identity; it signs the merge claims, the private key never leaving the backend
// limits:  contract only; backends live in sub-packages (-> adapters/signer/inmemory, openbao, azure)
//
// Package signer defines the server's merge-signing identity. The server signs
// the contribution/head and branch-table claims (the hard timestamp) with it;
// the contributor key that signs the CLAIMS themselves is the application's,
// not this.
//
// Per the foundation paper, identity is id(v) = Sign(H(S(v))), where Sign takes
// a hash and a private key and returns a deterministic, self-describing
// signature. A Signer IS a crypto.Signer: it performs that Sign over a digest
// and never hands back key material, so an in-process key (inmemory) and a key
// that never leaves an HSM/KMS (OpenBao Transit, Azure Key Vault) present
// identically. ranke-go composes the digest (S then H) and the multikey
// framing; this port supplies only the key's Sign + Public. core passes it to
// ranke-go (WithSigningKey) to attest merges and reads Public() to name the
// identity.
package signer

import "crypto"

// Signer is the server's signing identity — an alias for crypto.Signer, the
// exact type ranke-go consumes (WithSigningKey, ClaimBuilder.Sign,
// Contributor.SigningKey). Sign produces the signature over a claim's digest;
// Public names the identity. Backends supply one: in-memory (an ed25519 key is
// already a crypto.Signer), OpenBao Transit, Azure Key Vault (-> sub-packages).
type Signer = crypto.Signer
