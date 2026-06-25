// package: api / subjects
// type:    adapter
// job:     REST endpoints for the global subject list and enable/disable — root only
// limits:  thin over the authz engine (root-gated there); capability state only, not identity
package api

import (
	"context"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// SubjectView is a known subject and whether it is disabled (no identity/profile).
type SubjectView struct {
	ID       string `json:"id"`
	Disabled bool   `json:"disabled"`
}

// ListSubjectsEndpoint lists every known subject and its disabled state. Root only.
type ListSubjectsEndpoint struct{}

// Method is GET.
func (ListSubjectsEndpoint) Method() string { return "GET" }

// Path is the global users collection.
func (ListSubjectsEndpoint) Path() string { return "/api/users" }

// Auth requires a valid JWT.
func (ListSubjectsEndpoint) Auth() bool { return true }

// Handle returns the global subject list (root only).
func (ListSubjectsEndpoint) Handle(ctx context.Context, _ ListSubjectsReq) (ListSubjectsResp, error) {
	actor, _ := schemafapi.Subject(ctx)
	subs, err := svc.Authz().Subjects(ctx, actor)
	if err != nil {
		return ListSubjectsResp{}, mapErr(err)
	}
	out := make([]SubjectView, 0, len(subs))
	for _, s := range subs {
		out = append(out, SubjectView{ID: s.ID, Disabled: s.Disabled})
	}
	return ListSubjectsResp{Subjects: out}, nil
}

// ListSubjectsReq has no parameters.
type ListSubjectsReq struct{}

// ListSubjectsResp is the global subject list.
type ListSubjectsResp struct {
	Subjects []SubjectView `json:"subjects"`
}

var _ schemafapi.Endpoint[ListSubjectsReq, ListSubjectsResp] = ListSubjectsEndpoint{}

// SetUserDisabledEndpoint enables or disables a subject globally. Root only.
type SetUserDisabledEndpoint struct{}

// Method is PATCH.
func (SetUserDisabledEndpoint) Method() string { return "PATCH" }

// Path addresses a subject in the global users collection.
func (SetUserDisabledEndpoint) Path() string { return "/api/users/{subject}" }

// Auth requires a valid JWT.
func (SetUserDisabledEndpoint) Auth() bool { return true }

// Handle sets the subject's global disabled flag (root only).
func (SetUserDisabledEndpoint) Handle(ctx context.Context, req SetUserDisabledReq) (SubjectView, error) {
	actor, _ := schemafapi.Subject(ctx)
	if err := svc.Authz().SetDisabled(ctx, actor, req.Subject, req.Disabled); err != nil {
		return SubjectView{}, mapErr(err)
	}
	return SubjectView{ID: req.Subject, Disabled: req.Disabled}, nil
}

// SetUserDisabledReq names the subject (path) and the desired disabled state (body).
type SetUserDisabledReq struct {
	Subject  string `path:"subject"`
	Disabled bool   `json:"disabled"`
}

var _ schemafapi.Endpoint[SetUserDisabledReq, SubjectView] = SetUserDisabledEndpoint{}
