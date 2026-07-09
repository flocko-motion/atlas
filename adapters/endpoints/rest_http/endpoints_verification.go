// package: rest_http / transport
// type:    logic
// job:     the verification run endpoints under /system/verification, and report mapping
// limits:  translation only; runs are managed behind core.Handle (-> internal/core)
package rest_http

import (
	"encoding/json"
	"net/http"

	"github.com/flocko-motion/rankedb/api"
	"github.com/flocko-motion/rankedb/internal/core"
)

// StartVerification serves POST /system/verification.
func (s *Server) StartVerification(w http.ResponseWriter, r *http.Request) {
	var cfg api.VerificationConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "malformed verification config")
		return
	}
	vc := coreVerConfig(cfg)
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpStartVerification, VerConfig: &vc}
	if err := s.core.Handle(r.Context(), req); err != nil {
		s.fail(w, err)
		return
	}
	rep, ok := req.Response.(core.VerificationReport)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("Location", "/system/verification/"+rep.ID)
	w.Header().Set("Retry-After", "30")
	writeJSON(w, http.StatusAccepted, apiReport(rep))
}

// ListVerifications serves GET /system/verification.
func (s *Server) ListVerifications(w http.ResponseWriter, r *http.Request) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerifications}
	if err := s.core.Handle(r.Context(), req); err != nil {
		s.fail(w, err)
		return
	}
	reps, ok := req.Response.([]core.VerificationReport)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	out := api.VerificationReportList{Reports: make([]api.VerificationReport, 0, len(reps))}
	for _, rep := range reps {
		out.Reports = append(out.Reports, apiReport(rep))
	}
	writeJSON(w, http.StatusOK, out)
}

// GetVerification serves GET /system/verification/{id}.
func (s *Server) GetVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpVerification, VerID: reportId}
	if err := s.core.Handle(r.Context(), req); err != nil {
		s.fail(w, err)
		return
	}
	rep, ok := req.Response.(core.VerificationReport)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if rep.Status == core.RunRunning {
		w.Header().Set("Retry-After", "30")
	}
	writeJSON(w, http.StatusOK, apiReport(rep))
}

// CancelVerification serves POST /system/verification/{id}/cancel.
func (s *Server) CancelVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpCancelVerification, VerID: reportId}
	if err := s.core.Handle(r.Context(), req); err != nil {
		s.fail(w, err)
		return
	}
	rep, ok := req.Response.(core.VerificationReport)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, apiReport(rep))
}

// DeleteVerification serves DELETE /system/verification/{id}.
func (s *Server) DeleteVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	req := &core.Request{Credential: credentialOf(r.Context()), Op: core.OpDeleteVerification, VerID: reportId}
	if err := s.core.Handle(r.Context(), req); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- report mapping --------------------------------------------------------

func coreVerConfig(c api.VerificationConfig) core.VerificationConfig {
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

func apiVerConfig(c core.VerificationConfig) api.VerificationConfig {
	out := api.VerificationConfig{Closure: c.Closure}
	if c.Layer != "" {
		layer := c.Layer
		out.Layer = &layer
	}
	if c.Depth != "" {
		d := api.VerificationConfigDepth(string(c.Depth))
		out.Depth = &d
	}
	if c.ContentThreshold > 0 {
		ct := int(c.ContentThreshold)
		out.ContentThreshold = &ct
	}
	return out
}

func apiReport(r core.VerificationReport) api.VerificationReport {
	out := api.VerificationReport{
		Id:        r.ID,
		Config:    apiVerConfig(r.Config),
		Head:      idString(r.Head),
		Status:    api.VerificationReportStatus(string(r.Status)),
		StartedAt: r.StartedAt,
		Ok:        r.OK,
	}
	if !r.CompletedAt.IsZero() {
		completed := r.CompletedAt
		out.CompletedAt = &completed
	}
	cc := int(r.ClaimsChecked)
	out.ClaimsChecked = &cc
	br := int(r.BytesRead)
	out.BytesRead = &br
	if len(r.Failures) > 0 {
		fs := make([]api.VerificationFailure, 0, len(r.Failures))
		for _, f := range r.Failures {
			af := api.VerificationFailure{
				Id:    idString(f.ID),
				Mode:  api.VerificationFailureMode(string(f.Mode)),
				Layer: f.Layer,
			}
			if f.Detail != "" {
				detail := f.Detail
				af.Detail = &detail
			}
			fs = append(fs, af)
		}
		out.Failures = &fs
	}
	return out
}
