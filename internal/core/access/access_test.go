package access

import "testing"

// TestAllow walks the core-access scenarios: a grant permits a matching right on a
// matching branch, an out-of-glob or un-granted request is denied, the three
// reserved names are privileged, an ordinary glob never reaches them, an unknown
// account is denied, and the cross-branch delete rule falls out of per-branch calls.
func TestAllow(t *testing.T) {
	c, err := New(map[string][]string{
		"webapp":      {"CR foo-*"},
		"provisioner": {"C $branches"},
		"backup":      {"R $universe"},
		"seqop":       {"D $sequencer"},
		"wildcard":    {"R *"},
		"deleter":     {"D a"}, // holds D on branch a, not b
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		name  string
		acct  string
		right Right
		br    string
		want  bool
	}{
		{"contribute within glob", "webapp", Contribute, "foo-bar", true},
		{"read within glob", "webapp", Read, "foo-bar", true},
		{"right not granted", "webapp", Update, "foo-bar", false},
		{"branch outside glob", "webapp", Read, "bar-baz", false},
		{"branch-table admin", "provisioner", Contribute, Branches, true},
		{"privileged universe read", "backup", Read, Universe, true},
		{"privileged sequencer op", "seqop", Delete, Sequencer, true},
		{"ordinary glob misses universe", "wildcard", Read, Universe, false},
		{"ordinary glob misses sequencer", "wildcard", Read, Sequencer, false},
		{"ordinary glob matches branch", "wildcard", Read, "anything", true},
		{"unknown account", "ghost", Read, "foo-bar", false},
		{"delete on held branch", "deleter", Delete, "a", true},
		{"delete on other branch", "deleter", Delete, "b", false},
	}
	for _, tc := range cases {
		p := Principal{Account: tc.acct}
		if got := c.Allow(p, tc.right, tc.br); got != tc.want {
			t.Errorf("%s: Allow(%s, %c, %s) = %v, want %v", tc.name, tc.acct, tc.right, tc.br, got, tc.want)
		}
	}
}

// TestAllowCaveats covers attenuation: a caveat narrows an account's grants to
// their intersection, never widens them.
func TestAllowCaveats(t *testing.T) {
	c, err := New(map[string][]string{"webapp": {"CR foo-*"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	attenuated := Principal{Account: "webapp", Caveats: []Grant{mustGrant(t, "R foo-bar")}}

	cases := []struct {
		name  string
		right Right
		br    string
		want  bool
	}{
		{"caveat permits the narrowed read", Read, "foo-bar", true},
		{"caveat withholds contribute", Contribute, "foo-bar", false},
		{"caveat withholds a sibling branch", Read, "foo-qux", false},
	}
	for _, tc := range cases {
		if got := c.Allow(attenuated, tc.right, tc.br); got != tc.want {
			t.Errorf("%s: = %v, want %v", tc.name, got, tc.want)
		}
	}

	// A caveat cannot grant what the account never held.
	widen := Principal{Account: "webapp", Caveats: []Grant{mustGrant(t, "D foo-bar")}}
	if c.Allow(widen, Delete, "foo-bar") {
		t.Fatal("caveat widened the account beyond its grants")
	}
}

func mustGrant(t *testing.T, spec string) Grant {
	t.Helper()
	g, err := ParseGrant(spec)
	if err != nil {
		t.Fatalf("ParseGrant(%q): %v", spec, err)
	}
	return g
}

// TestParseGrantRejects covers the grant specs ParseGrant must reject at build
// (config-syntax) time — the dropped A right, an unknown reserved name, and branch
// names outside the lowercase/digit/'-' alphabet.
func TestParseGrantRejects(t *testing.T) {
	bad := map[string]string{
		"unknown right letter": "CX foo-*",
		"admin is not a right": "A foo-*",
		"missing glob":         "CR",
		"non-R on universe":    "CR $universe",
		"unknown reserved":     "R $secret",
		"uppercase branch":     "R Foo",
		"underscore branch":    "R foo_bar",
		"slash in branch":      "R foo/bar",
	}
	for name, spec := range bad {
		if _, err := ParseGrant(spec); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	if _, err := New(map[string][]string{"": {"R foo-*"}}); err == nil {
		t.Error("empty account name: want error")
	}
}

// TestParseGrantReserved confirms the three reserved names parse, with $universe
// held to R-only and the other two accepting CRUD (pending their final right-sets).
func TestParseGrantReserved(t *testing.T) {
	for _, spec := range []string{"R $universe", "C $branches", "CRUD $branches", "D $sequencer"} {
		if _, err := ParseGrant(spec); err != nil {
			t.Errorf("ParseGrant(%q): unexpected error %v", spec, err)
		}
	}
}
