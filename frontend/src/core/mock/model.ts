// package: core / mock
// type:    data
// job:     the invented vocabulary the mock claims are built from, and the archive shape
// limits:  types only; the generator that uses them is generate.ts
//
// The ADT is the library's: claims, edges, the node classes and the content declaration all
// come from `@rankegraph/ranke`. What is left here is what a generator invents — plausible
// subtypes — and the archive a generation returns, which is a mock's answer to a read rather
// than a thing the ADT names.

import { NodeClassDerivation, NodeClassEntity, NodeClassRelation, NodeClassSource } from '@rankegraph/ranke';
import type { NodeClass } from '@rankegraph/ranke';
import type { DrawnClaim } from '../claims.ts';

/**
 * SUBTYPES are plausible second-level names per node class — open vocabulary, which is why a
 * mock may invent them: the ADT closes the classes (`NodeClasses`) and the contribution
 * subtypes, and leaves the rest to whoever contributes.
 *
 * The `contribution/*` claims a generation mints are not here: their subtypes are the
 * library's constants, being structure rather than content.
 */
export const SUBTYPES: Record<ContentClass, readonly string[]> = {
  [NodeClassSource]: ['email', 'photo', 'letter', 'record', 'transcript', 'dataset'],
  [NodeClassDerivation]: ['extraction', 'classification', 'summary', 'ocr', 'resolution'],
  [NodeClassEntity]: ['person', 'organization', 'place', 'thing', 'work', 'event', 'role'],
  [NodeClassRelation]: ['knows', 'works_for', 'located_in', 'part_of', 'authored', 'family'],
};

/**
 * The words a mock's inline content is drawn from. A generated archive is read by eye as well as
 * measured, and a canvas of claims that say nothing exercises the paint but tells a reader
 * nothing about what an archive looks like when it is full.
 */
export const NAMES = [
  'Anna Weber', 'Piet de Vries', 'Johann Kessler', 'Marta Lindqvist', 'Elias Brandt',
  'Sofie Haan', 'Karel Mertens', 'Ruth Bergmann',
] as const;
export const PLACES = [
  'Bremen', 'Rotterdam', 'Gdańsk', 'Bergen', 'Antwerp', 'Riga', 'Hull', 'Malmö',
] as const;
export const ORGS = [
  'Hansa Werke', 'Nordreederei', 'Kessler & Sohn', 'Baltic Salvage', 'Weser Docks',
] as const;
export const THINGS = [
  'ledger', 'cargo manifest', 'crate', 'engine plate', 'harbour chart', 'crew list',
] as const;
export const OCCASIONS = [
  'the 1897 refit', 'the spring auction', 'the harbour fire', 'the second survey',
] as const;

/** ContentClass is a node class a contribution's *content* is drawn from — every one but structure. */
export type ContentClass = Exclude<NodeClass, 'contribution'>;

/** A generated archive: the claims plus the branch-table head that indexes them. */
export interface MockArchive {
  claims: DrawnClaim[];
  /** Id of the `contribution/branches` claim — the archive head `k`. */
  head: string;
  /** Branch name → head claim id, as the branch table records them. */
  branches: Record<string, string>;
  stats: {
    claims: number;
    edges: number;
    byClass: Record<string, number>;
    /** Contributions the archive was built from — the length of its history. */
    contributions: number;
    generateMs: number;
  };
}
