/**
 * @layer core
 * @description Typed API client wrapper. All API calls go through here.
 * @depends core/types/nodes
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

import type { Node, Edge } from './types/nodes';

const BASE = '';  // same origin — schemaf proxies

interface ApiNodeResponse {
  id: string;
  level: number;
  content_class: string;
  content_type: string;
  encoding_class: string;
  encoding_format: string;
  content_sha256: string;
  content_len: number;
  title?: string | null;
  content?: string | null;
  created_at: string;
  artifact_created_at?: string | null;
  artifact_created_at_blur?: string | null;
  origin?: string | null;
  original_name?: string | null;
  valid_from?: string | null;
  valid_from_blur?: string | null;
  valid_until?: string | null;
  valid_until_blur?: string | null;
  confidence?: number | null;
}

interface ApiEdgeResponse {
  id: string;
  source_node_id: string;
  target_node_id: string;
  type: string;
  confidence?: number | null;
  run_id?: string | null;
  created_at: string;
}

function parseNode(raw: ApiNodeResponse): Node {
  return {
    id: raw.id,
    level: raw.level as 0 | 1 | 2,
    contentClass: raw.content_class,
    contentType: raw.content_type,
    encodingClass: raw.encoding_class,
    encodingFormat: raw.encoding_format,
    contentSha256: raw.content_sha256,
    contentLen: raw.content_len,
    title: raw.title ?? null,
    content: raw.content ?? null,
    createdAt: new Date(raw.created_at),
    artifactCreatedAt: raw.artifact_created_at ? new Date(raw.artifact_created_at) : null,
    artifactCreatedAtBlur: raw.artifact_created_at_blur ?? '0',
    origin: raw.origin ?? null,
    originalName: raw.original_name ?? null,
    validFrom: raw.valid_from ? new Date(raw.valid_from) : null,
    validFromBlur: raw.valid_from_blur ?? '0',
    validUntil: raw.valid_until ? new Date(raw.valid_until) : null,
    validUntilBlur: raw.valid_until_blur ?? '0',
    confidence: raw.confidence ?? null,
  };
}

function parseEdge(raw: ApiEdgeResponse): Edge {
  return {
    id: raw.id,
    sourceNodeId: raw.source_node_id,
    targetNodeId: raw.target_node_id,
    type: raw.type as Edge['type'],
    confidence: raw.confidence ?? null,
    runId: raw.run_id ?? null,
    createdAt: new Date(raw.created_at),
  };
}

export async function fetchNodes(params?: {
  level?: number;
  content_class?: string;
  content_type?: string;
  limit?: number;
  offset?: number;
}): Promise<Node[]> {
  const query = new URLSearchParams();
  if (params?.level !== undefined) query.set('level', String(params.level));
  if (params?.content_class) query.set('content_class', params.content_class);
  if (params?.content_type) query.set('content_type', params.content_type);
  query.set('limit', String(params?.limit ?? 1000));
  if (params?.offset) query.set('offset', String(params.offset));

  const resp = await fetch(`${BASE}/api/nodes?${query}`);
  if (!resp.ok) throw new Error(`fetchNodes failed: ${resp.status}`);
  const data = await resp.json();
  return (data.nodes ?? []).map(parseNode);
}

export async function fetchEdges(params?: {
  type?: string;
  source_node_id?: string;
  target_node_id?: string;
  limit?: number;
}): Promise<Edge[]> {
  const query = new URLSearchParams();
  if (params?.type) query.set('type', params.type);
  if (params?.source_node_id) query.set('source_node_id', params.source_node_id);
  if (params?.target_node_id) query.set('target_node_id', params.target_node_id);
  query.set('limit', String(params?.limit ?? 5000));

  const resp = await fetch(`${BASE}/api/edges?${query}`);
  if (!resp.ok) throw new Error(`fetchEdges failed: ${resp.status}`);
  const data = await resp.json();
  return (data.edges ?? []).map(parseEdge);
}

export async function fetchHealth(): Promise<{ status: string; db?: string; s3?: string }> {
  const resp = await fetch(`${BASE}/api/health`);
  return resp.json();
}
