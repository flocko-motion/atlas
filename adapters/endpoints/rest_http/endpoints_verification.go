// package: rest_http / transport
// type:    logic
// job:     the verification run endpoints under /system/verification, and report mapping
// limits:  translation only; runs are managed behind coreapi.API (-> coreapi)
package rest_http

import (
	"encoding/json"
	"net/http"

	"github.com/flocko-motion/rankedb/adapters/endpoints/coreapi"
	"github.com/flocko-motion/rankedb/api"
)

// StartVerification serves POST /system/verification.
func (s *Server) StartVerification(w http.ResponseWriter, r *http.Request) {
	var cfg api.VerificationConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "malformed verification config")
		return
	}
	rep, err := s.core.StartVerification(r.Context(), subjectOf(r.Context()), coreVerConfig(cfg))
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Location", "/system/verification/"+rep.ID)
	w.Header().Set("Retry-After", "30")
	writeJSON(w, http.StatusAccepted, apiReport(rep))
}

// ListVerifications serves GET /system/verification.
func (s *Server) ListVerifications(w http.ResponseWriter, r *http.Request) {
	reps, err := s.core.Verifications(r.Context(), subjectOf(r.Context()))
	if err != nil {
		s.fail(w, err)
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
	rep, err := s.core.Verification(r.Context(), subjectOf(r.Context()), reportId)
	if err != nil {
		s.fail(w, err)
		return
	}
	if rep.Status == coreapi.RunRunning {
		w.Header().Set("Retry-After", "30")
	}
	writeJSON(w, http.StatusOK, apiReport(rep))
}

// CancelVerification serves POST /system/verification/{id}/cancel.
func (s *Server) CancelVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	rep, err := s.core.CancelVerification(r.Context(), subjectOf(r.Context()), reportId)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apiReport(rep))
}

// DeleteVerification serves DELETE /system/verification/{id}.
func (s *Server) DeleteVerification(w http.ResponseWriter, r *http.Request, reportId string) {
	if err := s.core.DeleteVerification(r.Context(), subjectOf(r.Context()), reportId); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- report mapping --------------------------------------------------------

func coreVerConfig(c api.VerificationConfig) coreapi.VerificationConfig {
	cfg := coreapi.VerificationConfig{Closure: c.Closure}
	if c.Layer != nil {
		cfg.Layer = *c.Layer
	}
	if c.Depth != nil {
		cfg.Depth = coreapi.VerificationDepth(string(*c.Depth))
	}
	if c.ContentThreshold != nil {
		cfg.ContentThreshold = int64(*c.ContentThreshold)
	}
	return cfg
}

func apiVerConfig(c coreapi.VerificationConfig) api.VerificationConfig {
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

func apiReport(r coreapi.VerificationReport) api.VerificationReport {
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
