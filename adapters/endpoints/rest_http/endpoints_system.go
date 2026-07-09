// package: rest_http / transport
// type:    logic
// job:     the operational endpoints — GET /health and GET /system/layers
// limits:  translation only; the values come from core.Handle (-> internal/core)
package rest_http

import (
	"net/http"

	"github.com/flocko-motion/rankedb/api"
	"github.com/flocko-motion/rankedb/internal/core"
)

// Health serves GET /health.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpHealth}
	if err := s.core.Handle(r.Context(), req); err != nil {
		s.fail(w, err)
		return
	}
	h, ok := req.Response.(core.Health)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := api.Health{Status: h.Status}
	if h.Signer != "" {
		out.Signer = &h.Signer
	}
	writeJSON(w, http.StatusOK, out)
}

// ListStorageLayers serves GET /system/layers.
func (s *Server) ListStorageLayers(w http.ResponseWriter, r *http.Request) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpLayers}
	if err := s.core.Handle(r.Context(), req); err != nil {
		s.fail(w, err)
		return
	}
	layers, ok := req.Response.([]core.StorageLayer)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := api.StorageLayerList{Layers: make([]api.StorageLayer, 0, len(layers))}
	for _, l := range layers {
		out.Layers = append(out.Layers, api.StorageLayer{Name: l.Name, Type: l.Type})
	}
	writeJSON(w, http.StatusOK, out)
}
