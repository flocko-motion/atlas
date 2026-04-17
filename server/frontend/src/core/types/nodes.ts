/**
 * @layer core
 * @description Node, Edge, and Relation types wrapping the generated API types.
 * @depends (none)
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

export interface Node {
  id: string;
  level: 0 | 1 | 2;
  contentClass: string;
  contentType: string;
  encodingClass: string;
  encodingFormat: string;
  contentSha256: string;
  contentLen: number;
  title: string | null;
  content: string | null;
  createdAt: Date;
  artifactCreatedAt: Date | null;
  artifactCreatedAtBlur: string;
  origin: string | null;
  originalName: string | null;
  validFrom: Date | null;
  validFromBlur: string;
  validUntil: Date | null;
  validUntilBlur: string;
  confidence: number | null;
}

export interface Edge {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  type:
    | 'provenance/input'
    | 'provenance/worker'
    | 'relation/head'
    | 'relation/tail';
  confidence: number | null;
  runId: string | null;
  createdAt: Date;
}

export interface Relation {
  node: Node;
  headEdges: Edge[];
  tailEdges: Edge[];
}
