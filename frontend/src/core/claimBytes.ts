/**
 * package: core / claimBytes
 * type:    data
 * job:     hold the claim CBOR this session has read
 * limits:  storage only; reading it is the port's, rendering it the UI's (-> core/data, ui)
 *
 * A claim's own bytes are what its id is the hash of, so they can never change once read —
 * the same reasoning core/content.ts rests on, for the content a claim declares rather than
 * the claim record itself. Two different things a claim can be asked for, so two caches
 * rather than one keyed by a made-up discriminator.
 *
 * Module scope, like the graph and the content cache: bytes are data, and data does not go
 * into React state.
 */

const cache = new Map<string, Uint8Array>();

/** claimBytesOf returns a claim's bytes already read, or null. */
export function claimBytesOf(id: string): Uint8Array | null {
  return cache.get(id) ?? null;
}

/** rememberClaimBytes keeps a claim's bytes for the session. */
export function rememberClaimBytes(id: string, bytes: Uint8Array): void {
  cache.set(id, bytes);
}

/** forgetClaimBytes empties the cache — a session reset, not something a read does. */
export function forgetClaimBytes(): void {
  cache.clear();
}

/** cachedClaimBytesCount is how many claims' bytes are held, for a test to pin the cache. */
export function cachedClaimBytesCount(): number {
  return cache.size;
}
