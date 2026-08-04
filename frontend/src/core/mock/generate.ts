/**
 * package: core / mock
 * type:    logic
 * job:     generate a deterministic Ranke-Graph shaped archive
 * limits:  shapes claims; serves no requests (-> core/data/source)
 *
 * Built contribution by contribution, which is what makes a Ranke-Graph tall: each mints
 * a head and a branch-table revision, so 100k claims sit ~10k contributions deep — a
 * ribbon, not a ball. Ids are synthetic but real-shaped.
 */

import type { ClaimId, MockArchive, MockClaim, MockEdge } from './model.ts';
import { NODE_CLASSES, classOf } from './model.ts';

const B32 = 'abcdefghijklmnopqrstuvwxyz234567';

/** Content-class mix within a contribution, in claims per 1000. */
const MIX = {
  source: 205,
  derivation: 450,
  entity: 205,
  relation: 140,
} as const;

/** How many contributor claims (initial nodes) the archive has. */
const CONTRIBUTORS = 4;
/**
 * Claims per contribution, counting its head and branch-table revisions. Ten is
 * the figure that puts a 100k-claim archive at a height of ~10k, which is the
 * shape a real archive has.
 */
const CLAIMS_PER_CONTRIBUTION = 10;
/** Fraction of contributions that are bulk ingests rather than ordinary commits. */
const P_BULK = 0.02;
/** How much larger a bulk ingest is than an ordinary contribution. */
const BULK_FACTOR = 25;
/** Recency window for derivation inputs — the source of visual clustering. */
const RECENT = 2048;
/** Probability an input is drawn from the recency window rather than all history. */
const P_RECENT = 0.75;
/** Sliding window over the entity reference pool, damping hub runaway. */
const HUB_WINDOW = 8192;
/** Probability a relation member is drawn from that recent window. */
const P_HUB_RECENT = 0.7;
/** Branch names the archive carries; `main` takes most contributions. */
const BRANCH_NAMES = ['main', 'ingest', 'review', 'entities', 'scratch', 'archive'];

export interface GenerateOptions {
  seed?: number;
  /** Average claims per contribution, head and table included. */
  claimsPerContribution?: number;
}

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

  /** drawWindowed favours the most recent entries, damping hub runaway. */
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

/** ClassCycle emits content classes in the MIX proportions, interleaved. */
class ClassCycle {
  private acc: Float64Array;
  private names: string[];
  private weights: number[];

  constructor() {
    const entries = Object.entries(MIX);
    this.names = entries.map(([k]) => k);
    const total = entries.reduce((s, [, w]) => s + w, 0);
    this.weights = entries.map(([, w]) => w / total);
    this.acc = new Float64Array(entries.length);
  }

  next(): string {
    let best = 0;
    for (let j = 0; j < this.weights.length; j++) {
      this.acc[j] += this.weights[j];
      if (this.acc[j] > this.acc[best]) best = j;
    }
    this.acc[best] -= 1;
    return this.names[best];
  }
}

/**
 * generate builds an archive of about `n` claims as a chain of contributions.
 * Same seed always yields the same archive.
 */
export function generate(n: number, seedOrOpts: number | GenerateOptions = 0x5eed): MockArchive {
  const opts: GenerateOptions = typeof seedOrOpts === 'number' ? { seed: seedOrOpts } : seedOrOpts;
  const seed = opts.seed ?? 0x5eed;
  const perContribution = Math.max(3, opts.claimsPerContribution ?? CLAIMS_PER_CONTRIBUTION);

  const t0 = performance.now();
  const rnd = mulberry32(seed);
  const minter = new IdMinter(rnd);
  const classes = new ClassCycle();
  const claims: MockClaim[] = [];
  const byClass: Record<string, number> = {};

  const contributors: number[] = [];
  const sources: number[] = [];
  const derivations: number[] = [];
  const entities: number[] = [];
  const entityPool = new Pool();
  /** Current head claim per branch — the chain each contribution extends. */
  const branchHeads = new Map<string, number>();

  let clock = Date.UTC(2024, 0, 1);
  let edgeCount = 0;
  let contributions = 0;
  let prevTable: number | undefined;

  const emit = (type: string, edges: MockEdge[], label: string, size?: number): number => {
    clock += 1 + ((rnd() * 9000) | 0);
    const claim: MockClaim = {
      id: minter.next(rnd),
      type,
      created_at: clock,
      label,
      contribution: contributions,
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

  for (let i = 0; i < CONTRIBUTORS; i++) {
    contributors.push(emit('contribution/contributor', [], `contributor-${i}`, 96));
  }

  /** Every claim names its contributor — the edge that makes contributors hubs. */
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

  /** contentClaim emits one ordinary claim and returns its index. */
  const contentClaim = (): number => {
    const kind = classes.next();

    if (kind === 'derivation' && sources.length > 0) {
      const edges: MockEdge[] = [contributorEdge()];
      const inputs = 1 + ((rnd() * 4) | 0);
      for (let k = 0; k < inputs; k++) {
        const from = rnd() < 0.6 || derivations.length === 0 ? sources : derivations;
        const ref = drawInput(from);
        if (ref !== undefined) edges.push({ type: 'derivation/input', reference: claims[ref].id });
      }
      const subtype = pick(NODE_CLASSES.derivation, rnd);
      const idx = emit(`derivation/${subtype}`, edges, `${subtype} ${claims.length}`, 64 + ((rnd() * 4096) | 0));
      derivations.push(idx);
      return idx;
    }

    if (kind === 'entity' && derivations.length > 0) {
      const edges: MockEdge[] = [contributorEdge()];
      const inputs = 1 + ((rnd() * 3) | 0);
      for (let k = 0; k < inputs; k++) {
        const ref = drawInput(derivations);
        if (ref !== undefined) edges.push({ type: 'derivation/input', reference: claims[ref].id });
      }
      const subtype = pick(NODE_CLASSES.entity, rnd);
      const idx = emit(`entity/${subtype}`, edges, `${subtype} ${claims.length}`, 32 + ((rnd() * 512) | 0));
      entities.push(idx);
      entityPool.push(idx);
      return idx;
    }

    if (kind === 'relation' && entities.length > 1) {
      const edges: MockEdge[] = [contributorEdge()];
      const arity = rnd() < 0.9 ? 2 : 3;
      const seen = new Set<number>();
      for (let k = 0; k < arity; k++) {
        const ref = entityPool.drawWindowed(rnd, HUB_WINDOW, P_HUB_RECENT);
        if (seen.has(ref)) continue;
        seen.add(ref);
        entityPool.push(ref);
        edges.push({
          type: 'relation/member',
          reference: claims[ref].id,
          relation_direction: k === 0 ? 1 : -1,
        });
      }
      const support = drawInput(derivations);
      if (support !== undefined) edges.push({ type: 'derivation/input', reference: claims[support].id });
      const subtype = pick(NODE_CLASSES.relation, rnd);
      return emit(`relation/${subtype}`, edges, subtype, undefined);
    }

    // Sources are always possible, and are what the archive starts with.
    const subtype = pick(NODE_CLASSES.source, rnd);
    const idx = emit(
      `source/${subtype}`,
      [contributorEdge()],
      `${subtype} ${claims.length}`,
      512 + ((rnd() * 262144) | 0),
    );
    sources.push(idx);
    return idx;
  };

  // Each pass through this loop is one contribution: content, then the head that
  // consolidates it over the previous head, then the branch-table revision that
  // diffs the table before it. Those last two are the commit history.
  const contentPerContribution = Math.max(1, perContribution - 2);
  while (claims.length < n) {
    const bulk = rnd() < P_BULK;
    const mean = bulk ? contentPerContribution * BULK_FACTOR : contentPerContribution;
    const size = Math.max(1, Math.round(mean * (0.5 + rnd())));

    const added: number[] = [];
    for (let k = 0; k < size && claims.length < n; k++) added.push(contentClaim());

    // Branch selection: `main` takes most contributions, the rest share the tail.
    const branch = rnd() < 0.6 ? BRANCH_NAMES[0] : pick(BRANCH_NAMES, rnd);

    // The head consolidates the previous head (all of history) plus what is new.
    const headEdges: MockEdge[] = [contributorEdge()];
    const prevHead = branchHeads.get(branch);
    if (prevHead !== undefined) {
      headEdges.push({ type: 'contribution/head', reference: claims[prevHead].id });
    }
    for (const idx of added) {
      headEdges.push({ type: 'contribution/head', reference: claims[idx].id });
    }
    const head = emit('contribution/head', headEdges, `${branch}@${contributions}`, undefined);
    branchHeads.set(branch, head);

    // The branch table: a diff over its predecessor, naming the branch it moved.
    const tableEdges: MockEdge[] = [contributorEdge()];
    if (prevTable !== undefined) {
      tableEdges.push({ type: 'contribution/diff', reference: claims[prevTable].id });
    }
    tableEdges.push({ type: 'contribution/branch', reference: claims[head].id, name: branch });
    prevTable = emit('contribution/branches', tableEdges, `branches r${contributions}`, undefined);

    contributions++;
  }

  const branches: Record<string, ClaimId> = {};
  for (const [name, idx] of branchHeads) branches[name] = claims[idx].id;

  return {
    claims,
    head: claims[prevTable ?? claims.length - 1].id,
    branches,
    stats: {
      claims: claims.length,
      edges: edgeCount,
      byClass,
      contributions,
      generateMs: performance.now() - t0,
    },
  };
}
