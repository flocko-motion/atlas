/**
 * package: core / mock
 * type:    logic
 * job:     generate a deterministic Ranke-Graph shaped archive
 * limits:  shapes claims; serves no requests (-> core/data/source)
 *
 * Built contribution by contribution, which is what makes a Ranke-Graph tall: each mints
 * a head and a branch-table revision, so 100k claims sit ~10k contributions deep — a
 * ribbon, not a ball.
 *
 * Every claim is built by the library's `claim_builder`, so a generated archive holds real
 * claims: content-addressed ids over the canonical encoding, the edge rules enforced, and the
 * same type a read from an instance returns. It buys a generator that cannot drift from the
 * model it stands in for, and it costs ~0.12 ms a claim — an encode and a hash per record, which
 * is what a claim costs (see README.md for where that figure went between library releases).
 * Claims are identity-signed (§5.7): no key is held here, so the contributors publish none.
 */

import { hashContent, newClaim } from '@flocko-motion/ranke';
import type { Claim, Contributor, EdgeInput } from '@flocko-motion/ranke';
import {
  EdgeTypeBranch,
  EdgeTypeHead,
  NodeClassDerivation,
  NodeClassEntity,
  NodeClassRelation,
  NodeClassSource,
  NodeTypeBranches,
  NodeTypeContributor,
  NodeTypeHead,
} from '@flocko-motion/ranke';
import { labelOf } from '../claims.ts';
import type { DrawnClaim } from '../claims.ts';
import { NAMES, OCCASIONS, ORGS, PLACES, SUBTYPES, THINGS } from './model.ts';
import type { ContentClass, MockArchive } from './model.ts';

/** Content-class mix within a contribution, in claims per 1000. */
const MIX: Record<ContentClass, number> = {
  [NodeClassSource]: 205,
  [NodeClassDerivation]: 450,
  [NodeClassEntity]: 205,
  [NodeClassRelation]: 140,
};

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
/** The encoding declared for generated content — text, so a reader knows what it would be. */
const CONTENT_ENCODING = 'text/plain';
/** How many content addresses the generator draws from; see ContentPool. */
const CONTENT_ADDRESSES = 256;

export interface GenerateOptions {
  seed?: number;
  /** Average claims per contribution, head and table included. */
  claimsPerContribution?: number;
}

/** A contributor publishing no key: what identity-signed claims are attributed to (§5.7). */
const KEYLESS = new Uint8Array(0);

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

/**
 * ContentPool hands out content addresses. An external address is H(bytes), so minting one
 * per claim would add a hash to a record that already costs two; a small pool cycled through
 * instead says several claims cite the same blob — which is what content addressing means,
 * not a shortcut around it. Cycled rather than drawn, so the archive a seed produces does
 * not depend on how many claims carry content.
 */
class ContentPool {
  private addresses: string[] = [];
  private n = 0;

  constructor() {
    for (let i = 0; i < CONTENT_ADDRESSES; i++) {
      this.addresses.push(hashContent(new TextEncoder().encode(`mock-content-${i}`)).toString());
    }
  }

  next(): string {
    return this.addresses[this.n++ % this.addresses.length];
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
  private names: ContentClass[];
  private weights: number[];

  constructor() {
    const entries = Object.entries(MIX) as [ContentClass, number][];
    this.names = entries.map(([k]) => k);
    const total = entries.reduce((s, [, w]) => s + w, 0);
    this.weights = entries.map(([, w]) => w / total);
    this.acc = new Float64Array(entries.length);
  }

  next(): ContentClass {
    let best = 0;
    for (let j = 0; j < this.weights.length; j++) {
      this.acc[j] += this.weights[j];
      if (this.acc[j] > this.acc[best]) best = j;
    }
    this.acc[best] -= 1;
    return this.names[best];
  }
}

/** What one generated claim states, before the builder turns it into a claim. */
interface Emit {
  type: string;
  /** Absent only on a root contributor claim, which has nothing to attribute to (§4.3). */
  contributor?: Contributor;
  /** Every edge names the claim it cites, so the builder can carry what travels with it. */
  edges?: EdgeInput[];
  /** External content of this size, for the classes whose content is a blob. */
  size?: number;
  /** Inline content: what this claim says, carried in the record and committed to by its id. */
  text?: string;
  /**
   * A caption for a claim carrying neither — the structural ones, whose content is their edges.
   * The kind above it is the claim's own subtype, so no caller repeats it (-> core/claims).
   */
  detail?: string;
  /** Makes the claim a diff over the revision it names, as a branch table is. */
  diffOf?: string;
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
  const classes = new ClassCycle();
  const addresses = new ContentPool();
  const claims: DrawnClaim[] = [];
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

  const claimAt = (index: number): Claim => claims[index].claim;

  /**
   * content declares what a claim carries. Short readable things are inline — the id commits to
   * the bytes, so a reader who has the claim has what it says. A source is a document, so it
   * stays external: an address and a size, with the blob in the Universe.
   */
  const content = (input: Emit) => {
    if (input.text !== undefined) {
      const bytes = new TextEncoder().encode(input.text);
      return { content: { kind: 'inline' as const, bytes, size: bytes.length, encoding: CONTENT_ENCODING } };
    }
    if (input.size === undefined) return {};
    return {
      content: {
        kind: 'external' as const,
        hash: addresses.next(),
        size: input.size,
        encoding: CONTENT_ENCODING,
      },
    };
  };

  /** says invents what a claim of this class states, from the mock's vocabulary. */
  const says = (cls: string, subtype: string): string => {
    const of = <T,>(list: readonly T[]): T => pick(list, rnd);
    if (cls === NodeClassEntity) {
      if (subtype === 'person' || subtype === 'role') return of(NAMES);
      if (subtype === 'organization') return of(ORGS);
      if (subtype === 'place') return of(PLACES);
      if (subtype === 'event') return `${of(PLACES)}, ${of(OCCASIONS)}`;
      return `${of(THINGS)} from ${of(PLACES)}`;
    }
    if (cls === NodeClassRelation) return `${of(NAMES)} — ${of(NAMES)}`;
    // A derivation says what it was drawn from; its own kind is the line above it.
    return `${of(THINGS)}, ${of(PLACES)}`;
  };

  const emit = (input: Emit): number => {
    clock += 1 + ((rnd() * 9000) | 0);
    const edges = input.edges ?? [];
    const { claim } = newClaim({
      type: input.type,
      contributor: input.contributor,
      createdAt: new Date(clock),
      height: heightOver(input.contributor, edges),
      ...content(input),
      edges,
      ...(input.diffOf === undefined ? {} : { diffOf: input.diffOf }),
    });
    claims.push({
      claim,
      contribution: contributions,
      // Set once the contribution's branch is chosen. The contributors emitted before any
      // contribution keep '', meaning every branch: each claim references one, so a scoped
      // read has to carry them whatever it is scoped to.
      branch: '',
      // Captioned the way a read's claims are, so a mock archive and a real one draw alike: the
      // library reads the inline bytes back out, and a structural claim falls back to its detail.
      label: input.text === undefined ? `${claim.typeSub}\n${input.detail ?? ''}` : labelOf(claim),
    });
    byClass[claim.typeClass] = (byClass[claim.typeClass] ?? 0) + 1;
    // The builder adds the contributor edge every attributed claim carries, so the count
    // comes off the built claim rather than off what was asked for.
    edgeCount += claim.edges.length;
    return claims.length - 1;
  };

  for (let i = 0; i < CONTRIBUTORS; i++) {
    contributors.push(emit({ type: NodeTypeContributor, detail: `#${i}` }));
  }

  /** Every claim names its contributor — the attribution that makes contributors hubs. */
  const contributorFor = (): Contributor => ({
    id: claimAt(contributors[(rnd() * contributors.length) | 0]).id,
    pubkey: KEYLESS,
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

  /** input is one `derivation/input` edge, carrying the claim it cites. */
  const input = (index: number): EdgeInput => ({
    reference: claimAt(index).id,
    type: 'derivation/input',
    referenced: claimAt(index),
  });

  /** contentClaim emits one ordinary claim and returns its index. */
  const contentClaim = (): number => {
    const kind = classes.next();

    if (kind === NodeClassDerivation && sources.length > 0) {
      // The contributor is drawn first, before anything the edges draw: one random stream
      // decides the whole archive, so where a draw happens is part of what a seed means.
      const contributor = contributorFor();
      const edges: EdgeInput[] = [];
      const inputs = 1 + ((rnd() * 4) | 0);
      for (let k = 0; k < inputs; k++) {
        const from = rnd() < 0.6 || derivations.length === 0 ? sources : derivations;
        const ref = drawInput(from);
        if (ref !== undefined) edges.push(input(ref));
      }
      const subtype = pick(SUBTYPES[NodeClassDerivation], rnd);
      const idx = emit({
        type: `${NodeClassDerivation}/${subtype}`,
        contributor,
        edges,
        text: says(NodeClassDerivation, subtype),
      });
      derivations.push(idx);
      return idx;
    }

    if (kind === NodeClassEntity && derivations.length > 0) {
      const contributor = contributorFor();
      const edges: EdgeInput[] = [];
      const inputs = 1 + ((rnd() * 3) | 0);
      for (let k = 0; k < inputs; k++) {
        const ref = drawInput(derivations);
        if (ref !== undefined) edges.push(input(ref));
      }
      const subtype = pick(SUBTYPES[NodeClassEntity], rnd);
      const idx = emit({
        type: `${NodeClassEntity}/${subtype}`,
        contributor,
        edges,
        text: says(NodeClassEntity, subtype),
      });
      entities.push(idx);
      entityPool.push(idx);
      return idx;
    }

    if (kind === NodeClassRelation && entities.length > 1) {
      const contributor = contributorFor();
      const edges: EdgeInput[] = [];
      const arity = rnd() < 0.9 ? 2 : 3;
      const seen = new Set<number>();
      for (let k = 0; k < arity; k++) {
        const ref = entityPool.drawWindowed(rnd, HUB_WINDOW, P_HUB_RECENT);
        if (seen.has(ref)) continue;
        seen.add(ref);
        entityPool.push(ref);
        edges.push({
          reference: claimAt(ref).id,
          type: 'relation/member',
          relationDirection: k === 0 ? 1 : -1,
          referenced: claimAt(ref),
        });
      }
      // A relation rests on stated provenance like any other claim (§3.5), so the support
      // edge is not optional: with nothing to cite this is a source instead.
      const support = drawInput(derivations);
      if (support !== undefined) {
        edges.push(input(support));
        const subtype = pick(SUBTYPES[NodeClassRelation], rnd);
        return emit({
          type: `${NodeClassRelation}/${subtype}`,
          contributor,
          edges,
          text: says(NodeClassRelation, subtype),
        });
      }
    }

    // Sources are always possible, and are what the archive starts with.
    const subtype = pick(SUBTYPES[NodeClassSource], rnd);
    const idx = emit({
      type: `${NodeClassSource}/${subtype}`,
      contributor: contributorFor(),
      // A source is a document, so its bytes live in the Universe and the claim carries an
      // address; the caption is what a catalogue entry for it would say.
      size: 512 + ((rnd() * 262144) | 0),
      detail: `${pick(THINGS, rnd)}, ${pick(PLACES, rnd)}`,
    });
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

    // Branch selection: `main` takes most contributions, the rest share the tail. Chosen
    // after the content is emitted, so the branch is stamped rather than threaded — which
    // keeps the random sequence, and so the archive a seed produces, unchanged.
    const branch = rnd() < 0.6 ? BRANCH_NAMES[0] : pick(BRANCH_NAMES, rnd);
    for (const idx of added) claims[idx].branch = branch;

    // The head consolidates the previous head (all of history) plus what is new.
    const headContributor = contributorFor();
    const headEdges: EdgeInput[] = [];
    const prevHead = branchHeads.get(branch);
    if (prevHead !== undefined) {
      headEdges.push({
        reference: claimAt(prevHead).id,
        type: EdgeTypeHead,
        referenced: claimAt(prevHead),
      });
    }
    for (const idx of added) {
      headEdges.push({ reference: claimAt(idx).id, type: EdgeTypeHead, referenced: claimAt(idx) });
    }
    const head = emit({
      type: NodeTypeHead,
      contributor: headContributor,
      edges: headEdges,
      detail: `${branch}@${contributions}`,
    });
    claims[head].branch = branch;
    branchHeads.set(branch, head);

    // The branch table: a diff over its predecessor, naming the branch it moved. The name
    // is what the overlay is keyed by, which is why a diff's edges all carry one.
    const tableContributor = contributorFor();
    prevTable = emit({
      type: NodeTypeBranches,
      contributor: tableContributor,
      edges: [
        {
          reference: claimAt(head).id,
          type: EdgeTypeBranch,
          fields: { name: branch },
          referenced: claimAt(head),
        },
      ],
      detail: `r${contributions}`,
      ...(prevTable === undefined ? {} : { diffOf: claimAt(prevTable).id }),
    });
    // The branch table indexes every branch, so it is shared rather than owned by one.
    claims[prevTable].branch = '';

    contributions++;
  }

  const branches: Record<string, string> = {};
  for (const [name, idx] of branchHeads) branches[name] = claimAt(idx).id;

  return {
    claims,
    head: claimAt(prevTable ?? claims.length - 1).id,
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

/**
 * heightOver is the generation number of a claim: 1 + the highest generation it reaches, over
 * these edges *and* the contributor edge the builder adds (§4.1). A root contributor claim
 * reaches nothing and is 0 — which is also why a source claim, citing nothing but its
 * contributor, is 1 rather than 0.
 *
 * The library's `heightOf` says this over *arguments*, which is the wrong shape here: a bulk
 * contribution's head cites tens of thousands of claims, and spreading those into a call is a
 * stack overflow waiting for the granularity sweep to find it.
 */
function heightOver(contributor: Contributor | undefined, edges: readonly EdgeInput[]): number {
  if (contributor === undefined) return 0;
  let highest = 0; // the contributor claim itself, which is an initial node
  for (const edge of edges) {
    const height = edge.referenced?.height ?? 0;
    if (height > highest) highest = height;
  }
  return highest + 1;
}
