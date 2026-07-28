/**
 * Deterministic generator for a Ranke-Graph shaped archive.
 *
 * The point of a render spike is that the mock graph stresses the renderer the
 * way the real one will, so the topology is not random: claims only ever
 * reference *earlier* claims (acyclic, monotonic `created_at`), derivations cite
 * recent inputs (locality → clusters), entities accumulate references by
 * preferential attachment (hubs → the layout's hard case), and periodic
 * `contribution/head` claims consolidate open claims into stars, as a real
 * archive's provenance does.
 *
 * Ids are synthetic, but multibase-base32 shaped and 57 chars long, so string
 * keys cost the renderer what real claim ids will cost it.
 */

import type { ClaimId, MockArchive, MockClaim, MockEdge } from './model.ts';
import { NODE_CLASSES, classOf } from './model.ts';

const B32 = 'abcdefghijklmnopqrstuvwxyz234567';

/** Claim-class mix, in claims per 1000. Sums to 1000 minus the fixed extras. */
const MIX = {
  source: 200,
  derivation: 440,
  entity: 200,
  relation: 145,
  head: 15,
} as const;

/** How many contributor claims (initial nodes) the archive has. */
const CONTRIBUTORS = 4;
/** Open claims a `contribution/head` consolidates at most. */
const HEAD_FANIN = 32;
/** Recency window for derivation inputs — the source of visual clustering. */
const RECENT = 2048;
/** Probability an input is drawn from the recency window rather than all history. */
const P_RECENT = 0.75;
/**
 * Preferential attachment is damped by drawing mostly from a sliding window over
 * the reference pool. Undamped rich-get-richer produced a single entity holding
 * a quarter of all relation references (degree 25k at 100k claims) — a shape no
 * real archive has, and one that would flatter Barnes-Hut by concentrating the
 * mass in one cell.
 */
const HUB_WINDOW = 8192;
/** Probability a relation member is drawn from the recent window of the pool. */
const P_HUB_RECENT = 0.7;
/** Branches the final branch table names. */
const BRANCHES = 6;

/** mulberry32 is a small deterministic PRNG — same seed, same archive. */
function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** IdMinter hands out unique, realistically shaped claim ids cheaply. */
class IdMinter {
  private chunks: string[] = [];
  private n = 0;

  constructor(rnd: () => number) {
    for (let i = 0; i < 1024; i++) {
      let s = '';
      for (let j = 0; j < 5; j++) s += B32[(rnd() * 32) | 0];
      this.chunks.push(s);
    }
  }

  /** next mints an id: 'b' + 50 random base32 chars + a 6-char unique tail. */
  next(rnd: () => number): ClaimId {
    let s = 'b';
    for (let i = 0; i < 10; i++) s += this.chunks[(rnd() * 1024) | 0];
    let c = this.n++;
    for (let i = 0; i < 6; i++) {
      s += B32[c & 31];
      c >>>= 5;
    }
    return s;
  }
}

/** Pool is a growable index bag; drawing from it gives degree-proportional picks. */
class Pool {
  private buf: Int32Array;
  private len = 0;

  constructor(capacity = 1024) {
    this.buf = new Int32Array(capacity);
  }

  push(v: number): void {
    if (this.len === this.buf.length) {
      const grown = new Int32Array(this.buf.length * 2);
      grown.set(this.buf);
      this.buf = grown;
    }
    this.buf[this.len++] = v;
  }

  get size(): number {
    return this.len;
  }

  /** draw picks degree-proportionally over the whole pool. */
  draw(rnd: () => number): number {
    return this.buf[(rnd() * this.len) | 0];
  }

  /** drawWindowed favours the most recent `window` entries, damping hub runaway. */
  drawWindowed(rnd: () => number, window: number, pRecent: number): number {
    if (rnd() < pRecent) {
      const lo = Math.max(0, this.len - window);
      return this.buf[lo + ((rnd() * (this.len - lo)) | 0)];
    }
    return this.buf[(rnd() * this.len) | 0];
  }
}

/** pick returns a deterministic element of a readonly list. */
function pick<T>(list: readonly T[], rnd: () => number): T {
  return list[(rnd() * list.length) | 0];
}

/** schedule spreads the class mix over n claims so classes interleave in time. */
function schedule(n: number): string[] {
  const weights = Object.entries(MIX);
  const total = weights.reduce((s, [, w]) => s + w, 0);
  const out: string[] = [];
  const acc = new Float64Array(weights.length);
  for (let i = 0; i < n; i++) {
    let best = 0;
    for (let j = 0; j < weights.length; j++) {
      acc[j] += weights[j][1] / total;
      if (acc[j] > acc[best]) best = j;
    }
    acc[best] -= 1;
    out.push(weights[best][0]);
  }
  return out;
}

/**
 * generate builds an archive of about `n` claims (plus contributors, heads and
 * one branch table). Same `seed` always yields the same archive.
 */
export function generate(n: number, seed = 0x5eed): MockArchive {
  const t0 = performance.now();
  const rnd = mulberry32(seed);
  const minter = new IdMinter(rnd);
  const claims: MockClaim[] = [];
  const byClass: Record<string, number> = {};

  const contributors: number[] = [];
  const sources: number[] = [];
  const derivations: number[] = [];
  const entityPool = new Pool();
  const entities: number[] = [];
  const open: number[] = [];
  const heads: number[] = [];

  let clock = Date.UTC(2024, 0, 1);
  let edgeCount = 0;

  const emit = (type: string, edges: MockEdge[], label: string, size?: number): number => {
    clock += 1 + ((rnd() * 9000) | 0);
    const claim: MockClaim = {
      id: minter.next(rnd),
      type,
      created_at: clock,
      label,
      edges,
    };
    if (size !== undefined) {
      claim.content_size = size;
      claim.encoding = 'text/plain';
    }
    const cls = classOf(type);
    byClass[cls] = (byClass[cls] ?? 0) + 1;
    edgeCount += edges.length;
    claims.push(claim);
    return claims.length - 1;
  };

  // Initial nodes: contributors sign everything that follows.
  for (let i = 0; i < CONTRIBUTORS; i++) {
    contributors.push(emit('contribution/contributor', [], `contributor-${i}`, 96));
  }

  const contributorEdge = (): MockEdge => ({
    type: 'contribution/contributor',
    reference: claims[contributors[(rnd() * contributors.length) | 0]].id,
  });

  /** drawInput picks an earlier claim, biased towards recent ones. */
  const drawInput = (from: number[]): number | undefined => {
    if (from.length === 0) return undefined;
    if (rnd() < P_RECENT) {
      const lo = Math.max(0, from.length - RECENT);
      return from[lo + ((rnd() * (from.length - lo)) | 0)];
    }
    return from[(rnd() * from.length) | 0];
  };

  for (const kind of schedule(n)) {
    let idx: number;

    if (kind === 'source' || (kind === 'derivation' && sources.length === 0)) {
      const subtype = pick(NODE_CLASSES.source, rnd);
      idx = emit(
        `source/${subtype}`,
        [contributorEdge()],
        `${subtype} ${claims.length}`,
        512 + ((rnd() * 262144) | 0),
      );
      sources.push(idx);
    } else if (kind === 'derivation') {
      const edges: MockEdge[] = [contributorEdge()];
      const inputs = 1 + ((rnd() * 4) | 0);
      for (let i = 0; i < inputs; i++) {
        const from = rnd() < 0.6 || derivations.length === 0 ? sources : derivations;
        const ref = drawInput(from);
        if (ref !== undefined) {
          edges.push({ type: `derivation/input`, reference: claims[ref].id });
        }
      }
      const subtype = pick(NODE_CLASSES.derivation, rnd);
      idx = emit(`derivation/${subtype}`, edges, `${subtype} ${claims.length}`, 64 + ((rnd() * 4096) | 0));
      derivations.push(idx);
    } else if (kind === 'entity' && derivations.length > 0) {
      const edges: MockEdge[] = [contributorEdge()];
      const inputs = 1 + ((rnd() * 3) | 0);
      for (let i = 0; i < inputs; i++) {
        const ref = drawInput(derivations);
        if (ref !== undefined) edges.push({ type: 'derivation/input', reference: claims[ref].id });
      }
      const subtype = pick(NODE_CLASSES.entity, rnd);
      idx = emit(`entity/${subtype}`, edges, `${subtype} ${claims.length}`, 32 + ((rnd() * 512) | 0));
      entities.push(idx);
      entityPool.push(idx);
    } else if (kind === 'relation' && entities.length > 1) {
      const edges: MockEdge[] = [contributorEdge()];
      const arity = rnd() < 0.9 ? 2 : 3;
      const seen = new Set<number>();
      for (let i = 0; i < arity; i++) {
        const ref = entityPool.drawWindowed(rnd, HUB_WINDOW, P_HUB_RECENT);
        if (seen.has(ref)) continue;
        seen.add(ref);
        entityPool.push(ref); // referenced entities grow hubs
        edges.push({
          type: `relation/member`,
          reference: claims[ref].id,
          relation_direction: i === 0 ? 1 : -1,
        });
      }
      if (seen.size < 2) continue;
      const support = drawInput(derivations);
      if (support !== undefined) {
        edges.push({ type: 'derivation/input', reference: claims[support].id });
      }
      const subtype = pick(NODE_CLASSES.relation, rnd);
      idx = emit(`relation/${subtype}`, edges, subtype, undefined);
    } else if (kind === 'head' && open.length > 0) {
      const edges: MockEdge[] = [contributorEdge()];
      const take = Math.min(HEAD_FANIN, open.length);
      for (let i = 0; i < take; i++) {
        const ref = open[open.length - 1 - i];
        edges.push({ type: 'contribution/head', reference: claims[ref].id });
      }
      open.length = Math.max(0, open.length - take);
      idx = emit('contribution/head', edges, `head ${heads.length}`, undefined);
      heads.push(idx);
    } else {
      // Class not yet possible this early (no inputs exist) — fall back to a source.
      const subtype = pick(NODE_CLASSES.source, rnd);
      idx = emit(`source/${subtype}`, [contributorEdge()], `${subtype} ${claims.length}`, 512);
      sources.push(idx);
    }

    open.push(idx);
    if (open.length > HEAD_FANIN * 8) open.splice(0, open.length - HEAD_FANIN * 8);
  }

  // The branch table: the archive head, naming one head claim per branch.
  const branches: Record<string, ClaimId> = {};
  const tableEdges: MockEdge[] = [contributorEdge()];
  const names = ['main', 'ingest', 'review', 'entities', 'scratch', 'archive'];
  for (let i = 0; i < Math.min(BRANCHES, heads.length); i++) {
    const head = claims[heads[heads.length - 1 - i]];
    branches[names[i]] = head.id;
    tableEdges.push({ type: 'contribution/branch', reference: head.id, name: names[i] });
  }
  const tableIdx = emit('contribution/branches', tableEdges, 'branch table', undefined);

  return {
    claims,
    head: claims[tableIdx].id,
    branches,
    stats: {
      claims: claims.length,
      edges: edgeCount,
      byClass,
      generateMs: performance.now() - t0,
    },
  };
}
