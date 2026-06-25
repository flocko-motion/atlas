// package: config / field
// type:    logic
// job:     model a resolved config value with its provenance — Field[T], the layer trail (seed/env/db) winner-first, and the values it shadowed
// limits:  data + rendering only — no storage (-> store.go), no resolution/overlay folding (-> assembler)
//
// Package config holds the server config datatype for the ranke-db control
// plane. The headline type is Field[T]: a resolved config value with the
// provenance of that value built in — the trail of layers (seed, env, db,
// …) that produced it, winner first, including the values it shadowed.
//
// Field is the ranke provenance idea echoed into the db domain: just as a
// ranke claim carries where it came from, a config value carries where it
// came from. It lets the assembler print where any field came from and
// prove a sealed field was never overridden.
package config

import (
	"fmt"
	"strings"
)

// Origin records one layer's contribution to a field: where it came from
// and the value that layer proposed (rendered, for display/audit).
type Origin struct {
	Backend string // "seed", "env", "postgres", "vault", "default"
	Locator string // "./dev.yaml", "RANKE_PORT", "accounts table", …
	Value   string // what this layer proposed, rendered
}

// String renders an origin's location as "backend:locator" (no value).
func (o Origin) String() string {
	if o.Locator == "" {
		return o.Backend
	}
	return o.Backend + ":" + o.Locator
}

// Field is a resolved config value with its provenance built in. Trail is
// in precedence order, winner first; Trail[1:] are the values the winner
// shadowed (the overlay story). Sealed reports that overlay (e.g. an env
// override) was forbidden for this field.
type Field[T any] struct {
	Value  T
	Trail  []Origin
	Sealed bool
}

// From returns the origin the effective Value came from — the zero Origin
// if the field was never set by any layer.
func (f Field[T]) From() Origin {
	if len(f.Trail) == 0 {
		return Origin{}
	}
	return f.Trail[0]
}

// Provenance renders the value and its full origin chain, e.g.:
//
//	9090 ⟵ env:RANKE_PORT (shadowed seed:./dev.yaml=8080)
//	main ⟵ seed:./dev.yaml [sealed]
func (f Field[T]) Provenance() string {
	if len(f.Trail) == 0 {
		return fmt.Sprintf("%v ⟵ (unset)", f.Value)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%v ⟵ %s", f.Value, f.From())
	if f.Sealed {
		b.WriteString(" [sealed]")
	}
	if len(f.Trail) > 1 {
		shadowed := make([]string, 0, len(f.Trail)-1)
		for _, o := range f.Trail[1:] {
			shadowed = append(shadowed, o.String()+"="+o.Value)
		}
		fmt.Fprintf(&b, " (shadowed %s)", strings.Join(shadowed, ", "))
	}
	return b.String()
}
