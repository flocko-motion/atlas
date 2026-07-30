// package: access / policy
// type:    checker
// job:     decide whether a system account may exercise a CRUD right on a branch
// limits:  pure policy from config; no ports, no ctx; core loops it for delete (-> config, core)
//
// Package access is the server's access checker. Access is this deployment's
// policy, declared in the configuration as system accounts each holding grants of
// CRUD rights over branches named by globs — never content, never in the graph.
// A request authenticates as a system account through the auth port; the account's
// grants then decide access. The checker is pure and immutable: built once from
// config (no runtime mutation), it answers a single (principal, right, branch)
// question. Composite rules live above it — the "D on every holding branch" purge
// rule is core calling Allow once per branch — and verifiability never consults it:
// a third party verifies an archive with no grant from this server.
//
// Rights are CRUD. The paper's fifth right, A (admin: create or hide branches), is
// not a separate right: the branch table is itself a claim, so creating or hiding
// a branch is contributing a new branch-table revision — a C aimed at the branch
// table. That target will be a reserved branch name (planned: "$branches",
// analogous to $universe); it is not yet wired here pending the model's sign-off,
// and the reserved-name matching below already generalises to it.
//
// Grants and caveats share one algebra and one type (Grant). A grant is a positive
// capability an account holds; a caveat is a bearer-supplied restriction that rides
// in an attenuated token (a macaroon) and NARROWS what the account may do. The
// effective permission is their intersection: the account's grants must allow the
// action AND, if the principal carries caveats, the caveats must allow it too.
package access

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Right is one CRUD access right.
type Right byte

const (
	Contribute Right = 'C' // contribute claims to the branch (creating it if new)
	Read       Right = 'R' // read the branch
	Update     Right = 'U' // overlay an existing claim with a newer version
	Delete     Right = 'D' // delete claims (needs D on every branch that holds the claim)
)

// The reserved branches a grant may target: $universe (by-head-id read), $archive (the
// whole archive, spec §RankeQL), $branches (the branch table), $sequencer (BTH
// advancement). '$' is illegal in ordinary names, so no glob confers them by accident.
const (
	Universe  = "$universe"
	Archive   = "$archive"
	Sequencer = "$sequencer"
	Branches  = "$branches"
)

var reserved = map[string]bool{Universe: true, Archive: true, Sequencer: true, Branches: true}

// branchGlob is an ordinary branch glob: lowercase, digits, '-', and the '*'/'?'
// wildcards — branch names are deliberately a small alphabet.
var branchGlob = regexp.MustCompile(`^[a-z0-9*?-]+$`)

// rightset is a bitmask over the CRUD rights.
type rightset uint8

// bit maps a right to its bit, reporting whether it is a valid CRUD letter.
func bit(r Right) (rightset, bool) {
	switch r {
	case Contribute:
		return 1 << 0, true
	case Read:
		return 1 << 1, true
	case Update:
		return 1 << 2, true
	case Delete:
		return 1 << 3, true
	}
	return 0, false
}

// Grant confers rights over the branches matching a glob — the unit of both an account
// grant and a token caveat, the Checker applying the polarity.
type Grant struct {
	rights rightset
	glob   string
}

// ParseGrant parses one "RIGHTS glob" spec ("CR foo-*", "R $universe"), rejecting
// unknown letters, malformed globs, and non-R rights on $universe. Caveats reuse it.
func ParseGrant(spec string) (Grant, error) {
	fields := strings.Fields(spec)
	if len(fields) != 2 {
		return Grant{}, fmt.Errorf("grant %q: want \"RIGHTS glob\"", spec)
	}
	letters, glob := fields[0], fields[1]

	var rs rightset
	for _, ch := range letters {
		b, ok := bit(Right(ch))
		if !ok {
			return Grant{}, fmt.Errorf("grant %q: unknown right %q", spec, string(ch))
		}
		rs |= b
	}

	if strings.HasPrefix(glob, "$") {
		if !reserved[glob] {
			return Grant{}, fmt.Errorf("grant %q: unknown reserved branch %q", spec, glob)
		}
		// $universe is read-only (paper); the other reserved names accept any CRUD
		// pending the access model's sign-off.
		if glob == Universe {
			if readOnly, _ := bit(Read); rs != readOnly {
				return Grant{}, fmt.Errorf("grant %q: only R applies to %s", spec, Universe)
			}
		}
	} else if !branchGlob.MatchString(glob) {
		return Grant{}, fmt.Errorf("grant %q: branch %q must be lowercase letters, digits and '-' (with * or ? wildcards)", spec, glob)
	}

	return Grant{rights: rs, glob: glob}, nil
}

// Allows reports whether this grant carries right and its glob matches branch.
func (g Grant) Allows(right Right, branch string) bool {
	b, ok := bit(right)
	if !ok || g.rights&b == 0 {
		return false
	}
	return matchBranch(g.glob, branch)
}

// Principal is the identity a request acts as: the account the credential resolved to,
// plus any caveats attenuating its grants (empty = none).
type Principal struct {
	Account string
	Caveats []Grant
}

// Checker answers access requests against a fixed set of accounts and grants.
type Checker struct {
	accounts map[string][]Grant
}

// New builds a checker from the configured accounts, each mapping to compact grant
// specs. It validates every grant offline and fails on the first malformed one.
func New(accounts map[string][]string) (*Checker, error) {
	c := &Checker{accounts: make(map[string][]Grant, len(accounts))}
	for name, specs := range accounts {
		if name == "" {
			return nil, fmt.Errorf("access: empty account name")
		}
		for _, spec := range specs {
			g, err := ParseGrant(spec)
			if err != nil {
				return nil, fmt.Errorf("access: account %q: %w", name, err)
			}
			c.accounts[name] = append(c.accounts[name], g)
		}
	}
	return c, nil
}

// Allow reports whether the principal may exercise right on branch: the account's
// grants and any caveats must both allow it. Unknown or ungranted is denied.
func (c *Checker) Allow(p Principal, right Right, branch string) bool {
	if !anyAllows(c.accounts[p.Account], right, branch) {
		return false
	}
	if len(p.Caveats) == 0 {
		return true
	}
	return anyAllows(p.Caveats, right, branch)
}

// anyAllows reports whether any grant in the set allows the action.
func anyAllows(grants []Grant, right Right, branch string) bool {
	for _, g := range grants {
		if g.Allows(right, branch) {
			return true
		}
	}
	return false
}

// matchBranch matches a grant glob against a branch. A "$..." name needs an exact
// literal grant, so "*" never reaches it; ordinary branches match by shell glob.
func matchBranch(glob, branch string) bool {
	if strings.HasPrefix(branch, "$") {
		return glob == branch
	}
	ok, err := path.Match(glob, branch)
	return err == nil && ok
}
