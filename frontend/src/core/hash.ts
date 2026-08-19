/**
 * package: core / hash
 * type:    logic
 * job:     a small deterministic string hash, shared wherever a claim needs one stable
 *          pseudo-random value derived from its id or type — never randomness itself
 * limits:  headless, single-purpose; what the hash is used for is the caller's
 */

/** hashString is a small, fast, deterministic string hash (FNV-1a, 32-bit). */
export function hashString(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}
