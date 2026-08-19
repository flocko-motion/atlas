/**
 * package: core / claims
 * type:    data
 * job:     hold a claim the way a drawing needs it — the library's Claim, plus what it lacks
 * limits:  the pairing and the label; the claim itself is the library's (-> @flocko-motion/ranke)
 *
 * The library's `Claim` and nothing here restates it. Beside it sits drawing state — an index a
 * layout sorts on, a branch a generator stamped, a caption — which describes a picture rather
 * than a claim, and so sits beside the type rather than inside a copy of it.
 */

import type { Claim } from '@flocko-motion/ranke';

/** DrawnClaim is one claim as a view holds it — the shape both sources answer in. */
export interface DrawnClaim {
  claim: Claim;
  /**
   * Index of the contribution that added this claim. Not the ADT's — an archive expresses that
   * order as the head chain — but it is the axis a reader understands, and not provenance depth.
   */
  contribution: number;
  /**
   * Branch whose contribution introduced this claim. An archive answers this by closure; a
   * generator standing in for one has to answer a scoped read, and stamping it costs nothing.
   */
  branch: string;
  /** The caption a node carries: the claim's subtype. */
  label: string;
}

/**
 * A read carries no contribution index — deriving it is a traversal, and no output field reports
 * it — so claims arrive at 0, which the layered layout draws and the history layout cannot.
 */
export const contributionUnknown = 0;

/**
 * labelOf captions a claim by its subtype alone — a canvas of hashes says only that the claims
 * differ, which the dots already say. The class is left out, since it is the colour of the dot
 * beside it. The claim's content belongs to the detail pane, not the canvas: quoting it in the
 * caption once put a claim's bytes on a thousand-node view at once, which is more than a graph
 * is for.
 */
export function labelOf(claim: Claim): string {
  return claim.typeSub;
}

/** shortId names a claim by the end of its id: every id here shares the prefix. */
export function shortId(id: string): string {
  return id.length > 8 ? `…${id.slice(-8)}` : id;
}

/** drawn pairs a claim read from an instance with the drawing state such a read cannot carry. */
export function drawn(claim: Claim): DrawnClaim {
  return { claim, contribution: contributionUnknown, branch: '', label: labelOf(claim) };
}
