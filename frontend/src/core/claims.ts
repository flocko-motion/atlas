/**
 * package: core / claims
 * type:    data
 * job:     hold a claim the way a drawing needs it — the library's Claim, plus what it lacks
 * limits:  the pairing and the label; the claim itself is the library's (-> @flocko-motion/ranke)
 *
 * The claim is `@flocko-motion/ranke`'s `Claim` and nothing here restates it. What sits
 * beside it is drawing state: an index a layout sorts on, a branch a generated archive
 * stamped, and a caption. Those describe a picture rather than a claim, which is why they
 * sit *beside* the type instead of inside a copy of it.
 */

import { contentEncoding, inlineBytes } from '@flocko-motion/ranke';
import type { Claim } from '@flocko-motion/ranke';

/**
 * DrawnClaim is one claim as a view holds it. Both sources answer in this shape, so a page
 * of generated claims and a page read from an instance travel the same path.
 */
export interface DrawnClaim {
  claim: Claim;
  /**
   * Index of the contribution that added this claim. Not part of the ADT — an archive
   * expresses that order as the head chain — but the renderer needs it, because contribution
   * order is the axis a reader understands and it is *not* provenance depth.
   */
  contribution: number;
  /**
   * Branch whose contribution introduced this claim. Not part of the ADT either — an archive
   * answers "which branch" by closure, and that is the engine's work — but a generator
   * standing in for a server has to answer a scoped read somehow, and stamping it as it
   * generates is the one way that costs nothing.
   */
  branch: string;
  /** The caption a node carries. Derived from content where there is text to read. */
  label: string;
}

/**
 * A read from an instance carries no contribution index: the archive expresses that order as
 * the head chain, deriving it is a traversal, and no output field reports it. So claims
 * arrive at 0 — which the layered layout draws and the history layout cannot.
 */
export const contributionUnknown = 0;

/**
 * labelOf captions a claim: inline text reads as text, anything else gets its type and a
 * short id. Content the read left external is a hash and no caption, so it falls back too.
 */
export function labelOf(claim: Claim): string {
  const short = `${claim.type} ${claim.id.slice(0, 8)}`;
  const bytes = inlineBytes(claim.content);
  if (!bytes || !contentEncoding(claim.content).startsWith('text/')) return short;
  const text = new TextDecoder().decode(bytes).trim();
  return text.length > 0 ? text.slice(0, 80) : short;
}

/** drawn pairs a claim read from an instance with the drawing state such a read cannot carry. */
export function drawn(claim: Claim): DrawnClaim {
  return { claim, contribution: contributionUnknown, branch: '', label: labelOf(claim) };
}
