/**
 * package: core / tests
 * type:    test
 * job:     pin that the model and the contract come from their libraries, not from here
 * limits:  what this repo may declare; what the types mean is the library's to test
 *
 * The three rules the graph-explorer capability states: a claim is the ADT library's type, the
 * globs are its matcher, and no route or wire field name is written out here. The last one is
 * a scan, because that is the only form that keeps a deleted duplicate from growing back.
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { DirectedGraph } from 'graphology';

import { addClaims } from './graph/build.ts';
import { claimsFromBody } from './data/source.ts';
import { contentOf, forgetContent } from './content.ts';
import { generate } from './mock/generate.ts';

/** oneRecord is a query's answer holding a single claim, framed as RFC 7464 asks. */
function oneRecord(): Response {
  const claim = {
    id: 'bciqdlnrhbcnkalcqxrpxpmroin6iu5w6dgfjqoemvxlvvhtwepbe6ma',
    type: 'entity/person',
    created_at: '2024-03-04T05:06:07.000000000Z',
    height: 4,
    edges: [{ type: 'derivation/input', reference: 'bciqfoo' }],
  };
  const body = `\u001e${JSON.stringify(claim)}\n`;
  return { body: null, text: async () => body } as unknown as Response;
}

// 7.1 — one type, one path: a decoded claim and a generated one merge into the same graph
// through the same function, and the node each produces carries the same attributes.
test('a decoded claim and a generated claim are one type on one path', async () => {
  const fromWire = await claimsFromBody(oneRecord());
  const fromGenerator = generate(60, { seed: 11, claimsPerContribution: 6 }).claims;
  assert.equal(fromWire.length, 1);
  assert.ok(fromGenerator.length > 1);

  for (const drawn of [fromWire[0], fromGenerator[0]]) {
    // The library's Claim, not a local restatement: the type arrives already split, the
    // timestamp twice over, the generation number derived, and content as a declaration.
    assert.equal(typeof drawn.claim.typeClass, 'string');
    assert.equal(typeof drawn.claim.typeSub, 'string');
    assert.equal(drawn.claim.type, `${drawn.claim.typeClass}/${drawn.claim.typeSub}`);
    assert.equal(typeof drawn.claim.createdAt, 'string');
    assert.equal(typeof drawn.claim.createdAtMs, 'number');
    assert.equal(typeof drawn.claim.height, 'number');
    assert.ok(drawn.claim.content.kind === 'none' || 'size' in drawn.claim.content);
  }

  const graph = new DirectedGraph({ allowSelfLoops: false });
  const merged = addClaims(graph, [...fromWire, ...fromGenerator]);
  assert.equal(merged.addedNodes, 1 + fromGenerator.length);
  assert.deepEqual(
    Object.keys(graph.getNodeAttributes(fromWire[0].claim.id)).sort(),
    Object.keys(graph.getNodeAttributes(fromGenerator[0].claim.id)).sort(),
    'the two sources produce differently shaped nodes',
  );
});

// A capped read may serve a prefix of a body (R-QCONTENT), which kind and bytes alone look
// no different from whole content. Only whole content enters the cache — a cached prefix
// would be served as the content wherever the id is next looked up — while the declared
// size stays on the node, so the detail pane still knows what exists.
test('partial inline content is not cached; whole content is', async () => {
  forgetContent();
  const record = (id: string, size: number) =>
    `\u001e${JSON.stringify({
      id,
      type: 'source/note',
      created_at: '2024-03-04T05:06:07.000000000Z',
      height: 0,
      content: 'aGk=', // "hi", 2 bytes of however many `size` declares
      content_size: size,
      encoding: 'text/plain',
    })}\n`;
  const body = record('bciqwhole', 2) + record('bciqcut', 10);
  const claims = await claimsFromBody({ body: null, text: async () => body } as unknown as Response);

  const graph = new DirectedGraph({ allowSelfLoops: false });
  addClaims(graph, claims);
  assert.deepEqual(contentOf('bciqwhole'), new Uint8Array([104, 105]));
  assert.equal(contentOf('bciqcut'), null, 'a prefix was cached as the content');
  assert.equal(graph.getNodeAttribute('bciqcut', 'contentSize'), 10, 'the declared size was lost');
});

// 7.3 — the edge filter is the library's matcher, so it follows the contract's glob rules
// rather than a prefix test that would call `contribution/heading` a head.
test('edge filtering follows the contract globs, exclusions included', () => {
  const archive = generate(200, { seed: 3, claimsPerContribution: 8 });
  const edgesWith = (drop: string[]): number => {
    const graph = new DirectedGraph({ allowSelfLoops: false });
    return addClaims(graph, archive.claims, { attrs: 'lean', dropEdgeTypes: drop }).addedEdges;
  };

  const all = edgesWith([]);
  const noContribution = edgesWith(['contribution/*']);
  const noContributor = edgesWith(['contribution/contributor']);
  // A class glob drops more than one exact type does, and both drop something.
  assert.ok(noContribution < noContributor, 'a class glob dropped no more than one type');
  assert.ok(noContributor < all, 'naming an edge type dropped nothing');

  // A leading `-` re-admits what the glob before it named: heads survive, the rest do not.
  const keepingHeads = edgesWith(['contribution/*', '-contribution/head']);
  assert.ok(
    keepingHeads > noContribution,
    'the exclusion kept nothing, so a leading - was read as part of a name',
  );
  assert.ok(keepingHeads < noContributor, 'the exclusion kept more than the heads');
});

// 7.4 — the check that keeps the deleted duplication from growing back. Route paths and the
// wire's field names belong to the generated client and to the codec; a literal here is a
// second copy of one of them. Tests are exempt: a fixture states the wire on purpose.
test('no source file spells out a route path or a wire field name', () => {
  const offences: string[] = [];
  for (const file of sources(join(import.meta.dirname, '..'))) {
    const code = withoutComments(readFileSync(file, 'utf8'));
    for (const [what, pattern] of Object.entries(FORBIDDEN)) {
      const found = code.match(pattern);
      if (found) offences.push(`${file.replace(/.*\/src\//, 'src/')}: ${what} ${found[0]}`);
    }
  }
  assert.deepEqual(offences, [], `a contract detail is written out by hand:\n${offences.join('\n')}`);
});

/**
 * FORBIDDEN is what the explorer must not spell out: a path naming one of the contract's
 * collections, and the snake_case field names of the JSON projection. A graphology attribute
 * of a similar name is fine — those are this repo's own, and are deliberately spelled
 * differently (`claimType`, never `type`).
 */
const FORBIDDEN: Record<string, RegExp> = {
  'a route path': /['"`]\/(branches|archive|universe|query|contribute|health|system)\b/,
  'a wire field name': /['"`](created_at|content_size|content_hash|relation_direction)['"`]/,
  'a report field name': /['"`](started_at|elapsed_ns|at_ns|duration_ns)['"`]/,
};

/** sources lists the explorer's own TypeScript: no generated file, and no test fixture. */
function sources(root: string): string[] {
  const out: string[] = [];
  for (const entry of readdirSync(root, { withFileTypes: true })) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      out.push(...sources(path));
    } else if (/\.tsx?$/.test(entry.name) && !/\.gen\.ts$|\.test\.tsx?$/.test(entry.name)) {
      out.push(path);
    }
  }
  return out;
}

/**
 * withoutComments strips what prose may say freely — a comment naming `GET /branches` is
 * documentation, not a request. Crude on purpose: mangling a URL inside a string is harmless
 * here, since what is left cannot become a false accusation.
 */
function withoutComments(code: string): string {
  return code.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/.*$/gm, '');
}
