// package: access / policy
// type:    checker
// job:     decide whether a system account may exercise a CRUD right on a branch
// limits:  pure policy from config; no ports, no ctx; core loops it for delete (-> config, core)
//
// Access is this deployment's policy: accounts holding CRUD grants over branch globs,
// declared in config, never in the graph. The checker answers one (principal, right,
// branch) question, and verifiability never consults it at all.
//
// The rights are CRUD. What was once a fifth, A for admin, is C on $branches: the branch
// table is itself a claim, so creating a branch contributes to it. $branches carries no
// glob, being one server-wide surface, and writing claims into a branch is the separate C on
// that branch. A caveat is a grant of opposite polarity, and the effective permission is
// their intersection.
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

// The reserved branches a grant may target. '$' is illegal in an ordinary name, so no
// glob confers one by accident.
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
//
// Caveats are successive attenuation steps, each a predicate the request must
// still satisfy — not alternatives, or a second narrowing would fail to narrow.
// A bearer can always attenuate further before passing a token on, so a flat
// []Grant can only represent one grant per step: to carry more than one right in
// a single step, list them on one Grant ("RIGHTS glob"), never as siblings.
func (c *Checker) Allow(p Principal, right Right, branch string) bool {
	if !anyAllows(c.accounts[p.Account], right, branch) {
		return false
	}
	return allAllows(p.Caveats, right, branch)
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

// allAllows reports whether every grant in the set allows the action — vacuously
// true with no caveats, since an absent caveat withholds nothing.
func allAllows(grants []Grant, right Right, branch string) bool {
	for _, g := range grants {
		if !g.Allows(right, branch) {
			return false
		}
	}
	return true
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
