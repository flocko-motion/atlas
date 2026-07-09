// package: rest_http / transport
// type:    logic
// job:     the verification run endpoints under /system/verification, and config mapping
// limits:  translation only; runs are managed behind core.Handle (-> internal/core)
package rest_http

import (
	"encoding/json"
	"net/http"

	"github.com/flocko-motion/rankedb/internal/core"
	"github.com/flocko-motion/rankedb/openapi"
)

// StartVerification serves POST /system/verification.
func (s *Server) StartVerification(w http.ResponseWriter, r *http.Request) {
	var cfg openapi.VerificationConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, core.CatInvalid, "malformed verification config")
		return
	}
	vc := coreVerConfig(cfg)
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerificationStart, VerConfig: &vc}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	// TODO(location): once execute produces the report, buffer the stream instead of
	// streaming it through, read the run id from the buffered body, set
	// Location: /system/verification/{id}, then write the buffer. The body already
	// carries the id; buffering lets this route derive the header with no core change
	// (a small create-response is cheap to buffer, unlike a query or a blob).
	s.respond(w, stream, http.StatusAccepted)
}

// ListVerifications serves GET /system/verification.
func (s *Server) ListVerifications(w http.ResponseWriter, r *http.Request) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerificationList}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// GetVerification serves GET /system/verification/{id}.
func (s *Server) GetVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerificationGet, VerID: reportId}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// CancelVerification serves POST /system/verification/{id}/cancel.
func (s *Server) CancelVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerificationCancel, VerID: reportId}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// DeleteVerification serves DELETE /system/verification/{id}.
func (s *Server) DeleteVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerificationDelete, VerID: reportId}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	_ = stream.Close() // no body on a 204
	w.WriteHeader(http.StatusNoContent)
}

// coreVerConfig maps the wire verification config to core's.
func coreVerConfig(c openapi.VerificationConfig) core.VerificationConfig {
	cfg := core.VerificationConfig{Closure: c.Closure}
	if c.Layer != nil {
		cfg.Layer = *c.Layer
	}
	if c.Depth != nil {
		cfg.Depth = core.VerificationDepth(string(*c.Depth))
	}
	if c.ContentThreshold != nil {
		cfg.ContentThreshold = int64(*c.ContentThreshold)
	}
	return cfg
}
