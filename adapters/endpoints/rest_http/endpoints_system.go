// package: rest_http / transport
// type:    logic
// job:     the operational endpoints — GET /health and GET /system/layers
// limits:  translation only; the values come from coreapi.API (-> coreapi)
package rest_http

import (
	"net/http"

	"github.com/flocko-motion/rankedb/api"
)

// Health serves GET /health.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	h, err := s.core.Health(r.Context())
	if err != nil {
		s.fail(w, err)
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
	layers, err := s.core.Layers(r.Context(), subjectOf(r.Context()))
	if err != nil {
		s.fail(w, err)
		return
	}
	out := api.StorageLayerList{Layers: make([]api.StorageLayer, 0, len(layers))}
	for _, l := range layers {
		out.Layers = append(out.Layers, api.StorageLayer{Name: l.Name, Type: l.Type})
	}
	writeJSON(w, http.StatusOK, out)
}
