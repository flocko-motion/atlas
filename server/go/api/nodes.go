package api

import (
	"context"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"rankedb/s3"

	"github.com/google/uuid"

	db "rankedb/db"

	schemafapi "github.com/flocko-motion/schemaf/api"
	schemafdb "github.com/flocko-motion/schemaf/db"
)

// contentCacheMaxBytes is the inline-cache threshold for L0 text content.
// Nodes whose content is text and <= this size get their full bytes mirrored
// into the Postgres content_cached column (for cheap API reads and FTS).
// Anything larger lives only in S3 and is streamed on demand.
const contentCacheMaxBytes = 32 * 1024

// ─── Create node ──────────────────────────────────────────────────────────────

// CreateNodeEndpoint creates a graph node.
// L0 sources use multipart form upload; L1/L2 use JSON.
// L0 root creation is idempotent via content_sha256.
type CreateNodeEndpoint struct{}

func (e CreateNodeEndpoint) Method() string { return "POST" }
func (e CreateNodeEndpoint) Path() string   { return "/api/nodes" }
func (e CreateNodeEndpoint) Auth() bool     { return false }

// createNodeRequest is the JSON body for node creation.
type createNodeRequest struct {
	Level                 int              `json:"level"`
	ContentClass          string           `json:"content_class"`
	ContentType           string           `json:"content_type"`
	EncodingClass         string           `json:"encoding_class"`
	EncodingFormat        string           `json:"encoding_format"`
	Title                 *string          `json:"title,omitempty"`
	Content               *string          `json:"content,omitempty"`
	ArtifactCreatedAt     *string          `json:"artifact_created_at,omitempty"`
	ArtifactCreatedAtBlur *string          `json:"artifact_created_at_blur,omitempty"`
	Origin                *string          `json:"origin,omitempty"`
	OriginalName          *string          `json:"original_name,omitempty"`
	ValidFrom             *string          `json:"valid_from,omitempty"`
	ValidFromBlur         *string          `json:"valid_from_blur,omitempty"`
	ValidUntil            *string          `json:"valid_until,omitempty"`
	ValidUntilBlur        *string          `json:"valid_until_blur,omitempty"`
	Confidence            *float64         `json:"confidence,omitempty"`
	Edges                 []createEdgeSpec `json:"edges,omitempty"`
}

type createEdgeSpec struct {
	SourceNodeID string   `json:"source_node_id"`
	TargetNodeID string   `json:"target_node_id"`
	Type         string   `json:"type"`
	Confidence   *float64 `json:"confidence,omitempty"`
	RunID        *string  `json:"run_id,omitempty"`
}

func (e CreateNodeEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	var req createNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return nil
	}

	// Resolve content bytes
	isText := req.EncodingClass == "text"
	var contentBytes []byte

	if req.Content != nil {
		if isText {
			contentBytes = []byte(*req.Content)
		} else {
			// Binary content arrives base64-encoded in the JSON string
			var err error
			contentBytes, err = base64.StdEncoding.DecodeString(*req.Content)
			if err != nil {
				http.Error(w, "binary content must be base64-encoded: "+err.Error(), http.StatusBadRequest)
				return nil
			}
		}
	}

	hash := sha256.Sum256(contentBytes)
	contentSha256 := hex.EncodeToString(hash[:])
	contentLen := int64(len(contentBytes))

	// Determine node ID
	var nodeID string
	ctx := r.Context()
	queries := db.New(schemafdb.DB())

	if req.Level == 0 {
		// L0 roots: deterministic ID from hash, idempotent
		existingID, err := queries.NodeExistsBySha256(ctx, contentSha256)
		if err == nil && existingID != "" {
			// Already exists, return existing node
			node, err := queries.GetNode(ctx, existingID)
			if err != nil {
				http.Error(w, "failed to fetch existing node: "+err.Error(), http.StatusInternalServerError)
				return nil
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return json.NewEncoder(w).Encode(nodeToResponse(node))
		}
		// Generate deterministic ID from sha256
		nodeID = fmt.Sprintf("n0-%s", contentSha256[:32])
	} else {
		// L1/L2: UUID v7
		id, err := uuid.NewV7()
		if err != nil {
			http.Error(w, "failed to generate UUID: "+err.Error(), http.StatusInternalServerError)
			return nil
		}
		nodeID = id.String()
	}

	// Validate edges for non-root nodes
	if req.Level > 0 {
		hasProvenanceInput := false
		provenanceWorkerCount := 0
		for _, edge := range req.Edges {
			if edge.Type == "provenance/input" {
				hasProvenanceInput = true
			}
			if edge.Type == "provenance/worker" {
				provenanceWorkerCount++
			}
		}
		if !hasProvenanceInput {
			http.Error(w, "non-root nodes require at least one provenance/input edge", http.StatusBadRequest)
			return nil
		}
		if provenanceWorkerCount != 1 {
			http.Error(w, "non-root nodes require exactly one provenance/worker edge", http.StatusBadRequest)
			return nil
		}
	}

	// Parse optional time fields
	artifactCreatedAt := sql.NullTime{}
	if req.ArtifactCreatedAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ArtifactCreatedAt)
		if err == nil {
			artifactCreatedAt = sql.NullTime{Time: t, Valid: true}
		}
	}
	validFrom := sql.NullTime{}
	if req.ValidFrom != nil {
		t, err := time.Parse(time.RFC3339, *req.ValidFrom)
		if err == nil {
			validFrom = sql.NullTime{Time: t, Valid: true}
		}
	}
	validUntil := sql.NullTime{}
	if req.ValidUntil != nil {
		t, err := time.Parse(time.RFC3339, *req.ValidUntil)
		if err == nil {
			validUntil = sql.NullTime{Time: t, Valid: true}
		}
	}

	// Upload content to S3
	if contentLen > 0 {
		mimeType := req.EncodingClass + "/" + req.EncodingFormat
		if err := s3.Put(ctx, contentSha256, bytes.NewReader(contentBytes), contentLen, mimeType); err != nil {
			http.Error(w, "failed to upload content to S3: "+err.Error(), http.StatusInternalServerError)
			return nil
		}
	}

	// Begin transaction
	tx, err := schemafdb.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "failed to begin transaction: "+err.Error(), http.StatusInternalServerError)
		return nil
	}
	defer tx.Rollback()

	qtx := db.New(tx)

	// Insert node
	err = qtx.InsertNode(ctx, db.InsertNodeParams{
		ID:             nodeID,
		Level:          int16(req.Level),
		ContentClass:   req.ContentClass,
		ContentType:    req.ContentType,
		EncodingClass:  req.EncodingClass,
		EncodingFormat: req.EncodingFormat,
		ContentSha256:  contentSha256,
		ContentLen:     contentLen,
		ContentCached: sql.NullString{
			String: string(contentBytes),
			Valid:  isText && len(contentBytes) > 0 && len(contentBytes) <= contentCacheMaxBytes,
		},
		Title: sql.NullString{
			String: ptrString(req.Title),
			Valid:  req.Title != nil,
		},
		ArtifactCreatedAt: artifactCreatedAt,
		ArtifactCreatedAtBlur: sql.NullString{
			String: ptrString(req.ArtifactCreatedAtBlur),
			Valid:  req.ArtifactCreatedAtBlur != nil,
		},
		Origin: sql.NullString{
			String: ptrString(req.Origin),
			Valid:  req.Origin != nil,
		},
		OriginalName: sql.NullString{
			String: ptrString(req.OriginalName),
			Valid:  req.OriginalName != nil,
		},
		ValidFrom: validFrom,
		ValidFromBlur: sql.NullString{
			String: ptrString(req.ValidFromBlur),
			Valid:  req.ValidFromBlur != nil,
		},
		ValidUntil: validUntil,
		ValidUntilBlur: sql.NullString{
			String: ptrString(req.ValidUntilBlur),
			Valid:  req.ValidUntilBlur != nil,
		},
		Confidence: sql.NullFloat64{
			Float64: ptrFloat64(req.Confidence),
			Valid:   req.Confidence != nil,
		},
	})
	if err != nil {
		http.Error(w, "failed to insert node: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	// Insert edges
	for _, edge := range req.Edges {
		edgeID, err := uuid.NewV7()
		if err != nil {
			http.Error(w, "failed to generate edge UUID: "+err.Error(), http.StatusInternalServerError)
			return nil
		}
		sourceID := edge.SourceNodeID
		targetID := edge.TargetNodeID
		// If source or target is empty, fill in the new node ID
		if sourceID == "" {
			sourceID = nodeID
		}
		if targetID == "" {
			targetID = nodeID
		}
		err = qtx.InsertEdge(ctx, db.InsertEdgeParams{
			ID:           edgeID.String(),
			SourceNodeID: sourceID,
			TargetNodeID: targetID,
			Type:         edge.Type,
			Confidence: sql.NullFloat64{
				Float64: ptrFloat64(edge.Confidence),
				Valid:   edge.Confidence != nil,
			},
			RunID: sql.NullString{
				String: ptrString(edge.RunID),
				Valid:  edge.RunID != nil,
			},
		})
		if err != nil {
			http.Error(w, "failed to insert edge: "+err.Error(), http.StatusInternalServerError)
			return nil
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	// Fetch the created node to get server-side defaults (created_at)
	node, err := queries.GetNode(ctx, nodeID)
	if err != nil {
		http.Error(w, "failed to fetch created node: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(nodeToResponse(node))
}

// ptrString safely dereferences a *string, returning "" if nil.
func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ptrFloat64 safely dereferences a *float64, returning 0 if nil.
func ptrFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// ─── Get node ─────────────────────────────────────────────────────────────────

// GetNodeEndpoint returns a node by ID with full metadata and inline content.
type GetNodeEndpoint struct{}

func (e GetNodeEndpoint) Method() string { return "GET" }
func (e GetNodeEndpoint) Path() string   { return "/api/nodes/{id}" }
func (e GetNodeEndpoint) Auth() bool     { return false }
func (e GetNodeEndpoint) Handle(ctx context.Context, req GetNodeReq) (NodeResponse, error) {
	queries := db.New(schemafdb.DB())
	id := req.ID
	// If the path param is a raw SHA-256 (64 hex chars), resolve to node ID first.
	// Lets clients precheck existence by content hash without a separate endpoint.
	if isSha256(id) {
		resolved, err := queries.NodeExistsBySha256(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return NodeResponse{}, schemafapi.ErrNotFound
			}
			return NodeResponse{}, err
		}
		id = resolved
	}
	node, err := queries.GetNode(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return NodeResponse{}, schemafapi.ErrNotFound
		}
		return NodeResponse{}, err
	}
	return nodeToResponse(node), nil
}

func isSha256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < 64; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

type GetNodeReq struct {
	ID string `path:"id"`
}

// ─── Get node content ─────────────────────────────────────────────────────────

// GetNodeContentEndpoint returns the raw content bytes of a node.
// Serves from Postgres cache if available, otherwise proxies from S3.
type GetNodeContentEndpoint struct{}

func (e GetNodeContentEndpoint) Method() string { return "GET" }
func (e GetNodeContentEndpoint) Path() string   { return "/api/nodes/{id}/content" }
func (e GetNodeContentEndpoint) Auth() bool     { return false }

func (e GetNodeContentEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	queries := db.New(schemafdb.DB())
	node, err := queries.GetNode(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "node not found", http.StatusNotFound)
			return nil
		}
		return err
	}

	// Serve from Postgres cache if available, otherwise proxy from S3
	if node.ContentCached.Valid {
		contentType := node.EncodingClass + "/" + node.EncodingFormat
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(node.ContentCached.String))
		return err
	}

	reader, size, contentType, err := s3.Get(r.Context(), node.ContentSha256)
	if err != nil {
		http.Error(w, "content not found in S3: "+err.Error(), http.StatusNotFound)
		return nil
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, reader)
	return err
}

// ─── Get node provenance ──────────────────────────────────────────────────────

// GetNodeProvenanceEndpoint returns the upstream provenance chain from a node back to L0 roots.
type GetNodeProvenanceEndpoint struct{}

func (e GetNodeProvenanceEndpoint) Method() string { return "GET" }
func (e GetNodeProvenanceEndpoint) Path() string   { return "/api/nodes/{id}/provenance" }
func (e GetNodeProvenanceEndpoint) Auth() bool     { return false }
func (e GetNodeProvenanceEndpoint) Handle(ctx context.Context, req GetNodeProvenanceReq) (ProvenanceSubgraph, error) {
	conn := schemafdb.DB()

	// Recursive CTE to traverse provenance edges upstream
	const query = `
WITH RECURSIVE prov AS (
    SELECT e.id, e.source_node_id, e.target_node_id, e.type, e.confidence, e.run_id, e.created_at
    FROM edges e
    WHERE e.target_node_id = $1 AND e.type IN ('provenance/input', 'provenance/worker')
    UNION
    SELECT e.id, e.source_node_id, e.target_node_id, e.type, e.confidence, e.run_id, e.created_at
    FROM edges e
    INNER JOIN prov p ON e.target_node_id = p.source_node_id
    WHERE e.type IN ('provenance/input', 'provenance/worker')
)
SELECT DISTINCT id, source_node_id, target_node_id, type, confidence, run_id, created_at FROM prov
`
	rows, err := conn.QueryContext(ctx, query, req.ID)
	if err != nil {
		return ProvenanceSubgraph{}, err
	}
	defer rows.Close()

	var edges []EdgeResponse
	nodeIDs := map[string]bool{req.ID: true}
	for rows.Next() {
		var e db.Edge
		if err := rows.Scan(&e.ID, &e.SourceNodeID, &e.TargetNodeID, &e.Type, &e.Confidence, &e.RunID, &e.CreatedAt); err != nil {
			return ProvenanceSubgraph{}, err
		}
		edges = append(edges, edgeToResponse(e))
		nodeIDs[e.SourceNodeID] = true
		nodeIDs[e.TargetNodeID] = true
	}
	if err := rows.Err(); err != nil {
		return ProvenanceSubgraph{}, err
	}

	// Fetch all referenced nodes
	queries := db.New(conn)
	var nodes []NodeResponse
	for id := range nodeIDs {
		node, err := queries.GetNode(ctx, id)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return ProvenanceSubgraph{}, err
		}
		nodes = append(nodes, nodeToResponse(node))
	}

	if edges == nil {
		edges = []EdgeResponse{}
	}
	if nodes == nil {
		nodes = []NodeResponse{}
	}

	return ProvenanceSubgraph{Nodes: nodes, Edges: edges}, nil
}

type GetNodeProvenanceReq struct {
	ID string `path:"id"`
}

// ─── List nodes ───────────────────────────────────────────────────────────────

// ListNodesEndpoint returns a filtered, paginated list of nodes.
type ListNodesEndpoint struct{}

func (e ListNodesEndpoint) Method() string { return "GET" }
func (e ListNodesEndpoint) Path() string   { return "/api/nodes" }
func (e ListNodesEndpoint) Auth() bool     { return false }
func (e ListNodesEndpoint) Handle(ctx context.Context, req ListNodesReq) (ListNodesResp, error) {
	queries := db.New(schemafdb.DB())
	limit := defaultLimit(req.Limit)
	offset := int32(req.Offset)

	var dbNodes []db.Node
	var err error

	switch {
	case req.Level != nil && req.ContentClass != nil && req.ContentType != nil:
		// Filter by level is not combined in a single query, use content class + type
		dbNodes, err = queries.ListNodesByContentClassAndType(ctx, db.ListNodesByContentClassAndTypeParams{
			ContentClass: *req.ContentClass,
			ContentType:  *req.ContentType,
			Limit:        limit,
			Offset:       offset,
		})
	case req.ContentClass != nil && req.ContentType != nil:
		dbNodes, err = queries.ListNodesByContentClassAndType(ctx, db.ListNodesByContentClassAndTypeParams{
			ContentClass: *req.ContentClass,
			ContentType:  *req.ContentType,
			Limit:        limit,
			Offset:       offset,
		})
	case req.ContentClass != nil:
		dbNodes, err = queries.ListNodesByContentClass(ctx, db.ListNodesByContentClassParams{
			ContentClass: *req.ContentClass,
			Limit:        limit,
			Offset:       offset,
		})
	case req.Level != nil:
		dbNodes, err = queries.ListNodesByLevel(ctx, db.ListNodesByLevelParams{
			Level:  int16(*req.Level),
			Limit:  limit,
			Offset: offset,
		})
	default:
		// Use raw SQL for ListNodes since the sqlc query hasn't been regenerated yet
		rows, qErr := schemafdb.DB().QueryContext(ctx,
			"SELECT id, level, content_class, content_type, encoding_class, encoding_format, content_sha256, content_len, content_cached, created_at, artifact_created_at, artifact_created_at_blur, origin, original_name, valid_from, valid_from_blur, valid_until, valid_until_blur, confidence, title FROM nodes ORDER BY created_at DESC LIMIT $1 OFFSET $2",
			limit, offset)
		if qErr != nil {
			return ListNodesResp{Nodes: []NodeResponse{}}, qErr
		}
		defer rows.Close()
		for rows.Next() {
			var n db.Node
			if scanErr := rows.Scan(
				&n.ID, &n.Level, &n.ContentClass, &n.ContentType,
				&n.EncodingClass, &n.EncodingFormat, &n.ContentSha256, &n.ContentLen,
				&n.ContentCached, &n.CreatedAt, &n.ArtifactCreatedAt, &n.ArtifactCreatedAtBlur,
				&n.Origin, &n.OriginalName, &n.ValidFrom, &n.ValidFromBlur,
				&n.ValidUntil, &n.ValidUntilBlur, &n.Confidence, &n.Title,
			); scanErr != nil {
				return ListNodesResp{Nodes: []NodeResponse{}}, scanErr
			}
			dbNodes = append(dbNodes, n)
		}
		if rows.Err() != nil {
			return ListNodesResp{Nodes: []NodeResponse{}}, rows.Err()
		}
	}
	if err != nil {
		return ListNodesResp{Nodes: []NodeResponse{}}, err
	}

	nodes := make([]NodeResponse, 0, len(dbNodes))
	for _, n := range dbNodes {
		nodes = append(nodes, nodeToResponse(n))
	}
	return ListNodesResp{Nodes: nodes}, nil
}

type ListNodesReq struct {
	Level        *int    `query:"level"`
	ContentClass *string `query:"content_class"`
	ContentType  *string `query:"content_type"`
	Limit        int     `query:"limit"`
	Offset       int     `query:"offset"`
}

type ListNodesResp struct {
	Nodes []NodeResponse `json:"nodes"`
}

// ─── Get node edges ───────────────────────────────────────────────────────────

// GetNodeEdgesEndpoint returns edges connected to a node, filterable by direction and type.
type GetNodeEdgesEndpoint struct{}

func (e GetNodeEdgesEndpoint) Method() string { return "GET" }
func (e GetNodeEdgesEndpoint) Path() string   { return "/api/nodes/{id}/edges" }
func (e GetNodeEdgesEndpoint) Auth() bool     { return false }
func (e GetNodeEdgesEndpoint) Handle(ctx context.Context, req GetNodeEdgesReq) (GetNodeEdgesResp, error) {
	queries := db.New(schemafdb.DB())

	var dbEdges []db.Edge
	var err error

	direction := "both"
	if req.Direction != nil {
		direction = *req.Direction
	}

	switch {
	case req.Type != nil && (direction == "outgoing" || direction == "source"):
		dbEdges, err = queries.GetEdgesBySourceAndType(ctx, db.GetEdgesBySourceAndTypeParams{
			SourceNodeID: req.ID,
			Type:         *req.Type,
		})
	case req.Type != nil && (direction == "incoming" || direction == "target"):
		dbEdges, err = queries.GetEdgesByTargetAndType(ctx, db.GetEdgesByTargetAndTypeParams{
			TargetNodeID: req.ID,
			Type:         *req.Type,
		})
	case direction == "outgoing" || direction == "source":
		dbEdges, err = queries.GetEdgesBySource(ctx, req.ID)
	case direction == "incoming" || direction == "target":
		dbEdges, err = queries.GetEdgesByTarget(ctx, req.ID)
	default:
		// both directions
		sourceEdges, sErr := queries.GetEdgesBySource(ctx, req.ID)
		if sErr != nil {
			return GetNodeEdgesResp{Edges: []EdgeResponse{}}, sErr
		}
		targetEdges, tErr := queries.GetEdgesByTarget(ctx, req.ID)
		if tErr != nil {
			return GetNodeEdgesResp{Edges: []EdgeResponse{}}, tErr
		}
		dbEdges = append(sourceEdges, targetEdges...)
		if req.Type != nil {
			filtered := make([]db.Edge, 0)
			for _, edge := range dbEdges {
				if edge.Type == *req.Type {
					filtered = append(filtered, edge)
				}
			}
			dbEdges = filtered
		}
	}
	if err != nil {
		return GetNodeEdgesResp{Edges: []EdgeResponse{}}, err
	}

	edges := make([]EdgeResponse, 0, len(dbEdges))
	for _, edge := range dbEdges {
		edges = append(edges, edgeToResponse(edge))
	}
	return GetNodeEdgesResp{Edges: edges}, nil
}

type GetNodeEdgesReq struct {
	ID        string  `path:"id"`
	Direction *string `query:"direction"` // incoming, outgoing, both
	Type      *string `query:"type"`      // provenance/input, provenance/worker, relation/head, relation/tail
	Limit     int     `query:"limit"`
	Offset    int     `query:"offset"`
}

type GetNodeEdgesResp struct {
	Edges []EdgeResponse `json:"edges"`
}
