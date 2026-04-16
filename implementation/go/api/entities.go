package api

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	db "rankedb/db"

	schemafdb "github.com/flocko-motion/schemaf/db"
)

// ─── Get entity ───────────────────────────────────────────────────────────────

// GetEntityEndpoint returns an entity with all its relations, sorted by temporal validity.
type GetEntityEndpoint struct{}

func (e GetEntityEndpoint) Method() string { return "GET" }
func (e GetEntityEndpoint) Path() string   { return "/api/entities/{id}" }
func (e GetEntityEndpoint) Auth() bool     { return false }
func (e GetEntityEndpoint) Handle(ctx context.Context, req GetEntityReq) (GetEntityResp, error) {
	queries := db.New(schemafdb.DB())

	// Fetch the entity node
	entity, err := queries.GetNode(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return GetEntityResp{}, fmt.Errorf("entity not found: %s", req.ID)
		}
		return GetEntityResp{}, err
	}

	relations, err := getEntityRelations(ctx, queries, req.ID)
	if err != nil {
		return GetEntityResp{}, err
	}

	return GetEntityResp{
		Entity:    nodeToResponse(entity),
		Relations: relations,
	}, nil
}

type GetEntityReq struct {
	ID            string   `path:"id"`
	MinConfidence *float64 `query:"min_confidence"`
}

type GetEntityResp struct {
	Entity    NodeResponse       `json:"entity"`
	Relations []RelationResponse `json:"relations"`
}

// getEntityRelations fetches all relations connected to an entity via relation/head and relation/tail edges.
func getEntityRelations(ctx context.Context, queries *db.Queries, entityID string) ([]RelationResponse, error) {
	// Get all edges where this entity is a target with type relation/head or relation/tail
	headEdges, err := queries.GetEdgesByTargetAndType(ctx, db.GetEdgesByTargetAndTypeParams{
		TargetNodeID: entityID,
		Type:         "relation/head",
	})
	if err != nil {
		return nil, err
	}
	tailEdges, err := queries.GetEdgesByTargetAndType(ctx, db.GetEdgesByTargetAndTypeParams{
		TargetNodeID: entityID,
		Type:         "relation/tail",
	})
	if err != nil {
		return nil, err
	}

	// Collect unique relation node IDs (sources of these edges)
	relationIDs := map[string]bool{}
	for _, e := range headEdges {
		relationIDs[e.SourceNodeID] = true
	}
	for _, e := range tailEdges {
		relationIDs[e.SourceNodeID] = true
	}

	// For each relation, fetch node + all head/tail edges
	var relations []RelationResponse
	for relID := range relationIDs {
		relNode, err := queries.GetNode(ctx, relID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, err
		}

		relHeadEdges, err := queries.GetEdgesBySourceAndType(ctx, db.GetEdgesBySourceAndTypeParams{
			SourceNodeID: relID,
			Type:         "relation/head",
		})
		if err != nil {
			return nil, err
		}
		relTailEdges, err := queries.GetEdgesBySourceAndType(ctx, db.GetEdgesBySourceAndTypeParams{
			SourceNodeID: relID,
			Type:         "relation/tail",
		})
		if err != nil {
			return nil, err
		}

		headResp := make([]EdgeResponse, 0, len(relHeadEdges))
		for _, e := range relHeadEdges {
			headResp = append(headResp, edgeToResponse(e))
		}
		tailResp := make([]EdgeResponse, 0, len(relTailEdges))
		for _, e := range relTailEdges {
			tailResp = append(tailResp, edgeToResponse(e))
		}

		relations = append(relations, RelationResponse{
			Node:      nodeToResponse(relNode),
			HeadEdges: headResp,
			TailEdges: tailResp,
		})
	}

	if relations == nil {
		relations = []RelationResponse{}
	}
	return relations, nil
}

// ─── Search entities ──────────────────────────────────────────────────────────

// SearchEntitiesEndpoint searches entities by full-text query over names and aliases.
type SearchEntitiesEndpoint struct{}

func (e SearchEntitiesEndpoint) Method() string { return "GET" }
func (e SearchEntitiesEndpoint) Path() string   { return "/api/entities" }
func (e SearchEntitiesEndpoint) Auth() bool     { return false }
func (e SearchEntitiesEndpoint) Handle(ctx context.Context, req SearchEntitiesReq) (SearchEntitiesResp, error) {
	conn := schemafdb.DB()
	queries := db.New(conn)
	limit := defaultLimit(req.Limit)
	offset := int32(req.Offset)

	var dbNodes []db.Node
	var err error

	if req.Q != nil && *req.Q != "" {
		// Full-text search filtered to entities
		// Use raw SQL since SearchNodesByContent won't be regenerated yet
		rows, qErr := conn.QueryContext(ctx,
			`SELECT id, level, content_class, content_type, encoding_class, encoding_format, content_sha256, content_len, content_cached, created_at, artifact_created_at, artifact_created_at_blur, origin, original_name, valid_from, valid_from_blur, valid_until, valid_until_blur, confidence
			 FROM nodes
			 WHERE content_class = 'entity'
			   AND content_cached IS NOT NULL
			   AND content_tsv @@ plainto_tsquery('english', $1)
			 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			*req.Q, limit, offset)
		if qErr != nil {
			return SearchEntitiesResp{Entities: []NodeResponse{}}, qErr
		}
		defer rows.Close()
		for rows.Next() {
			var n db.Node
			if scanErr := rows.Scan(
				&n.ID, &n.Level, &n.ContentClass, &n.ContentType,
				&n.EncodingClass, &n.EncodingFormat, &n.ContentSha256, &n.ContentLen,
				&n.ContentCached, &n.CreatedAt, &n.ArtifactCreatedAt, &n.ArtifactCreatedAtBlur,
				&n.Origin, &n.OriginalName, &n.ValidFrom, &n.ValidFromBlur,
				&n.ValidUntil, &n.ValidUntilBlur, &n.Confidence,
			); scanErr != nil {
				return SearchEntitiesResp{Entities: []NodeResponse{}}, scanErr
			}
			dbNodes = append(dbNodes, n)
		}
		if rows.Err() != nil {
			return SearchEntitiesResp{Entities: []NodeResponse{}}, rows.Err()
		}
	} else if req.Type != nil && *req.Type != "" {
		dbNodes, err = queries.ListNodesByContentClassAndType(ctx, db.ListNodesByContentClassAndTypeParams{
			ContentClass: "entity",
			ContentType:  *req.Type,
			Limit:        limit,
			Offset:       offset,
		})
	} else {
		dbNodes, err = queries.ListNodesByContentClass(ctx, db.ListNodesByContentClassParams{
			ContentClass: "entity",
			Limit:        limit,
			Offset:       offset,
		})
	}
	if err != nil {
		return SearchEntitiesResp{Entities: []NodeResponse{}}, err
	}

	entities := make([]NodeResponse, 0, len(dbNodes))
	for _, n := range dbNodes {
		entities = append(entities, nodeToResponse(n))
	}
	return SearchEntitiesResp{Entities: entities}, nil
}

type SearchEntitiesReq struct {
	Q      *string `query:"q"`
	Type   *string `query:"type"` // e.g. person, organization
	Limit  int     `query:"limit"`
	Offset int     `query:"offset"`
}

type SearchEntitiesResp struct {
	Entities []NodeResponse `json:"entities"`
}

// ─── Entity timeline ──────────────────────────────────────────────────────────

// GetEntityTimelineEndpoint returns all relations of an entity sorted chronologically by valid_from.
type GetEntityTimelineEndpoint struct{}

func (e GetEntityTimelineEndpoint) Method() string { return "GET" }
func (e GetEntityTimelineEndpoint) Path() string   { return "/api/entities/{id}/timeline" }
func (e GetEntityTimelineEndpoint) Auth() bool     { return false }
func (e GetEntityTimelineEndpoint) Handle(ctx context.Context, req GetEntityTimelineReq) (GetEntityTimelineResp, error) {
	queries := db.New(schemafdb.DB())

	relations, err := getEntityRelations(ctx, queries, req.ID)
	if err != nil {
		return GetEntityTimelineResp{Relations: []RelationResponse{}}, err
	}

	// Sort by valid_from ascending (nil values at the end)
	sort.Slice(relations, func(i, j int) bool {
		vi := relations[i].Node.ValidFrom
		vj := relations[j].Node.ValidFrom
		if vi == nil && vj == nil {
			return false
		}
		if vi == nil {
			return false
		}
		if vj == nil {
			return true
		}
		return *vi < *vj
	})

	return GetEntityTimelineResp{Relations: relations}, nil
}

type GetEntityTimelineReq struct {
	ID string `path:"id"`
}

type GetEntityTimelineResp struct {
	Relations []RelationResponse `json:"relations"`
}

// ─── List relations ───────────────────────────────────────────────────────────

// ListRelationsEndpoint returns filtered relation nodes.
type ListRelationsEndpoint struct{}

func (e ListRelationsEndpoint) Method() string { return "GET" }
func (e ListRelationsEndpoint) Path() string   { return "/api/relations" }
func (e ListRelationsEndpoint) Auth() bool     { return false }
func (e ListRelationsEndpoint) Handle(ctx context.Context, req ListRelationsReq) (ListRelationsResp, error) {
	conn := schemafdb.DB()
	queries := db.New(conn)
	limit := defaultLimit(req.Limit)
	offset := int32(req.Offset)

	var dbNodes []db.Node
	var err error

	if req.Type != nil && *req.Type != "" {
		dbNodes, err = queries.ListNodesByContentClassAndType(ctx, db.ListNodesByContentClassAndTypeParams{
			ContentClass: "relation",
			ContentType:  *req.Type,
			Limit:        limit,
			Offset:       offset,
		})
	} else {
		dbNodes, err = queries.ListNodesByContentClass(ctx, db.ListNodesByContentClassParams{
			ContentClass: "relation",
			Limit:        limit,
			Offset:       offset,
		})
	}
	if err != nil {
		return ListRelationsResp{Relations: []RelationResponse{}}, err
	}

	// Build relation responses with head/tail edges
	var relations []RelationResponse
	for _, n := range dbNodes {
		headEdges, hErr := queries.GetEdgesBySourceAndType(ctx, db.GetEdgesBySourceAndTypeParams{
			SourceNodeID: n.ID,
			Type:         "relation/head",
		})
		if hErr != nil {
			return ListRelationsResp{Relations: []RelationResponse{}}, hErr
		}
		tailEdges, tErr := queries.GetEdgesBySourceAndType(ctx, db.GetEdgesBySourceAndTypeParams{
			SourceNodeID: n.ID,
			Type:         "relation/tail",
		})
		if tErr != nil {
			return ListRelationsResp{Relations: []RelationResponse{}}, tErr
		}

		// If unresolved filter is set, check tail count
		if req.Unresolved != nil && *req.Unresolved {
			if len(tailEdges) == 1 {
				continue // resolved, skip
			}
		}

		headResp := make([]EdgeResponse, 0, len(headEdges))
		for _, e := range headEdges {
			headResp = append(headResp, edgeToResponse(e))
		}
		tailResp := make([]EdgeResponse, 0, len(tailEdges))
		for _, e := range tailEdges {
			tailResp = append(tailResp, edgeToResponse(e))
		}

		relations = append(relations, RelationResponse{
			Node:      nodeToResponse(n),
			HeadEdges: headResp,
			TailEdges: tailResp,
		})
	}

	if relations == nil {
		relations = []RelationResponse{}
	}
	return ListRelationsResp{Relations: relations}, nil
}

type ListRelationsReq struct {
	Unresolved *bool   `query:"unresolved"`
	Type       *string `query:"type"` // e.g. alias, has_role
	Limit      int     `query:"limit"`
	Offset     int     `query:"offset"`
}

type ListRelationsResp struct {
	Relations []RelationResponse `json:"relations"`
}
