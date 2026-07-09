// package: rest_http / transport
// type:    logic
// job:     the read endpoints — POST /query and the cacheable by-id GET reads — with query mapping and json-seq projection
// limits:  translation only; the reads run behind coreapi.API (-> coreapi)
package rest_http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ranke "github.com/flocko-motion/ranke-go"

	"github.com/flocko-motion/rankedb/adapters/endpoints/coreapi"
	"github.com/flocko-motion/rankedb/api"
)

// Query serves POST /query: it runs the declarative query and streams the
// results in the requested encoding.
func (s *Server) Query(w http.ResponseWriter, r *http.Request) {
	var q api.Query
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeError(w, http.StatusBadRequest, "malformed query")
		return
	}
	cq, err := coreQuery(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	stream, err := s.core.Query(r.Context(), subjectOf(r.Context()), cq)
	if err != nil {
		s.fail(w, err)
		return
	}
	defer stream.Close()

	encoding := "json-seq"
	if q.Output != nil && q.Output.Encoding != nil {
		encoding = string(*q.Output.Encoding)
	}
	// Once the status is written the stream can no longer signal a mid-stream
	// failure through the status code; that is inherent to a streamed response.
	if encoding == "cbor-seq" {
		w.Header().Set("Content-Type", "application/cbor-seq")
		w.WriteHeader(http.StatusOK)
		for stream.Next() {
			b, encErr := stream.Result().Claim.Encode()
			if encErr != nil {
				return
			}
			_, _ = w.Write(b)
		}
		// TODO: emit the trailing QueryReport as a final CBOR item when
		// stream.Report() != nil.
		return
	}

	w.Header().Set("Content-Type", "application/json-seq")
	w.WriteHeader(http.StatusOK)
	for stream.Next() {
		writeJSONSeq(w, projectResult(stream.Result()))
	}
	if rep := stream.Report(); rep != nil {
		writeJSONSeq(w, map[string]any{"report": projectReport(rep)})
	}
}

// GetBranchHead serves GET /{branch}/head.
func (s *Server) GetBranchHead(w http.ResponseWriter, r *http.Request, branch string) {
	head, err := s.core.Head(r.Context(), subjectOf(r.Context()), branch)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.BranchHead{Head: idString(head)})
}

// GetBranchClaim serves GET /{branch}/claim/{id}.
func (s *Server) GetBranchClaim(w http.ResponseWriter, r *http.Request, branch string, idParam string) {
	id, err := ranke.ParseId(idParam)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	claim, err := s.core.Claim(r.Context(), subjectOf(r.Context()), branch, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeClaim(w, claim)
}

// GetBranchContent serves GET /{branch}/content/{hash}.
func (s *Server) GetBranchContent(w http.ResponseWriter, r *http.Request, branch string, hashParam string) {
	hash, err := ranke.ParseId(hashParam)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rc, err := s.core.Content(r.Context(), subjectOf(r.Context()), branch, hash)
	if err != nil {
		s.fail(w, err)
		return
	}
	streamContent(w, hashParam, rc)
}

// GetUniverseClaim serves GET /$universe/claim/{id}.
func (s *Server) GetUniverseClaim(w http.ResponseWriter, r *http.Request, idParam string) {
	id, err := ranke.ParseId(idParam)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	claim, err := s.core.UniverseClaim(r.Context(), subjectOf(r.Context()), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeClaim(w, claim)
}

// GetUniverseContent serves GET /$universe/content/{hash}.
func (s *Server) GetUniverseContent(w http.ResponseWriter, r *http.Request, hashParam string) {
	hash, err := ranke.ParseId(hashParam)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rc, err := s.core.UniverseContent(r.Context(), subjectOf(r.Context()), hash)
	if err != nil {
		s.fail(w, err)
		return
	}
	streamContent(w, hashParam, rc)
}

// coreQuery maps the wire query to the core query. The wire encoding is not part
// of the core query — the adapter applies it to the streamed results.
func coreQuery(q api.Query) (coreapi.Query, error) {
	var cq coreapi.Query

	if q.Select.Branch != nil {
		cq.Select.Branch = *q.Select.Branch
	}
	if q.Select.Claim != nil {
		id, err := ranke.ParseId(*q.Select.Claim)
		if err != nil {
			return cq, fmt.Errorf("select.claim: invalid id")
		}
		cq.Select.Claim = id
	}
	if q.Select.Path != nil {
		for _, p := range *q.Select.Path {
			step := coreapi.PathStep{Edges: p.Edges}
			if p.Dir != nil {
				step.Dir = coreapi.Direction(string(*p.Dir))
			}
			if p.Depth != nil {
				step.Depth = *p.Depth
			}
			if p.Nodes != nil {
				step.Nodes = *p.Nodes
			}
			cq.Select.Path = append(cq.Select.Path, step)
		}
	}

	if q.Output != nil {
		if q.Output.Detail != nil {
			cq.Output.Detail = coreapi.Detail(string(*q.Output.Detail))
		}
		if q.Output.Overflow != nil {
			cq.Output.Overflow = coreapi.Overflow(string(*q.Output.Overflow))
		}
		// TODO: q.Output.Content is a bool|int|string union; resolve it (and a
		// human size like "4kb") to the coreapi.Output.Content byte cap.
	}

	if q.Order != nil {
		cq.Order = &coreapi.Order{
			Field: q.Order.Field,
			Desc:  q.Order.Dir != nil && string(*q.Order.Dir) == "desc",
		}
	}

	if q.Limit != nil {
		if q.Limit.Results != nil {
			cq.Limit.Results = *q.Limit.Results
		}
		if q.Limit.Time != nil {
			if d, err := time.ParseDuration(*q.Limit.Time); err == nil {
				cq.Limit.Time = d
			}
		}
	}

	if q.Execution != nil {
		if q.Execution.Layer != nil {
			cq.Execution.Layer = *q.Execution.Layer
		}
		if q.Execution.Report != nil {
			cq.Execution.Report = *q.Execution.Report
		}
	}

	// TODO: q.Where is a boolean-tree union (and/or/not/field-map); map it to
	// cq.Where. Until then a query filters nothing.

	return cq, nil
}

// writeJSONSeq writes one RFC 7464 record: RS, the JSON value, LF.
func writeJSONSeq(w http.ResponseWriter, v any) {
	_, _ = w.Write([]byte{0x1e})
	_ = json.NewEncoder(w).Encode(v) // Encode appends the LF
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// projectResult is the JSON (convenience) projection of one query result. It is
// not the canonical claim — that is the cbor-seq encoding.
//
// TODO: project the full claim (node fields, edges) per output.detail, not only
// the id and inlined content.
func projectResult(res coreapi.QueryResult) map[string]any {
	m := map[string]any{"id": idString(res.Claim.ID())}
	if res.Content != nil {
		m["content"] = res.Content // marshals to base64
	}
	return m
}

func projectReport(r *coreapi.QueryReport) map[string]any {
	return map[string]any{
		"engine":    r.Engine,
		"layer":     r.Layer,
		"lowered":   r.Lowered,
		"elapsedMs": r.Elapsed.Milliseconds(),
		"results":   r.Results,
		"truncated": r.Truncated,
	}
}
