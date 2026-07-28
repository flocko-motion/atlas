/**
 * Ranke-Graph shaped mock claims — the ADT vocabulary this spike renders.
 *
 * Mirrors paper 01 (`docs/papers/01-ranke-graph/ranke-graph.typ`, §Type Vocabulary):
 * five node classes, three edge classes, edges owned by the claim that references
 * out. Only what a renderer needs is modelled — content bytes are never produced,
 * just their sizes, because the spike measures topology and paint, not payloads.
 */

/** A claim id — content-addressed in the real ADT, synthetic but same-shaped here. */
export type ClaimId = string;

/** NODE_CLASSES are the five node classes of the ADT, with plausible subtypes. */
export const NODE_CLASSES = {
  source: ['email', 'photo', 'letter', 'record', 'transcript', 'dataset'],
  derivation: ['extraction', 'classification', 'summary', 'ocr', 'resolution'],
  entity: ['person', 'organization', 'place', 'thing', 'work', 'event', 'role'],
  relation: ['knows', 'works_for', 'located_in', 'part_of', 'authored', 'family'],
  contribution: ['contributor', 'head', 'branches'],
} as const;

/** An edge as it lives in a claim's `edges` set: a typed reference outward. */
export interface MockEdge {
  /** Edge type — `derivation/*`, `relation/*` or `contribution/*`. */
  type: string;
  /** The claim this edge points at. */
  reference: ClaimId;
  /** Branch name, set only on `contribution/branch` edges. */
  name?: string;
  /** `1` = from, `-1` = to; set only on `relation/*` edges. */
  relation_direction?: 1 | -1;
}

/** A claim: a node, its content metadata, and the edges created with it. */
export interface MockClaim {
  id: ClaimId;
  /** `class/subtype`, e.g. `source/email`. */
  type: string;
  /** UTC ms. Monotonic: never earlier than any claim it references. */
  created_at: number;
  encoding?: string;
  content_size?: number;
  /** A short display string a real explorer would derive from content. */
  label: string;
  edges: MockEdge[];
}

/** A generated archive: the claims plus the branch-table head that indexes them. */
export interface MockArchive {
  claims: MockClaim[];
  /** Id of the `contribution/branches` claim — the archive head `k`. */
  head: ClaimId;
  /** Branch name → head claim id, as the branch table records them. */
  branches: Record<string, ClaimId>;
  stats: {
    claims: number;
    edges: number;
    byClass: Record<string, number>;
    generateMs: number;
  };
}

/** classOf returns the class part of a `class/subtype` type string. */
export function classOf(type: string): string {
  const slash = type.indexOf('/');
  return slash === -1 ? type : type.slice(0, slash);
}
