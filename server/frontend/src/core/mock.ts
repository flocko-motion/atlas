/**
 * @layer core
 * @description Mock data generator for development and load testing.
 * @depends core/types/nodes
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

import type { Node, Edge } from './types/nodes';

/** Simple seeded PRNG (mulberry32). */
function createRng(seed: number): () => number {
  let s = seed | 0;
  return () => {
    s = (s + 0x6d2b79f5) | 0;
    let t = Math.imul(s ^ (s >>> 15), 1 | s);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function pick<T>(rng: () => number, arr: readonly T[]): T {
  return arr[Math.floor(rng() * arr.length)];
}

function uuid(rng: () => number): string {
  const hex = () =>
    Math.floor(rng() * 16)
      .toString(16);
  const seg = (n: number) => Array.from({ length: n }, hex).join('');
  return `${seg(8)}-${seg(4)}-4${seg(3)}-${pick(rng, ['8', '9', 'a', 'b'])}${seg(3)}-${seg(12)}`;
}

function randomDate(rng: () => number, from: Date, to: Date): Date {
  const t = from.getTime() + rng() * (to.getTime() - from.getTime());
  return new Date(t);
}

// Weighted toward recent dates (exponential)
function weightedRecentDate(rng: () => number, from: Date, to: Date): Date {
  const r = Math.pow(rng(), 0.5); // bias toward 1 = recent
  const t = from.getTime() + r * (to.getTime() - from.getTime());
  return new Date(t);
}

function sha256Fake(rng: () => number): string {
  return Array.from({ length: 64 }, () =>
    Math.floor(rng() * 16).toString(16)
  ).join('');
}

const L0_TYPES: [string, string][] = [
  ['source', 'conversation'],
  ['source', 'media'],
  ['source', 'record'],
  ['source', 'data'],
];

const L1_TYPES: [string, string][] = [
  ['classification', 'entity'],
  ['classification', 'topic'],
  ['observation', 'sentiment'],
  ['observation', 'intent'],
  ['observation', 'language'],
  ['summary', 'abstract'],
  ['summary', 'detail'],
  ['fact', 'claim'],
  ['fact', 'date'],
  ['fact', 'quantity'],
];

const L2_ENTITY_TYPES: [string, string][] = [
  ['entity', 'person'],
  ['entity', 'organization'],
  ['entity', 'place'],
];

const L2_RELATION_TYPES: [string, string][] = [
  ['relation', 'family'],
  ['relation', 'alias'],
  ['relation', 'has_role'],
];

const NAMES = [
  'Alice', 'Bob', 'Charlie', 'Diana', 'Eve', 'Frank', 'Grace', 'Hank',
  'Iris', 'Jack', 'Karen', 'Leo', 'Mona', 'Nick', 'Olivia', 'Paul',
];

const ORGS = [
  'Acme Corp', 'Globex', 'Initech', 'Umbrella', 'Cyberdyne', 'Stark Industries',
];

const PLACES = ['Berlin', 'Tokyo', 'New York', 'London', 'Paris', 'Sydney'];

const BLUR_VALUES = ['', 'P0D', 'P1D', 'P7D', 'P30D', 'P90D', 'P365D'];

function contentForNode(
  rng: () => number,
  level: number,
  contentClass: string,
  contentType: string,
): string {
  const name = pick(rng, NAMES);
  const name2 = pick(rng, NAMES);
  const org = pick(rng, ORGS);
  const place = pick(rng, PLACES);

  if (level === 0) {
    const templates: Record<string, string[]> = {
      conversation: [`Email from ${name} to ${name2}`, `Chat log: ${name} and ${name2}`, `Interview with ${name}`],
      media: [`Photo from ${place}`, `Voice memo by ${name}`, `Scan of document from ${org}`],
      record: [`Invoice #${Math.floor(rng() * 9000 + 1000)} from ${org}`, `Contract between ${name} and ${org}`],
      data: [`CSV export from ${org}`, `Database dump: ${place} office`, `Spreadsheet: ${name}'s finances`],
    };
    return pick(rng, templates[contentType] ?? [`Source: ${contentType}`]);
  }

  if (level === 1) {
    const templates: Record<string, string[]> = {
      entity: [`Identified: ${name}`, `Entity: ${org}`, `Location: ${place}`],
      topic: ['Topic: finance', 'Topic: family', 'Topic: employment', 'Topic: travel'],
      sentiment: ['Positive sentiment', 'Negative sentiment', 'Neutral tone'],
      intent: ['Request for information', 'Agreement', 'Complaint'],
      language: ['Language: English', 'Language: German', 'Language: Japanese'],
      abstract: [`Summary of conversation with ${name}`, `Overview: ${org} dealings`],
      detail: [`${name} mentioned meeting ${name2} in ${place}`, `${org} contract details`],
      claim: [`${name} worked at ${org}`, `${name} lived in ${place}`],
      date: [`Event date: ${2015 + Math.floor(rng() * 10)}`],
      quantity: [`Amount: $${Math.floor(rng() * 100000)}`],
    };
    return pick(rng, templates[contentType] ?? [`L1: ${contentClass}/${contentType}`]);
  }

  // L2
  if (contentClass === 'entity') {
    if (contentType === 'person') return `Person: ${name}`;
    if (contentType === 'organization') return `Org: ${org}`;
    if (contentType === 'place') return `Place: ${place}`;
  }
  const relLabels: Record<string, string[]> = {
    family: [`sister of`, `parent of`, `spouse of`, `child of`],
    alias: [`also known as`, `formerly known as`],
    has_role: [`CEO at ${org}`, `employee of ${org}`, `resident of ${place}`],
  };
  return pick(rng, relLabels[contentType] ?? [`relation: ${contentType}`]);
}

export function generateMockGraph(nodeCount: number): { nodes: Node[]; edges: Edge[] } {
  const rng = createRng(nodeCount);
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  const START = new Date('2015-01-01');
  const END = new Date('2026-04-01');

  // Determine counts per level
  const l0Count = Math.round(nodeCount * 0.4);
  const l1Count = Math.round(nodeCount * 0.4);
  const l2Count = nodeCount - l0Count - l1Count;
  const workerCount = Math.min(5, Math.max(3, Math.floor(nodeCount / 100)));

  // Create worker config nodes (special L0 nodes)
  const workerIds: string[] = [];
  for (let i = 0; i < workerCount; i++) {
    const id = uuid(rng);
    workerIds.push(id);
    const created = randomDate(rng, START, END);
    nodes.push({
      id,
      level: 0,
      contentClass: 'source',
      contentType: 'data',
      encodingClass: 'text',
      encodingFormat: 'application/json',
      contentSha256: sha256Fake(rng),
      contentLen: Math.floor(rng() * 2000 + 100),
      content: `Worker config #${i + 1}: ${pick(rng, ['entity-extractor', 'summarizer', 'classifier', 'relation-builder', 'fact-extractor'])}`,
      createdAt: created,
      artifactCreatedAt: created,
      artifactCreatedAtBlur: 'P0D',
      origin: 'system',
      originalName: `worker-config-${i + 1}.json`,
      validFrom: null,
      validFromBlur: '',
      validUntil: null,
      validUntilBlur: '',
      confidence: null,
    });
  }

  // L0 source nodes
  const l0Ids: string[] = [];
  for (let i = 0; i < l0Count; i++) {
    const id = uuid(rng);
    l0Ids.push(id);
    const [cc, ct] = pick(rng, L0_TYPES);
    const created = weightedRecentDate(rng, START, END);
    const blur = pick(rng, BLUR_VALUES);
    nodes.push({
      id,
      level: 0,
      contentClass: cc,
      contentType: ct,
      encodingClass: ct === 'media' ? 'binary' : 'text',
      encodingFormat: ct === 'media' ? 'image/jpeg' : 'text/plain',
      contentSha256: sha256Fake(rng),
      contentLen: Math.floor(rng() * 50000 + 100),
      content: contentForNode(rng, 0, cc, ct),
      createdAt: created,
      artifactCreatedAt: new Date(created.getTime() - rng() * 365 * 86400000),
      artifactCreatedAtBlur: blur,
      origin: pick(rng, ['upload', 'api', 'import', 'sync']),
      originalName: `${cc}-${ct}-${i}.${ct === 'media' ? 'jpg' : 'txt'}`,
      validFrom: null,
      validFromBlur: '',
      validUntil: null,
      validUntilBlur: '',
      confidence: null,
    });
  }

  // L1 cognition nodes + provenance edges
  const l1Ids: string[] = [];
  for (let i = 0; i < l1Count; i++) {
    const id = uuid(rng);
    l1Ids.push(id);
    const [cc, ct] = pick(rng, L1_TYPES);
    const created = weightedRecentDate(rng, START, END);
    const conf = rng() < 0.05 ? -(rng() * 0.5) : 0.5 + rng() * 0.5;
    nodes.push({
      id,
      level: 1,
      contentClass: cc,
      contentType: ct,
      encodingClass: 'text',
      encodingFormat: 'text/plain',
      contentSha256: sha256Fake(rng),
      contentLen: Math.floor(rng() * 500 + 10),
      content: contentForNode(rng, 1, cc, ct),
      createdAt: created,
      artifactCreatedAt: null,
      artifactCreatedAtBlur: '',
      origin: null,
      originalName: null,
      validFrom: null,
      validFromBlur: '',
      validUntil: null,
      validUntilBlur: '',
      confidence: Math.round(conf * 100) / 100,
    });

    // provenance/input: 1-3 L0 sources
    const inputCount = 1 + Math.floor(rng() * 3);
    for (let j = 0; j < inputCount && l0Ids.length > 0; j++) {
      edges.push({
        id: uuid(rng),
        sourceNodeId: id,
        targetNodeId: pick(rng, l0Ids),
        type: 'provenance/input',
        confidence: null,
        runId: uuid(rng),
        createdAt: created,
      });
    }

    // provenance/worker: 1 worker config
    edges.push({
      id: uuid(rng),
      sourceNodeId: id,
      targetNodeId: pick(rng, workerIds),
      type: 'provenance/worker',
      confidence: null,
      runId: uuid(rng),
      createdAt: created,
    });
  }

  // L2 nodes + provenance + semantic edges
  const l2EntityIds: string[] = [];
  const entityCount = Math.round(l2Count * 0.6);
  const relationCount = l2Count - entityCount;

  // L2 entities
  for (let i = 0; i < entityCount; i++) {
    const id = uuid(rng);
    l2EntityIds.push(id);
    const [cc, ct] = pick(rng, L2_ENTITY_TYPES);
    const created = weightedRecentDate(rng, START, END);
    const vFrom = rng() < 0.6 ? randomDate(rng, START, created) : null;
    const vUntil = vFrom && rng() < 0.4 ? randomDate(rng, vFrom, END) : null;
    nodes.push({
      id,
      level: 2,
      contentClass: cc,
      contentType: ct,
      encodingClass: 'text',
      encodingFormat: 'text/plain',
      contentSha256: sha256Fake(rng),
      contentLen: Math.floor(rng() * 200 + 5),
      content: contentForNode(rng, 2, cc, ct),
      createdAt: created,
      artifactCreatedAt: null,
      artifactCreatedAtBlur: '',
      origin: null,
      originalName: null,
      validFrom: vFrom,
      validFromBlur: vFrom ? pick(rng, BLUR_VALUES) : '',
      validUntil: vUntil,
      validUntilBlur: vUntil ? pick(rng, BLUR_VALUES) : '',
      confidence: Math.round((0.6 + rng() * 0.4) * 100) / 100,
    });

    // provenance: L2 entity -> L1 facts
    if (l1Ids.length > 0) {
      const pCount = 1 + Math.floor(rng() * 2);
      for (let j = 0; j < pCount; j++) {
        edges.push({
          id: uuid(rng),
          sourceNodeId: id,
          targetNodeId: pick(rng, l1Ids),
          type: 'provenance/input',
          confidence: null,
          runId: uuid(rng),
          createdAt: created,
        });
      }
      edges.push({
        id: uuid(rng),
        sourceNodeId: id,
        targetNodeId: pick(rng, workerIds),
        type: 'provenance/worker',
        confidence: null,
        runId: uuid(rng),
        createdAt: created,
      });
    }
  }

  // L2 relations
  for (let i = 0; i < relationCount; i++) {
    const id = uuid(rng);
    const [cc, ct] = pick(rng, L2_RELATION_TYPES);
    const created = weightedRecentDate(rng, START, END);
    const vFrom = rng() < 0.7 ? randomDate(rng, START, created) : null;
    const vUntil = vFrom && rng() < 0.3 ? randomDate(rng, vFrom, END) : null;
    const conf = rng() < 0.08 ? -(rng() * 0.3) : 0.6 + rng() * 0.4;
    nodes.push({
      id,
      level: 2,
      contentClass: cc,
      contentType: ct,
      encodingClass: 'text',
      encodingFormat: 'text/plain',
      contentSha256: sha256Fake(rng),
      contentLen: Math.floor(rng() * 100 + 5),
      content: contentForNode(rng, 2, cc, ct),
      createdAt: created,
      artifactCreatedAt: null,
      artifactCreatedAtBlur: '',
      origin: null,
      originalName: null,
      validFrom: vFrom,
      validFromBlur: vFrom ? pick(rng, BLUR_VALUES) : '',
      validUntil: vUntil,
      validUntilBlur: vUntil ? pick(rng, BLUR_VALUES) : '',
      confidence: Math.round(conf * 100) / 100,
    });

    // semantic edges: 1-3 head, 1-3 tail to entity nodes
    if (l2EntityIds.length >= 2) {
      const headCount = 1 + Math.floor(rng() * 3);
      for (let j = 0; j < headCount; j++) {
        edges.push({
          id: uuid(rng),
          sourceNodeId: id,
          targetNodeId: pick(rng, l2EntityIds),
          type: 'relation/head',
          confidence: Math.round(conf * 100) / 100,
          runId: null,
          createdAt: created,
        });
      }
      const tailCount = 1 + Math.floor(rng() * 3);
      for (let j = 0; j < tailCount; j++) {
        edges.push({
          id: uuid(rng),
          sourceNodeId: id,
          targetNodeId: pick(rng, l2EntityIds),
          type: 'relation/tail',
          confidence: Math.round(conf * 100) / 100,
          runId: null,
          createdAt: created,
        });
      }
    }

    // provenance
    if (l1Ids.length > 0) {
      edges.push({
        id: uuid(rng),
        sourceNodeId: id,
        targetNodeId: pick(rng, l1Ids),
        type: 'provenance/input',
        confidence: null,
        runId: uuid(rng),
        createdAt: created,
      });
      edges.push({
        id: uuid(rng),
        sourceNodeId: id,
        targetNodeId: pick(rng, workerIds),
        type: 'provenance/worker',
        confidence: null,
        runId: uuid(rng),
        createdAt: created,
      });
    }
  }

  return { nodes, edges };
}
