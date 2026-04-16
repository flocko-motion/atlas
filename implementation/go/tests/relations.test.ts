import { Api } from "../../frontend/src/api/generated/api.gen";

// Test: create a relation with heads and tails
export async function testCreateRelation(baseUrl: string) {
  const api = new Api({ baseUrl });

  // Setup: worker config
  const configResp = await api.nodes.createNode({
    level: 0,
    content_class: "source",
    content_type: "worker-config",
    encoding_class: "text",
    encoding_format: "json",
    content: '{"name": "relation-test-worker"}',
    edges: [],
  });
  const configId = configResp.data.id;

  // Setup: register run
  const runResp = await api.workers.createRun({ worker_config_id: configId });
  const runId = runResp.data.run_id;

  // Setup: create two entity nodes
  const aliceResp = await api.nodes.createNode({
    level: 2,
    content_class: "entity",
    content_type: "person",
    encoding_class: "text",
    encoding_format: "plain",
    content: "Alice",
    run_id: runId,
    edges: [
      { type: "provenance/input", target_node_id: configId },
      { type: "provenance/worker", target_node_id: configId },
    ],
  });
  const aliceId = aliceResp.data.id;

  const bobResp = await api.nodes.createNode({
    level: 2,
    content_class: "entity",
    content_type: "person",
    encoding_class: "text",
    encoding_format: "plain",
    content: "Bob",
    run_id: runId,
    edges: [
      { type: "provenance/input", target_node_id: configId },
      { type: "provenance/worker", target_node_id: configId },
    ],
  });
  const bobId = bobResp.data.id;

  // Create relation: Alice --[sister_of]--> Bob
  const relResp = await api.nodes.createNode({
    level: 2,
    content_class: "relation",
    content_type: "family",
    encoding_class: "text",
    encoding_format: "plain",
    content: "sister of",
    run_id: runId,
    edges: [
      { type: "provenance/input", target_node_id: configId },
      { type: "provenance/worker", target_node_id: configId },
      { type: "relation/tail", target_node_id: aliceId, confidence: 1.0 },
      { type: "relation/head", target_node_id: bobId, confidence: 1.0 },
    ],
  });
  if (relResp.status !== 200) throw new Error(`Relation creation failed: ${relResp.status} ${JSON.stringify(relResp.data)}`);

  // Verify edges
  const edgesResp = await api.nodes.getNodeEdges(relResp.data.id, { type: "all" });
  if (edgesResp.status !== 200) throw new Error(`Edge fetch failed: ${edgesResp.status}`);
  const edges = edgesResp.data.edges;
  const headEdges = edges.filter((e: any) => e.type === "relation/head");
  const tailEdges = edges.filter((e: any) => e.type === "relation/tail");
  if (headEdges.length !== 1) throw new Error(`Expected 1 head edge, got ${headEdges.length}`);
  if (tailEdges.length !== 1) throw new Error(`Expected 1 tail edge, got ${tailEdges.length}`);
}

// Test: ambiguity — relation with multiple tails (competing candidates)
export async function testAmbiguousRelation(baseUrl: string) {
  const api = new Api({ baseUrl });

  const configResp = await api.nodes.createNode({
    level: 0, content_class: "source", content_type: "worker-config",
    encoding_class: "text", encoding_format: "json",
    content: '{"name": "ambiguity-test"}', edges: [],
  });
  const configId = configResp.data.id;
  const runResp = await api.workers.createRun({ worker_config_id: configId });
  const runId = runResp.data.run_id;

  // Create three entities
  const ids: string[] = [];
  for (const name of ["Alice", "Bob", "Charlie"]) {
    const resp = await api.nodes.createNode({
      level: 2, content_class: "entity", content_type: "person",
      encoding_class: "text", encoding_format: "plain", content: name,
      run_id: runId,
      edges: [
        { type: "provenance/input", target_node_id: configId },
        { type: "provenance/worker", target_node_id: configId },
      ],
    });
    ids.push(resp.data.id);
  }

  // Alice's brother is either Bob (0.7) or Charlie (0.3)
  const relResp = await api.nodes.createNode({
    level: 2, content_class: "relation", content_type: "family",
    encoding_class: "text", encoding_format: "plain", content: "brother of",
    run_id: runId,
    edges: [
      { type: "provenance/input", target_node_id: configId },
      { type: "provenance/worker", target_node_id: configId },
      { type: "relation/tail", target_node_id: ids[1], confidence: 0.7 },
      { type: "relation/tail", target_node_id: ids[2], confidence: 0.3 },
      { type: "relation/head", target_node_id: ids[0], confidence: 1.0 },
    ],
  });
  if (relResp.status !== 200) throw new Error(`Ambiguous relation failed: ${relResp.status}`);

  // Should have 2 tails and 1 head
  const edgesResp = await api.nodes.getNodeEdges(relResp.data.id, { type: "all" });
  const tails = edgesResp.data.edges.filter((e: any) => e.type === "relation/tail");
  if (tails.length !== 2) throw new Error(`Expected 2 tail edges (ambiguity), got ${tails.length}`);
}
