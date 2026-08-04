// package: rest_http / transport
// type:    logic
// job:     the verification run endpoints under /system/verification, and config mapping
// limits:  translation only; runs are managed behind core.Handle (-> internal/core)
package rest_http

import (
	"bytes"
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
	// The contract promises Location and a first-poll hint on 202. Both come from the
	// report itself, so the body is buffered — cheap for a create-response, unlike a
	// query — and the id read back out of it rather than threaded through core.
	body, id, err := bufferRun(stream)
	if err != nil {
		s.fail(w, err)
		return
	}
	if id != "" {
		w.Header().Set("Location", "/system/verifications/"+id)
	}
	w.Header().Set("Retry-After", retryAfterSeconds)
	s.respondBytes(w, mediaJSON, http.StatusAccepted, body)
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

// GetVerification serves GET /system/verifications/{id}.
func (s *Server) GetVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerificationGet, VerID: reportId}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	// A running report is polled, so it carries the hint that paces the next poll.
	body, _, err := bufferRun(stream)
	if err != nil {
		s.fail(w, err)
		return
	}
	if runStatusOf(body) == string(core.RunRunning) {
		w.Header().Set("Retry-After", retryAfterSeconds)
	}
	s.respondBytes(w, mediaJSON, http.StatusOK, body)
}

// CancelVerification serves POST /system/verifications/{id}/cancel.
func (s *Server) CancelVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerificationCancel, VerID: reportId}
	stream, err := s.core.Handle(r.Context(), req)
	if err != nil {
		s.fail(w, err)
		return
	}
	s.respond(w, stream, http.StatusOK)
}

// DeleteVerification serves DELETE /system/verifications/{id}.
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

// retryAfterSeconds paces a poll of a verification run. A run is long — hours over a large
// closure — so the hint is generous enough that polling costs nothing and short enough
// that a finished run is noticed promptly.
const retryAfterSeconds = "5"

// bufferRun reads a run's report into memory and reports the id it carries. Buffering is
// what lets this route derive a header from the body; it is safe here and only here,
// because a report is small where a query or a blob is not.
func bufferRun(stream core.Stream) ([]byte, string, error) {
	defer func() { _ = stream.Close() }()
	var body bytes.Buffer
	if _, err := stream.WriteTo(&body); err != nil {
		return nil, "", err
	}
	var report struct {
		ID string `json:"id"`
	}
	// A body that does not carry an id is still served; only the header is skipped.
	_ = json.Unmarshal(body.Bytes(), &report)
	return body.Bytes(), report.ID, nil
}

// runStatusOf reads a report's status, empty when the body carries none.
func runStatusOf(body []byte) string {
	var report struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(body, &report)
	return report.Status
}
