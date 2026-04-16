import { Api } from "../../frontend/src/api/generated/api.gen";

// Test: create L0 source node and retrieve it
export async function testCreateAndGetSourceNode(baseUrl: string) {
  const api = new Api({ baseUrl });

  // Create a worker config as L0 source
  const createResp = await api.nodes.createNode({
    level: 0,
    content_class: "source",
    content_type: "worker-config",
    encoding_class: "text",
    encoding_format: "json",
    content: '{"name": "test-worker", "version": "0.1"}',
    edges: [],
  });
  if (createResp.status !== 200) throw new Error(`Create failed: ${createResp.status}`);
  const node = createResp.data;
  if (!node.id) throw new Error("No ID returned");
  if (node.level !== 0) throw new Error(`Expected level 0, got ${node.level}`);
  if (node.content_class !== "source") throw new Error(`Expected content_class 'source', got '${node.content_class}'`);

  // Retrieve the node
  const getResp = await api.nodes.getNode(node.id);
  if (getResp.status !== 200) throw new Error(`Get failed: ${getResp.status}`);
  if (getResp.data.id !== node.id) throw new Error("ID mismatch");
  if (getResp.data.content_sha256 !== node.content_sha256) throw new Error("SHA256 mismatch");
}

// Test: L0 root creation is idempotent
export async function testL0Idempotency(baseUrl: string) {
  const api = new Api({ baseUrl });
  const body = {
    level: 0,
    content_class: "source",
    content_type: "data",
    encoding_class: "text",
    encoding_format: "plain",
    content: "idempotency test content " + Date.now(),
    edges: [],
  };

  const resp1 = await api.nodes.createNode(body);
  const resp2 = await api.nodes.createNode(body);
  if (resp1.data.id !== resp2.data.id) throw new Error("L0 root should be idempotent — same content gave different IDs");
}

// Test: L1 node requires provenance edges
export async function testL1RequiresProvenance(baseUrl: string) {
  const api = new Api({ baseUrl });

  const resp = await api.nodes.createNode({
    level: 1,
    content_class: "classification",
    content_type: "entity",
    encoding_class: "text",
    encoding_format: "plain",
    content: "test entity",
    edges: [],
  });
  if (resp.status === 200) throw new Error("L1 node without provenance should be rejected");
}

// Test: create L1 node with proper provenance
export async function testCreateL1WithProvenance(baseUrl: string) {
  const api = new Api({ baseUrl });

  // Create worker config (L0 source)
  const configResp = await api.nodes.createNode({
    level: 0,
    content_class: "source",
    content_type: "worker-config",
    encoding_class: "text",
    encoding_format: "json",
    content: '{"name": "test-classifier", "version": "1.0"}',
    edges: [],
  });
  const configId = configResp.data.id;

  // Create a source node
  const sourceResp = await api.nodes.createNode({
    level: 0,
    content_class: "source",
    content_type: "data",
    encoding_class: "text",
    encoding_format: "plain",
    content: "some raw source content " + Date.now(),
    edges: [],
  });
  const sourceId = sourceResp.data.id;

  // Register a run
  const runResp = await api.workers.createRun({ worker_config_id: configId });
  if (runResp.status !== 200) throw new Error(`Run creation failed: ${runResp.status}`);
  const runId = runResp.data.run_id;

  // Create L1 node with provenance
  const l1Resp = await api.nodes.createNode({
    level: 1,
    content_class: "classification",
    content_type: "entity",
    encoding_class: "text",
    encoding_format: "plain",
    content: "Person: Alice",
    run_id: runId,
    edges: [
      { type: "provenance/input", target_node_id: sourceId },
      { type: "provenance/worker", target_node_id: configId },
    ],
  });
  if (l1Resp.status !== 200) throw new Error(`L1 creation failed: ${l1Resp.status} ${JSON.stringify(l1Resp.data)}`);
  if (l1Resp.data.level !== 1) throw new Error(`Expected level 1, got ${l1Resp.data.level}`);

  // Verify provenance chain
  const provResp = await api.nodes.getNodeProvenance(l1Resp.data.id);
  if (provResp.status !== 200) throw new Error(`Provenance fetch failed: ${provResp.status}`);
  if (provResp.data.nodes.length < 2) throw new Error("Provenance should include at least the source and L1 node");
}

// Test: list nodes
export async function testListNodes(baseUrl: string) {
  const api = new Api({ baseUrl });

  const resp = await api.nodes.listNodes({ limit: 10 });
  if (resp.status !== 200) throw new Error(`List failed: ${resp.status}`);
  if (!Array.isArray(resp.data.nodes)) throw new Error("Expected nodes array");
}
