// package: core / config
// type:    logic
// job:     read tenant→archive definitions from the flat config map — names, titles, target lifecycle state, and each archive's assembler.Spec; validate Name slugs
// limits:  config-key layout lives here (not the assembler); opens no backends (-> reconcile). Invalid Name slugs fail the load; bad backends surface later as a failed archive.
package core

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"rankedb/adapter/config"
	"rankedb/assembler"
)

// nameRe is the Name slug: lowercase alphanumeric with interior hyphens, no
// leading/trailing hyphen, 1–63 chars. Names are the URL/config/scope identity;
// Titles (free-form) are separate metadata.
var nameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func validName(s string) bool { return nameRe.MatchString(s) }

// archiveDef is one archive resolved from config: its identity (tenant/ra
// names), its display Title, its target lifecycle State, and the build Spec.
type archiveDef struct {
	Tenant string
	RA     string
	Title  string
	Target State
	Spec   assembler.Spec
}

// loadArchives parses every tenants.<t>.archives.<a>.… group from the config
// into archiveDefs. Tenant/archive Names must be valid slugs (else an error —
// malformed config). A missing state defaults to running; an unknown state is
// an error. Backend validity is NOT checked here — that surfaces as a failed
// archive during reconciliation, so one bad archive never blocks the rest.
func loadArchives(entries config.Entries) ([]archiveDef, error) {
	type id struct{ t, a string }
	seen := map[id]bool{}
	var ids []id
	for k := range entries {
		p := strings.Split(k, ".")
		if len(p) >= 5 && p[0] == "tenants" && p[2] == "archives" {
			x := id{p[1], p[3]}
			if !seen[x] {
				seen[x] = true
				ids = append(ids, x)
			}
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].t != ids[j].t {
			return ids[i].t < ids[j].t
		}
		return ids[i].a < ids[j].a
	})

	out := make([]archiveDef, 0, len(ids))
	for _, x := range ids {
		if !validName(x.t) {
			return nil, fmt.Errorf("core: invalid tenant name %q", x.t)
		}
		if !validName(x.a) {
			return nil, fmt.Errorf("core: invalid archive name %q in tenant %q", x.a, x.t)
		}
		prefix := "tenants." + x.t + ".archives." + x.a + "."
		get := func(f string) string { return strings.TrimSpace(entries[prefix+f]) }

		target := State(get("state"))
		if target == "" {
			target = StateRunning // a configured archive runs unless told to stop
		}
		if target != StateRunning && target != StateReadonly && target != StateStopped {
			return nil, fmt.Errorf("core: archive %s/%s: invalid target state %q", x.t, x.a, target)
		}

		spec := assembler.Spec{
			Storage: assembler.StorageSpec{
				Backend: get("storage.backend"),
				Dir:     get("storage.dir"),
				DSN:     get("storage.dsn"),
			},
			Sequencer: assembler.SequencerSpec{
				Backend: get("sequencer.backend"),
				Path:    get("sequencer.path"),
				DSN:     get("sequencer.dsn"),
				Key:     get("sequencer.key"),
			},
		}
		// A DB-backed sequencer keys its head row by the archive's identity unless overridden.
		if (spec.Sequencer.Backend == "postgres" || spec.Sequencer.Backend == "internal") && spec.Sequencer.Key == "" {
			spec.Sequencer.Key = x.t + "/" + x.a
		}

		out = append(out, archiveDef{Tenant: x.t, RA: x.a, Title: get("title"), Target: target, Spec: spec})
	}
	return out, nil
}
